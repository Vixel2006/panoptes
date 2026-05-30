package adapter

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/core/ports"
)

type InterceptAdapter struct {
	certGen      port.CertificateIssuer
	barrier      port.BarrierPort
	interceptor  port.InterceptorPort
	decompressor port.Decompressor
	idGen        port.IDGenerator
	forwarder    port.HTTPForwarder
	requestCh    chan model.Request
}

type bufferedConn struct {
	net.Conn
	r io.Reader
}

func (bc *bufferedConn) Read(b []byte) (int, error) { return bc.r.Read(b) }

func NewInterceptAdapter(
	certGen port.CertificateIssuer,
	barrier port.BarrierPort,
	interceptor port.InterceptorPort,
	decompressor port.Decompressor,
	idGen port.IDGenerator,
	forwarder port.HTTPForwarder,
	requestCh chan model.Request,
) *InterceptAdapter {
	return &InterceptAdapter{
		certGen:      certGen,
		barrier:      barrier,
		interceptor:  interceptor,
		decompressor: decompressor,
		idGen:        idGen,
		forwarder:    forwarder,
		requestCh:    requestCh,
	}
}

func (a *InterceptAdapter) Close() {
	a.interceptor.Stop()
}

func (i *InterceptAdapter) Barrier() port.BarrierPort {
	return i.barrier
}

func (i *InterceptAdapter) Interceptor() port.InterceptorPort {
	return i.interceptor
}

func (i *InterceptAdapter) RequestCh() <-chan model.Request {
	return i.requestCh
}

func (i *InterceptAdapter) buildRequest(r *http.Request) model.Request {
	rawBody, _ := io.ReadAll(r.Body)
	r.Body.Close()

	r.Body = io.NopCloser(bytes.NewReader(rawBody))
	r.ContentLength = int64(len(rawBody))
	r.Header.Del("Transfer-Encoding")

	storedBody, _ := i.decompressor.Decompress(r.Header.Get("Content-Encoding"), rawBody)
	headerJSON, _ := json.Marshal(r.Header)

	return model.Request{
		ID:      i.idGen.New(),
		URL:     r.URL.String(),
		Method:  r.Method,
		Header:  json.RawMessage(headerJSON),
		Payload: json.RawMessage(storedBody),
		Length:  r.ContentLength,
	}
}

func (i *InterceptAdapter) buildResponse(r *http.Response) model.Response {
	rawBody, _ := io.ReadAll(r.Body)
	r.Body.Close()

	r.Body = io.NopCloser(bytes.NewReader(rawBody))
	r.ContentLength = int64(len(rawBody))
	r.Header.Del("Transfer-Encoding")

	storedBody, _ := i.decompressor.Decompress(r.Header.Get("Content-Encoding"), rawBody)
	headerJSON, _ := json.Marshal(r.Header)

	return model.Response{
		ID:         i.idGen.New(),
		Status:     r.Status,
		StatusCode: r.StatusCode,
		Header:     json.RawMessage(headerJSON),
		Payload:    json.RawMessage(storedBody),
		Length:     r.ContentLength,
	}
}

// InterceptRoundTrip runs the full interception pipeline for a single request:
// build model, intercept, check barrier, forward, build response, intercept response.
// It writes the final HTTP response to w.
func (i *InterceptAdapter) InterceptRoundTrip(req *http.Request, w io.Writer) error {
	modelReq := i.buildRequest(req)
	i.interceptor.InterceptRequest(modelReq)

	i.barrier.Lock()
	forward := i.barrier.Decision()
	i.barrier.Unlock()

	if !forward {
		dropResp := &http.Response{
			StatusCode: http.StatusForbidden,
			ProtoMajor: 1,
			ProtoMinor: 1,
		}
		return dropResp.Write(w)
	}

	upstreamResp, err := i.forwarder.RoundTrip(req)
	if err != nil {
		errResp := &http.Response{
			StatusCode: http.StatusBadGateway,
			ProtoMajor: 1,
			ProtoMinor: 1,
		}
		return errResp.Write(w)
	}
	defer upstreamResp.Body.Close()

	modelResp := i.buildResponse(upstreamResp)
	i.interceptor.InterceptResponse(modelResp, modelReq.ID)
	return upstreamResp.Write(w)
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

		if err := i.InterceptRoundTrip(req, outer); err != nil {
			break
		}
	}
}

func (i *InterceptAdapter) interceptTLS(tlsConn net.Conn, host string) {
	tlsBr := bufio.NewReader(tlsConn)
	for {
		req, err := http.ReadRequest(tlsBr)
		if err != nil {
			break
		}
		req.URL = &url.URL{Scheme: "https", Host: host, Path: req.RequestURI}
		req.RequestURI = ""

		if err := i.InterceptRoundTrip(req, tlsConn); err != nil {
			break
		}
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

	i.InterceptRoundTrip(req, conn)
}
