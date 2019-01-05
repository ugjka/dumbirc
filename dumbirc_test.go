package dumbirc

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	irc "gopkg.in/sorcix/irc.v2"
)

const nick = "ugjka"
const channel = "#ugjka"
const password = "hunter123"

func TestHandleJoin(t *testing.T) {
	tt := []*irc.Message{
		irc.ParseMessage(fmt.Sprintf("USER %s +iw * %s", nick, nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
		irc.ParseMessage(fmt.Sprintf("JOIN %s", channel)),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.SetThrottle(0)
	bot.HandleJoin([]string{channel})
	bot.Start()
	srv.encode(fmt.Sprintf(":example.com 001 %s :Welcome Internet Relay Chat Network", nick))
	for _, tc := range tt {
		msg, err := srv.decode()
		if err != nil {
			t.Errorf("decoding a message failed: %v", err)
			t.FailNow()
		}
		if !reflect.DeepEqual(tc, msg) {
			t.Errorf("expected %v, got %v", tc, msg)
		}
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()
}

func TestHandleJoinPassword(t *testing.T) {
	tt := []*irc.Message{
		irc.ParseMessage(fmt.Sprintf("PASS %s", password)),
		irc.ParseMessage(fmt.Sprintf("USER %s +iw * %s", nick, nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
		irc.ParseMessage(fmt.Sprintf("JOIN %s", channel)),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.SetThrottle(0)
	bot.testing = true
	bot.HandleJoin([]string{channel})
	bot.SetPassword(password)
	bot.Start()
	srv.encode(fmt.Sprintf(":example.com 001 %s :Welcome Internet Relay Chat Network", nick))
	for i, tc := range tt {
		msg, err := srv.decode()
		if err != nil {
			t.Errorf("decoding a message failed: %v", err)
			t.FailNow()
		}
		if !reflect.DeepEqual(tc, msg) {
			t.Errorf("expected %v, got %v", tc, msg)
		}
		if i == 2 {
			<-bot.testchan
			srv.encode(fmt.Sprintf(":connect!admin@test.com NOTICE %s :You are now identified for", nick))
			<-bot.testchan
		}
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()
}

func TestHandleJoinPasswordTimeout(t *testing.T) {
	tt := []*irc.Message{
		irc.ParseMessage(fmt.Sprintf("PASS %s", password)),
		irc.ParseMessage(fmt.Sprintf("USER %s +iw * %s", nick, nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
		irc.ParseMessage(fmt.Sprintf("JOIN %s", channel)),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.SetThrottle(0)
	bot.HandleJoin([]string{channel})
	bot.joinTimeout = time.Nanosecond
	bot.SetPassword(password)
	bot.Start()
	srv.encode(fmt.Sprintf(":example.com 001 %s :Welcome Internet Relay Chat Network", nick))
	for _, tc := range tt {
		msg, err := srv.decode()
		if err != nil {
			t.Errorf("decoding a message failed: %v", err)
			t.FailNow()
		}
		if !reflect.DeepEqual(tc, msg) {
			t.Errorf("expected %v, got %v", tc, msg)
		}
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()
}

func TestHandlePingPong(t *testing.T) {
	tt := []*irc.Message{
		irc.ParseMessage(fmt.Sprintf("USER %s +iw * %s", nick, nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
		irc.ParseMessage(fmt.Sprintf("PING %s", SERVER)),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.SetThrottle(0)
	bot.pingTick = time.Nanosecond
	bot.HandlePingPong()
	bot.Start()
	srv.encode(":test@test!example.com PRIVMSG :hello")
	srv.encode(":test@test!example.com PRIVMSG :hello")
	for _, tc := range tt {
		msg, err := srv.decode()
		if err != nil {
			t.Errorf("decoding a message failed: %v", err)
			t.FailNow()
		}
		if !reflect.DeepEqual(tc, msg) {
			t.Errorf("expected %v, got %v", tc, msg)
		}
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()
}

func TestHandlePing(t *testing.T) {
	tt := []*irc.Message{
		irc.ParseMessage(fmt.Sprintf("USER %s +iw * %s", nick, nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
		irc.ParseMessage("PONG"),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.SetThrottle(0)
	bot.pingTick = time.Minute
	bot.HandlePingPong()
	bot.Start()
	srv.encode(":example.com PING")
	for _, tc := range tt {
		msg, err := srv.decode()
		if err != nil {
			t.Errorf("decoding a message failed: %v", err)
			t.FailNow()
		}
		if !reflect.DeepEqual(tc, msg) {
			t.Errorf("expected %v, got %v", tc, msg)
		}
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()

}

func TestGetPrefix(t *testing.T) {
	join := fmt.Sprintf(":%s!%s@example.com JOIN %s", nick, nick, channel)
	tt := []*irc.Message{
		irc.ParseMessage(fmt.Sprintf("USER %s +iw * %s", nick, nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.SetThrottle(0)
	bot.testing = true
	bot.Start()
	srv.encode(join)
	<-bot.testchan
	srv.encode(fmt.Sprintf(":example.com 001 %s :Welcome Internet Relay Chat Network", nick))
	for _, tc := range tt {
		msg, err := srv.decode()
		if err != nil {
			t.Errorf("decoding a message failed: %v", err)
			t.FailNow()
		}
		if !reflect.DeepEqual(tc, msg) {
			t.Errorf("expected %v, got %v", tc, msg)
		}
	}
	m := irc.ParseMessage(join)
	<-bot.testchan
	prflen := <-bot.prefixlenGet
	if m.Prefix.Len() != prflen {
		t.Errorf("expected prefix lenght of %d, got %d", m.Prefix.Len(), prflen)
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()

}

func TestHandleNickTaken(t *testing.T) {
	nicktaken := fmt.Sprintf(":example.com 433 * %s :Nickname is already in use.", nick)
	tt := []*irc.Message{
		irc.ParseMessage(fmt.Sprintf("USER %s +iw * %s", nick, nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s_", nick)),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.HandleNickTaken()
	bot.SetThrottle(0)
	bot.testing = false
	bot.Start()
	srv.encode(nicktaken)
	for _, tc := range tt {
		msg, err := srv.decode()
		if err != nil {
			t.Errorf("decoding a message failed: %v", err)
			t.FailNow()
		}
		if !reflect.DeepEqual(tc, msg) {
			t.Errorf("expected %v, got %v", tc, msg)
		}
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()
}

func TestHandleNickTakenPass(t *testing.T) {
	pass := "pass"
	nicktaken := fmt.Sprintf(":example.com 433 * %s :Nickname is already in use.", nick)
	tt := []*irc.Message{
		irc.ParseMessage(fmt.Sprintf("PASS %s", pass)),
		irc.ParseMessage(fmt.Sprintf("USER %s +iw * %s", nick, nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s_", nick)),
		irc.ParseMessage(fmt.Sprintf("PRIVMSG NickServ :GHOST %s %s", nick, pass)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
		irc.ParseMessage(fmt.Sprintf("PRIVMSG NickServ :identify %s %s", nick, pass)),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.SetPassword(pass)
	bot.HandleNickTaken()
	bot.SetThrottle(0)
	bot.testing = true
	bot.Start()
	srv.encode(nicktaken)
	for i, tc := range tt {
		msg, err := srv.decode()
		if err != nil {
			t.Errorf("decoding a message failed: %v", err)
			t.FailNow()
		}
		if i == 3 {
			if !strings.HasPrefix(msg.Trailing(), nick) {
				t.Errorf("expected nick prefix %s, got %s", nick, msg.Name)
				t.FailNow()
			}
			continue
		}
		if !reflect.DeepEqual(tc, msg) {
			t.Errorf("expected %v, got %v", tc, msg)
		}
		if i == 4 {
			<-bot.testchan
			srv.encode(":NickServ!NickServ@services. NOTICE ugjka :ugjka has been ghosted")
			<-bot.testchan
		}
		if i == 5 {
			<-bot.testchan
			srv.encode(":NickServ!NickServ@services. NOTICE ugjka :You are now identified")
			<-bot.testchan
		}
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()
}

func TestNewMessage(t *testing.T) {
	msg := NewMessage()
	if msg.Prefix == nil {
		t.Error("prefix is nil")
	}
	if msg.Params == nil {
		t.Error("params is nil")
	}
}

func TestMsg(t *testing.T) {
	chans := []string{"#test", "#test2"}
	longmsg := strings.Repeat("*", 512)
	prfxlen := len("PRIVMSG #test :") + len(nick) + 2
	tt := []*irc.Message{
		irc.ParseMessage(fmt.Sprintf("USER %s +iw * %s", nick, nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
		irc.ParseMessage(fmt.Sprintf("PRIVMSG %s :hello", chans[0])),
		irc.ParseMessage(fmt.Sprintf("PRIVMSG %s :hello", chans[1])),
		irc.ParseMessage(fmt.Sprintf("PRIVMSG #test :%s", longmsg[:510-prfxlen])),
		irc.ParseMessage(fmt.Sprintf("PRIVMSG #test :%s", longmsg[:len(longmsg)-510+prfxlen])),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.SetThrottle(0)
	bot.Start()
	go func(bot *Connection) {
		bot.MsgBulk(chans, "hello")
		bot.Msg("#test", longmsg)
	}(bot)
	for _, tc := range tt {
		msg, err := srv.decode()
		if err != nil {
			t.Errorf("decoding a message failed: %v", err)
			t.FailNow()
		}
		if !reflect.DeepEqual(tc, msg) {
			t.Errorf("expected %v, got %v", tc, msg)
		}
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()
}

func TestReply(t *testing.T) {
	msgs := []*Message{
		NewMessage(),
		NewMessage(),
	}
	msgs[0].To = nick
	msgs[1].To = "#test"
	tt := []*irc.Message{
		irc.ParseMessage(fmt.Sprintf("USER %s +iw * %s", nick, nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
		irc.ParseMessage(fmt.Sprintf("PRIVMSG :hello")),
		irc.ParseMessage(fmt.Sprintf("PRIVMSG %s :hello", "#test")),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.SetThrottle(0)
	bot.Start()
	go func(bot *Connection) {
		bot.Reply(msgs[0], "hello")
		bot.Reply(msgs[1], "hello")
	}(bot)
	for _, tc := range tt {
		msg, err := srv.decode()
		if err != nil {
			t.Errorf("decoding a message failed: %v", err)
			t.FailNow()
		}
		if !reflect.DeepEqual(tc, msg) {
			t.Errorf("expected %v, got %v", tc, msg)
		}
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()
}

func TestNotice(t *testing.T) {
	longmsg := strings.Repeat("*", 512)
	dst := "test"
	prfxlen := len(fmt.Sprintf("NOTICE %s :", dst)) + len(nick) + 2
	tt := []*irc.Message{
		irc.ParseMessage(fmt.Sprintf("USER %s +iw * %s", nick, nick)),
		irc.ParseMessage(fmt.Sprintf("NICK %s", nick)),
		irc.ParseMessage(fmt.Sprintf("NOTICE %s :hello", dst)),
		irc.ParseMessage(fmt.Sprintf("NOTICE %s :%s", dst, longmsg[:510-prfxlen])),
		irc.ParseMessage(fmt.Sprintf("NOTICE %s :%s", dst, longmsg[:len(longmsg)-510+prfxlen])),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.SetThrottle(0)
	bot.Start()
	go func(bot *Connection) {
		bot.Notice(dst, "hello")
		bot.Notice(dst, longmsg)
	}(bot)
	for _, tc := range tt {
		msg, err := srv.decode()
		if err != nil {
			t.Errorf("decoding a message failed: %v", err)
			t.FailNow()
		}
		if !reflect.DeepEqual(tc, msg) {
			t.Errorf("expected %v, got %v", tc, msg)
		}
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()
}
