package adapter

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Vixel2006/panoptes/internal/core/services"

	cert "github.com/Vixel2006/panoptes/internal/infra/tls"
)

type InterceptAdapter struct {
	certGen     *cert.CertificateGenerator
	barrier     *service.Barrier
	interceptor *service.Interceptor
}

func NewInterceptAdapter(certGen *cert.CertificateGenerator) *InterceptAdapter {
	return &InterceptAdapter{
		certGen:     certGen,
		barrier:     service.NewBarrier(),
		interceptor: &service.Interceptor{},
	}
}

func (i *InterceptAdapter) HandleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer conn.SetDeadline(time.Time{})

	prefix := make([]byte, 4)
	if _, err := io.ReadFull(conn, prefix); err != nil {
		return
	}

	if string(prefix) == "CONN" {
		i.handleConnect(conn, prefix)
	} else {
		i.handleHTTP(conn, prefix)
	}
}

func (i *InterceptAdapter) handleConnect(conn net.Conn, prefix []byte) {
	rest := make([]byte, 0, 256)
	for {
		b := make([]byte, 1)
		if _, err := conn.Read(b); err != nil {
			return
		}
		rest = append(rest, b[0])
		if b[0] == '\n' {
			break
		}
	}

	parts := strings.Fields(string(prefix) + string(rest))
	if len(parts) < 2 {
		return
	}
	host := parts[1]
	if !strings.Contains(host, ":") {
		host += ":443"
	}

	if _, err := fmt.Fprintf(conn, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		return
	}

	hostname := strings.Split(host, ":")[0]
	leafCert, err := i.certGen.IssueLeaf(hostname)
	if err != nil {
		return
	}

	tlsConn := tls.Server(conn, &tls.Config{
		Certificates: []tls.Certificate{*leafCert},
	})
	if err := tlsConn.Handshake(); err != nil {
		return
	}
	defer tlsConn.Close()

	tlsBr := bufio.NewReader(tlsConn)
	httpReq, err := http.ReadRequest(tlsBr)
	if err != nil {
		return
	}

	httpReq.URL.Scheme = "https"
	httpReq.URL.Host = host
	if httpReq.Host == "" {
		httpReq.Host = hostname
	}

	i.interceptor.InterceptRequest(httpReq)

	i.barrier.Lock()
	forward := i.barrier.Decision()
	i.barrier.Unlock()

	if !forward {
		dropResp := &http.Response{
			StatusCode: http.StatusForbidden,
			ProtoMajor: 1,
			ProtoMinor: 1,
		}
		dropResp.Write(tlsConn)
		return
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}
	upstreamResp, err := transport.RoundTrip(httpReq)
	if err != nil {
		errResp := &http.Response{
			StatusCode: http.StatusBadGateway,
			ProtoMajor: 1,
			ProtoMinor: 1,
		}
		errResp.Write(tlsConn)
		return
	}
	defer upstreamResp.Body.Close()

	i.interceptor.InterceptResponse(upstreamResp)
	upstreamResp.Write(tlsConn)
}

func (i *InterceptAdapter) handleHTTP(conn net.Conn, prefix []byte) {
	br := bufio.NewReader(io.MultiReader(bytes.NewReader(prefix), conn))
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}

	if !req.URL.IsAbs() {
		req.URL.Scheme = "http"
		req.URL.Host = req.Host
	}

	i.interceptor.InterceptRequest(req)

	i.barrier.Lock()
	forward := i.barrier.Decision()
	i.barrier.Unlock()

	if !forward {
		dropResp := &http.Response{
			StatusCode: http.StatusForbidden,
			ProtoMajor: 1,
			ProtoMinor: 1,
		}
		dropResp.Write(conn)
		return
	}

	transport := &http.Transport{}
	upstreamResp, err := transport.RoundTrip(req)
	if err != nil {
		errResp := &http.Response{
			StatusCode: http.StatusBadGateway,
			ProtoMajor: 1,
			ProtoMinor: 1,
		}
		errResp.Write(conn)
		return
	}
	defer upstreamResp.Body.Close()

	i.interceptor.InterceptResponse(upstreamResp)
	upstreamResp.Write(conn)
}
