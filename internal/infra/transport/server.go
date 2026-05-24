package transport

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
)

type Server struct {
	ListenAddr string
	Port       int32
	Handler    func(net.Conn)
}

func NewServer(listenAddr string, port int32, handler func(net.Conn)) *Server {
	return &Server{
		ListenAddr: listenAddr,
		Port:       port,
		Handler:    handler,
	}
}

func (s *Server) Start(ctx context.Context) {
	if s.Handler == nil {
		log.Fatal("no connection handler set")
	}

	port := ":" + strconv.Itoa(int(s.Port))
	server, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("error opening the server: ", err)
	}

	for {
		conn, err := server.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err)
			continue
		}
		go s.Handler(conn)
	}
}
