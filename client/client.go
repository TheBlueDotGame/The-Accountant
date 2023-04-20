package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bartossh/The-Accountant/server"
	"github.com/valyala/fasthttp"
)

var (
	ErrApiVersionMismatch = fmt.Errorf("api version mismatch")
	ErrApiHeaderMismatch  = fmt.Errorf("api header mismatch")
)

// Rest is a rest client for the API.
type Rest struct {
	apiRoot string
	timeout time.Duration
}

// NewRest creates a new rest client.
func NewRest(apiRoot string, timeout time.Duration) *Rest {
	return &Rest{apiRoot: apiRoot, timeout: timeout}
}

func (r *Rest) makePost(path string, out, in any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(fmt.Sprintf("%s/%s", r.apiRoot, path))
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	raw, err := json.Marshal(out)
	if err != nil {
		return err
	}
	req.SetBody(raw)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.DoTimeout(req, resp, r.timeout); err != nil {
		return err
	}

	switch resp.StatusCode() {
	case fasthttp.StatusOK, fasthttp.StatusCreated, fasthttp.StatusAccepted:
	case fasthttp.StatusNoContent:
		return nil
	default:
		return fmt.Errorf("expected status code %d but got %d", fasthttp.StatusOK, resp.StatusCode())
	}

	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		return fmt.Errorf("expected content type application/json but got %s", contentType)
	}

	return json.Unmarshal(resp.Body(), in)
}

func (r *Rest) makeGet(path string, out any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(fmt.Sprintf("%s/%s", r.apiRoot, path))
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.DoTimeout(req, resp, r.timeout); err != nil {
		return err
	}

	switch resp.StatusCode() {
	case fasthttp.StatusOK:
	case fasthttp.StatusNoContent:
		return nil
	default:
		return fmt.Errorf("expected status code %d but got %d", fasthttp.StatusOK, resp.StatusCode())
	}

	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		return fmt.Errorf("expected content type application/json but got %s", contentType)
	}

	return json.Unmarshal(resp.Body(), out)
}

// ValidateApiVersion makes a call to the API server and validates client and server API versions and header correctness.
func (r *Rest) ValidateApiVersion() error {
	var alive server.AliveResponse
	if err := r.makeGet("alive", &alive); err != nil {
		return err
	}

	if alive.APIVersion != server.ApiVersion {
		return errors.Join(ErrApiVersionMismatch, fmt.Errorf("expected %s but got %s", server.ApiVersion, alive.APIVersion))
	}

	if alive.APIHeader != server.Header {
		return errors.Join(ErrApiHeaderMismatch, fmt.Errorf("expected %s but got %s", server.Header, alive.APIHeader))
	}

	return nil
}

// DataToValidate makes http request to the API server and returns data to validate.
func (r *Rest) DataToSign(address string) (server.DataToSignResponse, error) {
	var req server.DataToSignRequest
	var resp server.DataToSignResponse
	if err := r.makePost(server.DataToValidateURL, req, &resp); err != nil {
		return server.DataToSignResponse{}, err
	}
	return resp, nil
}
