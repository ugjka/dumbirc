package main

import (
	"log"
	"time"

	"github.com/ugjka/dumbirc"
)

func main() {
	channels := []string{"#testy123"}
	conn := dumbirc.New("testbbt", "bottob", "irc.freenode.net:7000", true)
	conn.AddCallback(dumbirc.WELCOME, func(msg dumbirc.Message) {
		conn.Join(channels)
	})
	conn.AddCallback(dumbirc.PING, func(msg dumbirc.Message) {
		conn.Pong()
	})
	conn.AddCallback(dumbirc.PRIVMSG, func(msg dumbirc.Message) {
		if msg.Trailing == "hello" {
			conn.Reply(msg, "Hi, How are you?")
		}
	})
	conn.AddCallback(dumbirc.NICKTAKEN, func(msg dumbirc.Message) {
		conn.Nick += "_"
		conn.NewNick(conn.Nick)
	})
	conn.Start()
	//Ping the server
	go func() {
		for {
			time.Sleep(time.Minute)
			conn.Ping()
		}
	}()
	//If error then exit
	log.Println(<-conn.Errchan)
}
