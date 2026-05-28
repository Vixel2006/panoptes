package adapter

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/core/services"

	cert "github.com/Vixel2006/panoptes/internal/infra/tls"
)

type InterceptAdapter struct {
	certGen     *cert.CertificateGenerator
	barrier     *service.Barrier
	interceptor *service.Interceptor
	requestCh   chan model.Request
}

type bufferedConn struct {
	net.Conn
	r io.Reader
}

func (bc *bufferedConn) Read(b []byte) (int, error) { return bc.r.Read(b) }

func NewInterceptAdapter(certGen *cert.CertificateGenerator) *InterceptAdapter {
	requestCh := make(chan model.Request, 100)
	interceptor := service.NewInterceptor(requestCh)
	return &InterceptAdapter{
		certGen:     certGen,
		barrier:     service.NewBarrier(),
		interceptor: interceptor,
		requestCh:   requestCh,
	}
}

func (i *InterceptAdapter) Barrier() *service.Barrier {
	return i.barrier
}

func (i *InterceptAdapter) RequestCh() <-chan model.Request {
	return i.requestCh
}

func (i *InterceptAdapter) HandleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	br := bufio.NewReader(conn)

	peek, err := br.Peek(1)
	if err != nil {
		return
	}

	switch {
	case peek[0] == 0x16:
		conn.SetDeadline(time.Time{})
		i.handleTLS(conn, br)
	default:
		conn.SetDeadline(time.Time{})
		fullPeek, err := br.Peek(7)
		if err != nil {
			return
		}
		if string(fullPeek) == "CONNECT" {
			i.handleConnect(conn, br)
		} else {
			i.handleHTTP(conn, br)
		}
	}
}

func (i *InterceptAdapter) handleConnect(conn net.Conn, br *bufio.Reader) {
	bc := &bufferedConn{Conn: conn, r: br}

	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	host := req.Host

	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	tlsConfig := &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			name := hello.ServerName
			if name == "" {
				name, _, _ = net.SplitHostPort(host)
			}
			return i.certGen.IssueLeaf(name)
		},
	}
	tlsConn := tls.Server(bc, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return
	}

	i.interceptTLS(tlsConn, host)
}

func (i *InterceptAdapter) handleTLS(conn net.Conn, br *bufio.Reader) {
	bc := &bufferedConn{Conn: conn, r: br}

	outerConfig := &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			name := hello.ServerName
			if name == "" {
				name = "localhost"
			}
			return i.certGen.IssueLeaf(name)
		},
	}
	outer := tls.Server(bc, outerConfig)
	if err := outer.Handshake(); err != nil {
		return
	}
	defer outer.Close()

	outerBr := bufio.NewReader(outer)
	for {
		req, err := http.ReadRequest(outerBr)
		if err != nil {
			break
		}

		if req.Method == "CONNECT" {
			host := req.Host
			outer.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

			innerBC := &bufferedConn{Conn: outer, r: outerBr}
			innerConfig := &tls.Config{
				GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
					name := hello.ServerName
					if name == "" {
						name, _, _ = net.SplitHostPort(host)
					}
					return i.certGen.IssueLeaf(name)
				},
			}
			inner := tls.Server(innerBC, innerConfig)
			if err := inner.Handshake(); err != nil {
				break
			}
			i.interceptTLS(inner, host)
			break
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
			dropResp.Write(outer)
			continue
		}

		transport := &http.Transport{Proxy: nil}
		upstreamResp, err := transport.RoundTrip(req)
		if err != nil {
			errResp := &http.Response{
				StatusCode: http.StatusBadGateway,
				ProtoMajor: 1,
				ProtoMinor: 1,
			}
			errResp.Write(outer)
			continue
		}
		i.interceptor.InterceptResponse(upstreamResp)
		upstreamResp.Write(outer)
		upstreamResp.Body.Close()
	}
}

func (i *InterceptAdapter) interceptTLS(tlsConn net.Conn, host string) {
	tlsBr := bufio.NewReader(tlsConn)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}
	for {
		req, err := http.ReadRequest(tlsBr)
		if err != nil {
			break
		}
		req.URL = &url.URL{Scheme: "https", Host: host, Path: req.RequestURI}
		req.RequestURI = ""

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
			dropResp.Write(tlsConn)
			continue
		}

		upstreamResp, err := transport.RoundTrip(req)
		if err != nil {
			errResp := &http.Response{
				StatusCode: http.StatusBadGateway,
				ProtoMajor: 1,
				ProtoMinor: 1,
			}
			errResp.Write(tlsConn)
			continue
		}
		i.interceptor.InterceptResponse(upstreamResp)
		upstreamResp.Write(tlsConn)
		upstreamResp.Body.Close()
	}
}

func (i *InterceptAdapter) handleHTTP(conn net.Conn, br *bufio.Reader) {
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

	transport := &http.Transport{Proxy: nil}
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
