package adapter

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	infraTls "github.com/Vixel2006/panoptes/internal/infra/tls"
)

type peekConn struct {
	*bufio.Reader
	net.Conn
}

func (c *peekConn) Read(b []byte) (int, error) {
	return c.Reader.Read(b)
}

type InterceptAdapter struct {
	certGen *infraTls.CertificateGenerator
}

func NewInterceptAdapter(certGen *infraTls.CertificateGenerator) *InterceptAdapter {
	return &InterceptAdapter{certGen: certGen}
}

func (a *InterceptAdapter) HandleConn(conn net.Conn) {
	defer conn.Close()

	peekBuf := bufio.NewReader(conn)
	b, err := peekBuf.Peek(1)
	if err != nil {
		return
	}

	// TLS record: 0x16 = Handshake, 0x03 = major version 3
	if b[0] == 0x16 {
		a.handleTLS(&peekConn{Reader: peekBuf, Conn: conn})
		return
	}

	req, err := http.ReadRequest(peekBuf)
	if err != nil {
		fmt.Printf("Error reading request: %v\n", err)
		return
	}

	if req.Method == "CONNECT" {
		a.handleConnect(conn, req)
		return
	}

	a.handleHTTP(conn, req)
}

func (a *InterceptAdapter) handleTLS(conn net.Conn) {
	tlsConfig := &tls.Config{
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			hostname := info.ServerName
			if hostname == "" {
				hostname = "proxy.local"
			}
			cert, err := a.certGen.IssueLeaf(hostname)
			if err != nil {
				return nil, err
			}
			return &tls.Config{Certificates: []tls.Certificate{*cert}}, nil
		},
	}

	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		fmt.Printf("TLS handshake error: %v\n", err)
		return
	}

	reader := bufio.NewReader(tlsConn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			break
		}

		if req.Method == "CONNECT" {
			a.handleConnect(tlsConn, req)
			continue
		}

		a.handleHTTPTLS(tlsConn, req)
	}
}

func (a *InterceptAdapter) handleConnect(conn net.Conn, req *http.Request) {
	hostname := strings.SplitN(req.Host, ":", 2)[0]
	fmt.Printf("CONNECT to %s\n", hostname)

	cert, err := a.certGen.IssueLeaf(hostname)
	if err != nil {
		fmt.Printf("Error issuing cert for %s: %v\n", hostname, err)
		return
	}

	_, err = io.WriteString(conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
	if err != nil {
		fmt.Printf("Error sending 200: %v\n", err)
		return
	}

	tlsConn := tls.Server(conn, &tls.Config{
		Certificates: []tls.Certificate{*cert},
	})

	if err := tlsConn.Handshake(); err != nil {
		fmt.Printf("TLS handshake failed with %s: %v\n", hostname, err)
		fmt.Println("  → Install the root CA (certs/panoptes-ca.crt) in your browser to intercept HTTPS.")
		return
	}
	fmt.Printf("TLS established with %s\n", hostname)

	reader := bufio.NewReader(tlsConn)
	for {
		subReq, err := http.ReadRequest(reader)
		if err != nil {
			break
		}

		target := subReq.Host
		if !strings.Contains(target, ":") {
			target = target + ":443"
		}

		upConn, err := tls.Dial("tcp", target, &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         hostname,
		})
		if err != nil {
			fmt.Printf("Error dialing upstream %s: %v\n", target, err)
			subReq.Body.Close()
			break
		}

		err = subReq.Write(upConn)
		if err != nil {
			fmt.Printf("Error writing to upstream: %v\n", err)
			upConn.Close()
			subReq.Body.Close()
			break
		}

		resp, err := http.ReadResponse(bufio.NewReader(upConn), subReq)
		if err != nil {
			fmt.Printf("Error reading upstream response: %v\n", err)
			upConn.Close()
			subReq.Body.Close()
			break
		}

		resp.Write(tlsConn)
		resp.Body.Close()
		upConn.Close()
		subReq.Body.Close()
	}
}

func (a *InterceptAdapter) handleHTTP(conn net.Conn, req *http.Request) {
	targetHost := req.Host
	if targetHost == "" {
		targetHost = req.URL.Host
	}
	if !strings.Contains(targetHost, ":") {
		targetHost = targetHost + ":80"
	}

	fmt.Printf("HTTP %s to %s\n", req.Method, targetHost)

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

func (a *InterceptAdapter) handleHTTPTLS(conn net.Conn, req *http.Request) {
	target := req.Host
	if target == "" {
		target = req.URL.Host
	}
	if !strings.Contains(target, ":") {
		target = target + ":443"
	}

	fmt.Printf("HTTPS %s to %s\n", req.Method, target)

	upConn, err := tls.Dial("tcp", target, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		fmt.Printf("Error dialing upstream %s: %v\n", target, err)
		return
	}
	defer upConn.Close()

	origURL := req.URL
	req.URL = &url.URL{
		Path:     origURL.Path,
		RawQuery: origURL.RawQuery,
	}
	req.RequestURI = ""

	err = req.Write(upConn)
	if err != nil {
		fmt.Printf("Error forwarding request: %v\n", err)
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(upConn), req)
	if err != nil {
		fmt.Printf("Error reading upstream response: %v\n", err)
		return
	}
	defer resp.Body.Close()

	err = resp.Write(conn)
	if err != nil {
		fmt.Printf("Error writing response: %v\n", err)
	}
}
