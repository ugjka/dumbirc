package dumbirc

import (
	"fmt"
	"os"
	"reflect"
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
	bot.SetDebugOutput(os.Stderr)
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
	bot.HandleJoin([]string{channel})
	bot.SetDebugOutput(os.Stderr)
	bot.SetLogOutput(os.Stderr)
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
			time.Sleep(time.Millisecond * 100)
			srv.encode(fmt.Sprintf(":connect!admin@test.com NOTICE %s :You are now identified for", nick))
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
	bot.SetDebugOutput(os.Stderr)
	bot.joinTimeout = time.Nanosecond
	bot.SetLogOutput(os.Stderr)
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
	bot.SetDebugOutput(os.Stderr)
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
	bot.SetDebugOutput(os.Stderr)
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
	bot.SetDebugOutput(os.Stderr)
	bot.Start()
	srv.encode(join)
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
	time.Sleep(time.Millisecond)
	prflen := <-bot.prefixlenGet
	if m.Prefix.Len() != prflen {
		t.Errorf("expected prefix lenght of %d, got %d", m.Prefix.Len(), prflen)
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()

}
