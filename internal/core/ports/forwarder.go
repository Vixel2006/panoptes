package port

import "net/http"

type HTTPForwarder interface {
	RoundTrip(*http.Request) (*http.Response, error)
}
