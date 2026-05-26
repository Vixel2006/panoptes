package port

import (
	"net/http"
)

type InterceptorPort interface {
	InterceptRequest(req *http.Request) error
	InterceptResponse(resp *http.Response) error
}
