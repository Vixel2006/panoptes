package adapter

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type InterceptAdapter struct{}

func NewInterceptAdapter() *InterceptAdapter {
	return &InterceptAdapter{}
}

func (a *InterceptAdapter) HandleConn(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("Intercepted connection from: %s\n", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		fmt.Printf("Error reading request: %v\n", err)
		return
	}

	targetHost := req.Host
	if targetHost == "" {
		targetHost = req.URL.Host
	}
	if !strings.Contains(targetHost, ":") {
		targetHost = targetHost + ":80"
	}

	fmt.Printf("Forwarding to: %s %s\n", req.Method, targetHost)

	targetConn, err := net.Dial("tcp", targetHost)
	if err != nil {
		fmt.Printf("Error dialing target: %v\n", err)
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer targetConn.Close()

	origURL := req.URL
	req.URL = &url.URL{
		Path:     origURL.Path,
		RawQuery: origURL.RawQuery,
	}
	req.RequestURI = ""

	err = req.Write(targetConn)
	if err != nil {
		fmt.Printf("Error forwarding request: %v\n", err)
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(targetConn), req)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}
	defer resp.Body.Close()

	err = resp.Write(conn)
	if err != nil {
		fmt.Printf("Error writing response: %v\n", err)
	}
}
