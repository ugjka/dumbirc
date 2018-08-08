package dumbirc

import (
	"crypto/tls"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	irc "github.com/sorcix/irc"
)

//TODO
// add debug logging

//Connection Settings
type Connection struct {
	Nick      string
	User      string
	Server    string
	TLS       bool
	conn      *irc.Conn
	callbacks map[string]func(*Message)
	Errchan   chan error
	connected bool
	sync.RWMutex
	triggers []Trigger
	//Fake Connected status
	DebugFakeConn bool
	Log           *log.Logger
}

//Trigger scheme
type Trigger struct {
	Condition func(*Message) bool
	Response  func(*Message)
}

//Message event
type Message struct {
	*irc.Message
}

//NewMessage returns an empty message
func NewMessage() *Message {
	msg := new(irc.Message)
	msg.Prefix = new(irc.Prefix)
	msg.Params = make([]string, 0)
	return &Message{msg}
}

//Map event codes
const (
	PRIVMSG   = irc.PRIVMSG
	PING      = irc.PING
	PONG      = irc.PONG
	WELCOME   = irc.RPL_WELCOME
	NICKTAKEN = irc.ERR_NICKNAMEINUSE
	JOIN      = irc.JOIN
	//Useful if you wanna check for activity
	ANYMESSAGE = "ANY"
)

//New creates a new irc object
func New(nick string, user string, server string, tls bool) *Connection {
	return &Connection{
		nick,
		user,
		server,
		tls,
		&irc.Conn{},
		make(map[string]func(*Message)),
		make(chan error),
		false,
		sync.RWMutex{},
		make([]Trigger, 0),
		false,
		log.New(&devnull{}, "", log.Ldate|log.Ltime),
	}
}

type devnull struct {
}

func (d *devnull) Write(p []byte) (n int, err error) {
	return len(p), nil
}

//SetLogOutput sets where to log
func (c *Connection) SetLogOutput(w io.Writer) {
	c.Log.SetOutput(w)
}

//IsConnected returns connection status
func (c *Connection) IsConnected() bool {
	c.RLock()
	defer c.RUnlock()
	return c.connected
}

//AddCallback Adds callback to an event
func (c *Connection) AddCallback(event string, callback func(*Message)) {
	c.callbacks[event] = callback
}

//AddTrigger adds triggers
func (c *Connection) AddTrigger(t Trigger) {
	c.triggers = append(c.triggers, t)
}

//RunTriggers ...
func (c *Connection) RunTriggers(msg *Message) {
	for _, v := range c.triggers {
		if v.Condition(msg) {
			v.Response(msg)
		}
	}
}

//RunCallbacks ...
func (c *Connection) RunCallbacks(msg *Message) {
	if v, ok := c.callbacks[ANYMESSAGE]; ok {
		v(msg)
	}
	if v, ok := c.callbacks[msg.Command]; ok {
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

//Msg sends privmessage
func (c *Connection) Msg(dest string, msg string) {
	if !c.IsConnected() {
		return
	}
	_, err := c.conn.Write([]byte(irc.PRIVMSG + " " + dest + " :" + msg))
	if err != nil {
		c.Disconnect()
		c.Errchan <- err
	}
}

//MsgBulk sends message to many
func (c *Connection) MsgBulk(list []string, msg string) {
	if !c.IsConnected() {
		return
	}
	for _, k := range list {
		c.Msg(k, msg)
	}
}

//Disconnect disconnects from irc
func (c *Connection) Disconnect() {
	c.Lock()
	defer c.Unlock()
	if c.connected == true {
		c.conn.Close()
	}
	c.connected = false
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
func (c *Connection) Reply(msg *Message, reply string) {
	if !c.IsConnected() {
		return
	}
	if msg.Params[0] == c.Nick {
		c.Msg(msg.Name, reply)
	} else {
		c.Msg(msg.Params[0], reply)
	}
}

func changeNick(n string) string {
	if len(n) < 16 {
		n += "_"
		return n
	}
	n = strings.TrimRight(n, "_")
	if len(n) > 12 {
		n = n[:12] + "_"
	}
	return n
}

//HandleNickTaken changes nick when nick taken
func (c *Connection) HandleNickTaken() {
	c.AddCallback(NICKTAKEN, func(msg *Message) {
		c.Log.Printf("nick %s taken, changing nick", c.Nick)
		c.Nick = changeNick(c.Nick)
		c.NewNick(c.Nick)
	})
}

func pingpong(c chan bool) {
	select {
	case c <- true:
	default:
		return
	}
}

//HandlePingPong replies to and sends pings
func (c *Connection) HandlePingPong() {
	c.AddCallback(PING, func(msg *Message) {
		c.Log.Println("got ping sending pong")
		c.Pong()
	})
	pp := make(chan bool, 1)
	c.AddCallback(ANYMESSAGE, func(msg *Message) {
		pingpong(pp)
	})
	pingTick := time.NewTicker(time.Minute * 1)
	go func(tick *time.Ticker) {
		for range tick.C {
			select {
			case <-pp:
				c.Ping()
			default:
				c.Log.Println("got no pong")
			}
		}
	}(pingTick)
}

//HandleJoin joins channels on welcome
func (c *Connection) HandleJoin(chans []string) {
	c.AddCallback(WELCOME, func(msg *Message) {
		c.Log.Println("joining channels")
		c.Join(chans)
	})
}

// Start the bot
func (c *Connection) Start() {
	if c.IsConnected() || c.DebugFakeConn {
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
			raw, err := c.conn.Decode()
			if err != nil {
				c.Disconnect()
				c.Errchan <- err
				return
			}
			msg := &Message{raw}
			go c.RunCallbacks(msg)
			go c.RunTriggers(msg)
		}
	}(c)

}
