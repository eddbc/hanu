package hanu

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/websocket"
	"regexp"
	"log"
)

type handshakeResponseSelf struct {
	ID string `json:"id"`
}

type handshakeResponse struct {
	Ok    bool                  `json:"ok"`
	Error string                `json:"error"`
	URL   string                `json:"url"`
	Self  handshakeResponseSelf `json:"self"`
}

// Bot is the main object
type Bot struct {
	Socket		*websocket.Conn
	Token		string
	ID			string
	Commands	[]CommandInterface
	Listeners	[]ListenerInterface
	Prefix		string
}

// New creates a new bot
func New(token string) (*Bot, error) {
	bot := Bot{
		Token: token,
		Prefix: "!",
	}

	return bot.Handshake()
}

func (b *Bot) SetPrefix(prefix string) {
	b.Prefix = prefix
}

// Handshake connects to the Slack API to get a socket connection
func (b *Bot) Handshake() (*Bot, error) {
	// Check for HTTP error on connection
	res, err := http.Get(fmt.Sprintf("https://slack.com/api/rtm.start?token=%s", b.Token))
	if err != nil {
		return nil, errors.New("Failed to connect to Slack RTM API")
	}

	// Check for HTTP status code
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Failed with HTTP Code: %d", res.StatusCode)
	}

	// Read response body
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("Failed to read body from response")
	}

	// Parse response
	var response handshakeResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal JSON: %s", body)
	}

	// Check for Slack error
	if !response.Ok {
		return nil, errors.New(response.Error)
	}

	// Assign Slack user ID
	b.ID = response.Self.ID

	// Connect to websocket
	b.Socket, err = websocket.Dial(response.URL, "", "https://api.slack.com/")
	if err != nil {
		return nil, errors.New("Failed to connect to Websocket")
	}

	return b, nil
}

// Process incoming message
func (b *Bot) process(message Message) {
	if message.IsBotMessage(b.Prefix, b.ID) {
		// Strip @BotName from public message
		message.StripMention(b.ID)
		// Strip Slack's link markup
		message.StripLinkMarkup()
		// Strip the defined command prefix
		message.StripPrefix(b.Prefix)

		// Check if the message requests the auto-generated help command list
		// or if we need to search for a command matching the request
		if message.IsHelpRequest() {
			b.sendHelp(message)
			return
		} else {
			if b.searchCommand(message) {
				return
			}
		}
	}

	// message is not a command, so pass it on to listeners
	b.searchListener(message)
}

// Search for a command matching the message
func (b *Bot) searchCommand(msg Message) bool {
	var cmd CommandInterface
	c := false

	for i := 0; i < len(b.Commands); i++ {
		cmd = b.Commands[i]

		match, err := cmd.Get().Match(msg.Text())
		if err == nil {
			cmd.Handle(NewConversation(match, msg, b.Socket))
			c = true
		}
	}

	return c
}

func (b *Bot) searchListener(msg Message) bool {
	var lst ListenerInterface
	l := false

	for i := 0; i < len(b.Listeners) ;i++  {
		lst = b.Listeners[i]

		r, _ := regexp.Compile(lst.Get())

		if r.MatchString(msg.Message) {
			log.Printf("Listener Matched: %v\n", msg.Message)
			lst.Handle(NewListenerConversation(msg, b.Socket))
			l = true
		}
	}

	return l
}

// Send the response for a help request
func (b *Bot) sendHelp(msg Message) {
	var cmd CommandInterface
	help := "I can support you with those features:\n\n"

	for i := 0; i < len(b.Commands); i++ {
		cmd = b.Commands[i]

		help = help + "`" + cmd.Get().Text() + "`"
		if cmd.Description() != "" {
			help = help + " *–* " + cmd.Description()
		}

		help = help + "\n"
	}

	if !msg.IsDirectMessage() {
		help = "<@" + msg.User() + ">: " + help
	}

	msg.SetText(help)
	websocket.JSON.Send(b.Socket, msg)
}

// Listen for message on socket
func (b *Bot) Listen() {
	var msg Message

	for {
		if websocket.JSON.Receive(b.Socket, &msg) == nil {
			go b.process(msg)

			// Clean up message after processign it
			msg = Message{}
		}
	}
}

// Command adds a new command with custom handler
func (b *Bot) Command(cmd string, handler Handler) {
	b.Commands = append(b.Commands, NewCommand(cmd, "", handler))
}

// Hear adds a new listener with a custom handler
func (b *Bot) Hear(regex string, handler ListenerHandler) {
	b.Listeners = append(b.Listeners, NewListener(regex, handler))
}

// RegisterCommand registers a Command
func (b *Bot) RegisterCommand(cmd CommandInterface) {
	b.Commands = append(b.Commands, cmd)
}

// RegisterListener registers a Listener
func (b *Bot) RegisterListener(lst ListenerInterface) {
	b.Listeners = append(b.Listeners, lst)
}

