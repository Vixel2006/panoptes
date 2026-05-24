package adapter

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

type ProxyAdapter struct {
	ListenAddr string
	Port       int32
}

func NewServer(listenAddr string, port int32) *ProxyAdapter {
	return &ProxyAdapter{
		ListenAddr: listenAddr,
		Port:       port,
	}
}

func (p *ProxyAdapter) Connect(ctx context.Context) {
	conn, err := net.Dial("tcp", b.ListenAddr)
	if err != nil {
		log.Fatal("error connecting to server: ", err)
	}

	fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")
}
