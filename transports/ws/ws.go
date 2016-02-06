package ws

import (
	"io"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
	"google.golang.org/grpc"
)

func NewDialer() grpc.DialOption {
	return grpc.WithDialer(func(addr string, _ time.Duration) (net.Conn, error) {
		conn, err := websocket.Dial(addr, "ws", "localhost")
		if err != nil {
			return nil, err
		}
		return conn, nil
	})
}

func NewListener() (net.Listener, http.Handler) {
	l, push := newFakeListener(nil)
	h := websocket.Handler(func(conn *websocket.Conn) {

		if err := push(conn); err != nil {
			return
		}

	})
	return l, h
}

func newFakeListener(addr net.Addr) (net.Listener, func(net.Conn) error) {
	l := &fakeListner{
		addr:    addr,
		inbound: make(chan net.Conn),
		closed:  make(chan struct{}),
	}
	return l, l.push
}

type fakeListner struct {
	addr    net.Addr
	inbound chan net.Conn
	closed  chan struct{}
}

func (l *fakeListner) push(conn net.Conn) error {
	select {
	case <-l.closed:
		return io.EOF
	case l.inbound <- conn:
		return nil
	}
}

func (l *fakeListner) Addr() net.Addr { return l.addr }

func (l *fakeListner) Close() error {
	defer func() { recover() }()
	close(l.closed)
	return nil
}

func (l *fakeListner) Accept() (net.Conn, error) {
	for {
		select {
		case <-l.closed:
			return nil, io.EOF
		case inbound := <-l.inbound:
			if inbound != nil {
				return inbound, nil
			}
		}
	}
}
