package main

import (
	"log"
	"time"

	"github.com/ugjka/dumbirc"
)

func main() {
	channels := []string{"#testy123"}
	conn := dumbirc.New("testbbt", "bottob", "irc.freenode.net:7000", true)
	conn.AddCallback(dumbirc.WELCOME, func() {
		conn.Join(channels)
	})
	conn.AddCallback(dumbirc.PING, func() {
		conn.Pong()
	})
	conn.AddCallback(dumbirc.PRIVMSG, func() {
		if conn.Msg.Trailing == "hello" {
			conn.Reply("Hi, How are you?")
		}
	})
	conn.AddCallback(dumbirc.NICKTAKEN, func() {
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
