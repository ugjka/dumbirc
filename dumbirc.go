package dumbirc

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	irc "github.com/sorcix/irc"
)

//Map event codes
const (
	PRIVMSG   = irc.PRIVMSG
	PING      = irc.PING
	PONG      = irc.PONG
	WELCOME   = irc.RPL_WELCOME
	NICKTAKEN = irc.ERR_NICKNAMEINUSE
	JOIN      = irc.JOIN
	KICK      = irc.KICK
	NOTICE    = irc.NOTICE
	//Useful if you wanna check for activity
	ANYMESSAGE = "ANY"
)

//Connection Settings
type Connection struct {
	Nick      string
	User      string
	RealN     string
	Server    string
	TLS       bool
	Password  string
	Throttle  time.Duration
	connected bool
	//Fake Connected status
	DebugFakeConn bool
	conn          *irc.Conn
	callbacks     map[string][]func(*Message)
	triggers      []Trigger
	Log           *log.Logger
	Debug         *log.Logger
	Errchan       chan error
	Send          chan []byte
	incomingID    int
	incoming      map[int]chan *Message
	incomingMu    sync.RWMutex
	sync.RWMutex
}

//New creates a new irc object
func New(nick string, user string, server string, tls bool) *Connection {
	return &Connection{
		Nick:       nick,
		User:       user,
		Server:     server,
		TLS:        tls,
		Throttle:   time.Millisecond * 500,
		conn:       &irc.Conn{},
		callbacks:  make(map[string][]func(*Message)),
		triggers:   make([]Trigger, 0),
		Log:        log.New(&devNull{}, "", log.Ldate|log.Ltime),
		Debug:      log.New(&devNull{}, "debug", log.Ltime),
		Errchan:    make(chan error),
		RWMutex:    sync.RWMutex{},
		incoming:   make(map[int]chan *Message),
		incomingMu: sync.RWMutex{},
	}
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

type devNull struct {
}

func (d *devNull) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// WaitFor will block until a message matching the given filter is received
func (c *Connection) WaitFor(filter func(*Message) bool) {
	if !c.IsConnected() {
		return
	}
	c.incomingMu.Lock()
	c.incomingID++
	tmpID := c.incomingID
	c.incoming[tmpID] = make(chan *Message)
	c.incomingMu.Unlock()
	defer func() {
		c.incomingMu.Lock()
		close(c.incoming[tmpID])
		delete(c.incoming, tmpID)
		c.incomingMu.Unlock()
	}()
	for mes := range c.incoming[tmpID] {
		if filter(mes) {
			return
		}
	}
	return
}

//SetThrottle sets post delay
func (c *Connection) SetThrottle(d time.Duration) {
	c.Throttle = d
}

//SetPassword sets the irc password
func (c *Connection) SetPassword(pass string) {
	c.Password = pass
}

//SetLogOutput sets where to log
func (c *Connection) SetLogOutput(w io.Writer) {
	c.Log.SetOutput(w)
}

//EnableDebug enables irc message debugging
func (c *Connection) EnableDebug(w io.Writer) {
	c.Debug.SetOutput(w)
}

//IsConnected returns connection status
func (c *Connection) IsConnected() bool {
	c.RLock()
	defer c.RUnlock()
	return c.connected
}

//AddCallback Adds callback to an event
func (c *Connection) AddCallback(event string, callback func(*Message)) {
	c.callbacks[event] = append(c.callbacks[event], callback)
}

//Trigger scheme
type Trigger struct {
	Condition func(*Message) bool
	Response  func(*Message)
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
		for _, v := range v {
			v(msg)
		}
	}
	if v, ok := c.callbacks[msg.Command]; ok {
		for _, v := range v {
			v(msg)
		}
	}
}

//Join channels
func (c *Connection) Join(ch []string) {
	for _, v := range ch {
		if !c.IsConnected() {
			return
		}
		c.Send <- []byte(irc.JOIN + " " + v)
	}
}

//Pong sends pong
func (c *Connection) Pong() {
	if !c.IsConnected() {
		return
	}
	c.Send <- []byte(irc.PONG)
}

//Ping sends ping
func (c *Connection) Ping() {
	if !c.IsConnected() {
		return
	}
	c.Send <- []byte(irc.PING + " " + c.Server)
}

//Cmd sends command
func (c *Connection) Cmd(cmd string) {
	if !c.IsConnected() {
		return
	}
	c.Send <- []byte(cmd)
}

//Msg sends privmessage
func (c *Connection) Msg(dest string, msg string) {
	if !c.IsConnected() {
		return
	}
	c.Send <- []byte(irc.PRIVMSG + " " + dest + " :" + msg)
}

//MsgBulk sends message to many
func (c *Connection) MsgBulk(list []string, msg string) {
	for _, k := range list {
		if !c.IsConnected() {
			return
		}
		c.Msg(k, msg)
	}
}

//NewNick Changes nick
func (c *Connection) NewNick(n string) {
	if !c.IsConnected() {
		return
	}
	c.Send <- []byte(irc.NICK + " " + n)
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

//Disconnect disconnects from irc
func (c *Connection) Disconnect() {
	c.Lock()
	defer c.Unlock()
	if c.connected {
		c.connected = false
		c.conn.Close()
		c.incomingMu.Lock()
		for k := range c.incoming {
			close(c.incoming[k])
			delete(c.incoming, k)
		}
		c.incomingMu.Unlock()
	Loop:
		for {
			select {
			case <-c.Send:
			default:
				close(c.Send)
				break Loop
			}
		}
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

//LogNotices logs notice messages
func (c *Connection) LogNotices() {
	c.AddCallback(NOTICE, func(m *Message) {
		c.Log.Printf("NOTICE %s %s", m.Params[0], m.Trailing)
	})
}

//HandleNickTaken changes nick when nick taken
func (c *Connection) HandleNickTaken() {
	c.AddCallback(NICKTAKEN, func(msg *Message) {
		if c.Password != "" {
			rand.Seed(time.Now().UnixNano())
			tmp := ""
			for i := 0; i < 4; i++ {
				tmp += fmt.Sprintf("%d", rand.Intn(9))
			}
			if len(c.Nick) > 12 {
				c.NewNick(c.Nick[:12] + tmp)
			} else {
				c.NewNick(c.Nick + tmp)
			}
			c.Log.Println("nick taken, GHOSTING " + c.Nick)
			c.Msg("NickServ", "GHOST "+c.Nick+" "+c.Password)
			c.WaitFor(func(m *Message) bool {
				return m.Command == NOTICE &&
					strings.Contains(m.Trailing, "has been ghosted")
			})
			c.NewNick(c.Nick)
			c.Msg("NickServ", "identify "+c.Nick+" "+c.Password)
			c.WaitFor(func(m *Message) bool {
				return m.Command == NOTICE &&
					strings.Contains(m.Trailing, "You are now identified")
			})
			return
		}
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
	c.Lock()
	c.Send = make(chan []byte)
	c.connected = true
	c.Unlock()
	if c.Password != "" {
		out := []byte("PASS " + c.Password)
		c.Debug.Printf("→ %s", out)
		_, err := c.conn.Write(out)
		if err != nil {
			c.Disconnect()
			c.Errchan <- err
			return
		}
	}
	if c.RealN == "" {
		c.RealN = c.User
	}
	out := []byte("USER " + c.User + " +iw * :" + c.RealN)
	c.Debug.Printf("→ %s", out)
	_, err := c.conn.Write(out)
	if err != nil {
		c.Disconnect()
		c.Errchan <- err
		return
	}
	out = []byte(irc.NICK + " " + c.Nick)
	c.Debug.Printf("→ %s", out)
	_, err = c.conn.Write(out)
	if err != nil {
		c.Disconnect()
		c.Errchan <- err
		return
	}
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
			c.Debug.Printf("← %s", raw)
			msg := &Message{raw}
			c.incomingMu.RLock()
			for k := range c.incoming {
				c.incoming[k] <- msg
			}
			c.incomingMu.RUnlock()
			go c.RunCallbacks(msg)
			go c.RunTriggers(msg)
		}
	}(c)
	go func(c *Connection) {
		for {
			if !c.IsConnected() {
				return
			}
			v, ok := <-c.Send
			if !ok {
				return
			}
			c.Debug.Printf("→ %s", v)
			_, err := c.conn.Write(v)
			if err != nil {
				c.Disconnect()
				c.Errchan <- err
				return
			}
			time.Sleep(c.Throttle)
		}
	}(c)

}
