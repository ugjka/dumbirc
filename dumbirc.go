package dumbirc

import (
	"crypto/tls"
	"errors"

	irc "github.com/sorcix/irc"
)

//Connection Settings
type Connection struct {
	Nick      string
	User      string
	Server    string
	TLS       bool
	conn      *irc.Conn
	callbacks map[Event]func(Message)
	Errchan   chan error
}

// Event codes
type Event string

//Message event
type Message irc.Message

// Map event codes
const (
	PRIVMSG   Event = irc.PRIVMSG
	PING      Event = irc.PING
	WELCOME   Event = irc.RPL_WELCOME
	NICKTAKEN Event = irc.ERR_NICKNAMEINUSE
)

//New creates a new bot
func New(nick string, user string, server string, tls bool) *Connection {
	return &Connection{
		nick,
		user,
		server,
		tls,
		&irc.Conn{},
		make(map[Event]func(Message)),
		make(chan error),
	}
}

//AddCallback Adds callback to an event
func (c *Connection) AddCallback(event Event, callback func(Message)) {
	c.callbacks[event] = callback
}

// Run Callbacks
func (c *Connection) runCallbacks(msg Message) {
	for i, v := range c.callbacks {
		if i == Event(msg.Command) {
			v(msg)
		}
	}
}

// Join channels
func (c *Connection) Join(ch []string) {
	for _, v := range ch {
		_, err := c.conn.Write([]byte(irc.JOIN + " " + v))
		if err != nil {
			c.Errchan <- err
		}
	}
}

//Pong sends pong
func (c *Connection) Pong() {
	_, err := c.conn.Write([]byte(irc.PONG))
	if err != nil {
		c.Errchan <- err
	}
}

//Ping sends ping
func (c *Connection) Ping() {
	_, err := c.conn.Write([]byte(irc.PING + " " + c.Server))
	if err != nil {
		c.Disconnect()
	}
}

//PrivMsg sends privmessage
func (c *Connection) PrivMsg(dest string, msg string) {
	_, err := c.conn.Write([]byte(irc.PRIVMSG + " " + dest + " :" + msg))
	if err != nil {
		c.Errchan <- err
	}
}

//Disconnect disconnects from irc
func (c *Connection) Disconnect() {
	c.conn.Close()
	c.Errchan <- errors.New("Disconnected!")
}

//NewNick Changes nick
func (c *Connection) NewNick(n string) {
	_, err := c.conn.Write([]byte(irc.NICK + " " + n))
	if err != nil {
		c.Errchan <- err
	}
}

//Reply replies to a message
func (c *Connection) Reply(msg Message, reply string) {
	if msg.Params[0] == c.Nick {
		c.PrivMsg(msg.Name, reply)
	} else {
		c.PrivMsg(msg.Params[0], reply)
	}
}

// Start the bot
func (c *Connection) Start() {
	if c.TLS {
		tls, err := tls.Dial("tcp", c.Server, &tls.Config{})
		if err != nil {
			c.Errchan <- err
			return
		}
		c.conn = irc.NewConn(tls)
	} else {
		var err error
		c.conn, err = irc.Dial(c.Server)
		if err != nil {
			c.Errchan <- err
			return
		}
	}
	_, err := c.conn.Write([]byte("USER " + c.Nick + " +iw * :" + c.User))
	if err != nil {
		c.Errchan <- err
		return
	}
	_, err = c.conn.Write([]byte(irc.NICK + " " + c.Nick))
	if err != nil {
		c.Errchan <- err
		return
	}
	go func(c *Connection) {
		for {
			msg, err := c.conn.Decode()
			if err != nil {
				c.Errchan <- err
				return
			}
			go c.runCallbacks(Message(*msg))
		}
	}(c)

}
