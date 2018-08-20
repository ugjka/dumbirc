package main

import (
	"log"
	"os"
	"strings"

	"github.com/ugjka/dumbirc"
)

func main() {
	channels := []string{"#test13246"}
	irc := dumbirc.New("testnick2344", "testnick", "irc.freenode.net:7000", true)
	irc.HandleJoin(channels)
	irc.HandleNickTaken()
	irc.HandlePingPong()
	irc.LogNotices()
	irc.SetLogOutput(os.Stdout)
	//irc.EnableDebug(os.Stdout)
	irc.AddCallback(dumbirc.PRIVMSG, func(msg *dumbirc.Message) {
		if msg.Trailing == "hello" {
			irc.Reply(msg, "Hi, How are you?")
		}
	})
	irc.AddTrigger(dumbirc.Trigger{
		Condition: func(m *dumbirc.Message) bool {
			return m.Command == dumbirc.KICK && strings.HasPrefix(m.Params[1], irc.Nick)
		},
		Response: func(m *dumbirc.Message) {
			irc.Join([]string{m.Params[0]})
		},
	})
	irc.Start()
	//If error then exit
	log.Println(<-irc.Errchan)
}
