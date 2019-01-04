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
			srv.encode(fmt.Sprintf(":connect!admin@test.com NOTICE %s :You are now identified for", nick))
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
		irc.ParseMessage("PONG"),
	}
	srv := newServer()
	bot := New(nick, nick, SERVER, false)
	bot.SetThrottle(0)
	bot.pingTick = time.Nanosecond
	bot.HandlePingPong()
	bot.SetDebugOutput(os.Stderr)
	bot.Start()
	srv.encode("ANY")
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
			srv.encode(fmt.Sprintf("PING"))
		}
	}
	bot.Disconnect()
	Destroy(bot)
	srv.stop()

}
