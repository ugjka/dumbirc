package dumbirc

import (
	"net"

	irc "gopkg.in/sorcix/irc.v2"
)

const SERVER = "127.0.0.1:54321"

type ircServer struct {
	dec       *irc.Decoder
	enc       *irc.Encoder
	listener  net.Listener
	conn      net.Conn
	connReady chan struct{}
}

func (i *ircServer) Write(b []byte) (n int, err error) {
	<-i.connReady
	return i.conn.Write(b)
}

func (i *ircServer) Read(b []byte) (n int, err error) {
	<-i.connReady
	return i.conn.Read(b)
}

func newServer() *ircServer {
	s := &ircServer{
		connReady: make(chan struct{}),
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
	i.conn = conn
	close(i.connReady)
}

func (i *ircServer) stop() {
	i.listener.Close()
	i.conn.Close()
}

func (i *ircServer) encode(msg string) (err error) {
	m := irc.ParseMessage(msg)
	return i.enc.Encode(m)
}

func (i *ircServer) decode() (msg *irc.Message, err error) {
	return i.dec.Decode()
}
