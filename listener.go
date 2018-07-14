package hanu

// Handler is the interface for the handler function
type ListenerHandler func(ListenerConversationInterface)

// CommandInterface defines a command interface
type ListenerInterface interface {
	Get() string
	Handle(conv ConversationInterface)
}

// Command a command
type Listener struct {
	regex		string
	handler		ListenerHandler
}

// SetHandler sets the handler
func (c *Listener) SetHandler(handler ListenerHandler) {
	c.handler = handler
}

// Handle calls the Listener's handler
func (c Listener) Handle(conv ConversationInterface) {
	go c.handler(conv)
}

// Get returns the regex
func (c Listener) Get() string {
	return c.regex
}

// Set defines the regex
func (c *Listener) Set(cmd string) {
	c.regex = cmd
}

// NewListener creates a new Listener
func NewListener(regex string, handler ListenerHandler) Listener {
	cmd := Listener{}
	cmd.Set(regex)
	cmd.SetHandler(handler)

	return cmd
}
