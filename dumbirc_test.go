package dumbirc

import (
	"fmt"
	"os"
	"testing"
)

func TestServer(t *testing.T) {
	srv := newServer()
	bot := New("ugjka", "ugjka", SERVER, false)
	bot.HandleJoin([]string{"#ugjka"})
	bot.SetDebugOutput(os.Stderr)
	bot.Start()
	srv.encode(":tepper.freenode.net 001 ugjka :Welcome Internet Relay Chat Network ugjka\n")
	for {
		msg, err := srv.decode()
		if err != nil {
			break
		}
		fmt.Println(msg)
	}
	srv.stop()
}
