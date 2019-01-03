package dumbirc

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"

	irc "gopkg.in/sorcix/irc.v2"
)

const SERVER = ":54321"

type ircServer struct {
	in       chan []byte
	out      chan []byte
	outBuff  *bytes.Buffer
	dec      *irc.Decoder
	enc      *irc.Encoder
	listener net.Listener
	kill     chan struct{}
	wg       *sync.WaitGroup
}

func (i *ircServer) Write(b []byte) (n int, err error) {
	i.in <- b
	return len(b), nil
}

func (i *ircServer) Read(b []byte) (n int, err error) {
	tmp := <-i.out
	copy(b, tmp)
	return len(tmp), nil
}

func newServer() *ircServer {
	s := &ircServer{
		in:      make(chan []byte, 100),
		out:     make(chan []byte, 100),
		outBuff: bytes.NewBuffer(nil),
		kill:    make(chan struct{}, 0),
		wg:      &sync.WaitGroup{},
	}
	s.dec = irc.NewDecoder(s)
	s.enc = irc.NewEncoder(s)
	s.startListener()
	go s.monitor()
	return s
}

func (i *ircServer) startListener() {
	l, err := net.Listen("tcp", SERVER)
	if err != nil {
		panic(err)
	}
	i.listener = l
}

func (i *ircServer) monitor() {
	conn, err := i.listener.Accept()
	if err != nil {
		panic(err)
	}
	i.wg.Add(1)
	go func(i *ircServer, c net.Conn) {
		defer i.wg.Done()
		for {
			select {
			case <-i.kill:
				return
			case tmp := <-i.in:
				io.WriteString(c, string(tmp))
			}
		}
	}(i, conn)
	i.wg.Add(1)
	go func(i *ircServer, c net.Conn) {
		defer i.wg.Done()
		sc := bufio.NewScanner(c)
		for sc.Scan() {
			fmt.Println(sc.Text())
			i.out <- []byte(sc.Text() + "\n")
		}
	}(i, conn)
}

func (i *ircServer) stop() {
	i.listener.Close()
	close(i.kill)
}

func (i *ircServer) wait() {
	i.wg.Wait()
}

func (i *ircServer) encode(msg string) (err error) {
	m := irc.ParseMessage(msg)
	return i.enc.Encode(m)
}

func (i *ircServer) decode() (msg *irc.Message, err error) {
	return i.dec.Decode()
}
