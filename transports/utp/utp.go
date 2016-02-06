package utp

import (
	"net"

	"github.com/anacrolix/utp"
	"google.golang.org/grpc"
)

func NewDialer(s *utp.Socket) grpc.DialOption {
	if s == nil {
		return grpc.WithDialer(utp.DialTimeout)
	}
	return grpc.WithDialer(s.DialTimeout)
}

func NewListener(s *utp.Socket) net.Listener {
	return s
}
