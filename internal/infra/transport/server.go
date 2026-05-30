package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
)

type Server struct {
	ListenAddr string
	Port       int32
	Handler    func(net.Conn)
	listener   net.Listener
}

func NewServer(listenAddr string, port int32, handler func(net.Conn)) *Server {
	return &Server{
		ListenAddr: listenAddr,
		Port:       port,
		Handler:    handler,
	}
}

func (s *Server) Start(ctx context.Context) error {
	if s.Handler == nil {
		return errors.New("no connection handler set")
	}

	port := ":" + strconv.Itoa(int(s.Port))
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = listener

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			continue
		}
		go s.Handler(conn)
	}
}

func (s *Server) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}
