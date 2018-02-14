package dumbirc

import (
	"crypto/tls"
	"sync"

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
	connected bool
	sync.RWMutex
	triggers []Trigger
}

//Trigger scheme
type Trigger struct {
	Condition func(Message) bool
	Response  func(Message)
}

//Event codes
type Event string

//Message event
type Message *irc.Message

//Map event codes
const (
	PRIVMSG   Event = irc.PRIVMSG
	PING      Event = irc.PING
	PONG      Event = irc.PONG
	WELCOME   Event = irc.RPL_WELCOME
	NICKTAKEN Event = irc.ERR_NICKNAMEINUSE
	//Useful if you wanna check for activity
	ANYMESSAGE Event = "ANY"
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
		false,
		sync.RWMutex{},
		make([]Trigger, 0),
	}
}

//IsConnected returns connection status
func (c *Connection) IsConnected() bool {
	c.RLock()
	defer c.RUnlock()
	return c.connected
}

//AddCallback Adds callback to an event
func (c *Connection) AddCallback(event Event, callback func(Message)) {
	c.callbacks[event] = callback
}

//AddTrigger adds triggers
func (c *Connection) AddTrigger(t Trigger) {
	c.triggers = append(c.triggers, t)
}

//Run Triggers
func (c *Connection) runTriggers(msg Message) {
	for _, v := range c.triggers {
		if v.Condition(msg) {
			v.Response(msg)
		}
	}
}

//Run Callbacks
func (c *Connection) runCallbacks(msg Message) {
	if v, ok := c.callbacks[ANYMESSAGE]; ok {
		v(msg)
	}
	if v, ok := c.callbacks[Event(msg.Command)]; ok {
		v(msg)
	}
}

//Join channels
func (c *Connection) Join(ch []string) {
	for _, v := range ch {
		if !c.IsConnected() {
			return
		}
		_, err := c.conn.Write([]byte(irc.JOIN + " " + v))
		if err != nil {
			c.Disconnect()
			c.Errchan <- err
		}
	}
}

//Pong sends pong
func (c *Connection) Pong() {
	if !c.IsConnected() {
		return
	}
	_, err := c.conn.Write([]byte(irc.PONG))
	if err != nil {
		c.Disconnect()
		c.Errchan <- err
	}
}

//Ping sends ping
func (c *Connection) Ping() {
	if !c.IsConnected() {
		return
	}
	_, err := c.conn.Write([]byte(irc.PING + " " + c.Server))
	if err != nil {
		c.Disconnect()
		c.Errchan <- err
	}
}

//PrivMsg sends privmessage
func (c *Connection) PrivMsg(dest string, msg string) {
	if !c.IsConnected() {
		return
	}
	_, err := c.conn.Write([]byte(irc.PRIVMSG + " " + dest + " :" + msg))
	if err != nil {
		c.Disconnect()
		c.Errchan <- err
	}
}

//PrivMsgBulk sends message to many
func (c *Connection) PrivMsgBulk(list []string, msg string) {
	for _, k := range list {
		c.PrivMsg(k, msg)
	}
}

//Disconnect disconnects from irc
func (c *Connection) Disconnect() {
	if c.IsConnected() {
		c.conn.Close()
	}
	c.Lock()
	c.connected = false
	c.Unlock()
}

//NewNick Changes nick
func (c *Connection) NewNick(n string) {
	if !c.IsConnected() {
		return
	}
	_, err := c.conn.Write([]byte(irc.NICK + " " + n))
	if err != nil {
		c.Disconnect()
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
	if c.IsConnected() {
		return
	}
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
		c.Disconnect()
		c.Errchan <- err
		return
	}
	_, err = c.conn.Write([]byte(irc.NICK + " " + c.Nick))
	if err != nil {
		c.Disconnect()
		c.Errchan <- err
		return
	}
	c.Lock()
	c.connected = true
	c.Unlock()
	go func(c *Connection) {
		for {
			if !c.IsConnected() {
				return
			}
			msg, err := c.conn.Decode()
			if err != nil {
				c.Disconnect()
				c.Errchan <- err
				return
			}
			go c.runCallbacks(Message(msg))
			go c.runTriggers(Message(msg))
		}
	}(c)

}
