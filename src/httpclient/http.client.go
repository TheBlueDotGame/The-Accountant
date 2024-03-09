package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/valyala/fasthttp"
)

var (
	ErrApiVersionMismatch            = fmt.Errorf("api version mismatch")
	ErrApiHeaderMismatch             = fmt.Errorf("api header mismatch")
	ErrStatusCodeMismatch            = fmt.Errorf("status code mismatch")
	ErrContentTypeMismatch           = fmt.Errorf("content type mismatch")
	ErrWalletChecksumMismatch        = fmt.Errorf("wallet checksum mismatch")
	ErrWalletVersionMismatch         = fmt.Errorf("wallet version mismatch")
	ErrServerReturnsInconsistentData = fmt.Errorf("server returns inconsistent data")
	ErrRejectedByServer              = fmt.Errorf("rejected by server")
	ErrWalletNotReady                = fmt.Errorf("wallet not ready, read wallet first")
	ErrSigningFailed                 = fmt.Errorf("signing failed")
)

func MakePost(timeout time.Duration, url string, out, in any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")

	return makePost(req, timeout, out, in)
}

func MakeGet(timeout time.Duration, url string, in any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	req.Header.SetMethod("GET")

	return makeGet(req, timeout, in)
}

// MakePostAuth make a post request with serialized 'out' structure which is send to the given 'url' with authorization token
// 'in' is a pointer to the structure to be deserialized from the received json data.
func MakePostAuth(timeout time.Duration, token, url string, out, in any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Set("accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	return makePost(req, timeout, out, in)
}

// MakeGetAuth make a get request to the given 'url' with authorization token
// 'in' is a pointer to the structure to be deserialized from the received json data.
func MakeGetAuth(timeout time.Duration, token, url string, in any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	req.Header.SetMethod("GET")
	req.Header.SetContentType("application/json")
	req.Header.Set("accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	return makeGet(req, timeout, in)
}

func makePost(req *fasthttp.Request, timeout time.Duration, out, in any) error {
	raw, err := json.Marshal(out)
	if err != nil {
		return err
	}
	req.SetBody(raw)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.DoTimeout(req, resp, timeout); err != nil {
		return fmt.Errorf("request failed %s", err)
	}

	switch resp.StatusCode() {
	case fasthttp.StatusOK, fasthttp.StatusCreated, fasthttp.StatusAccepted:
	case fasthttp.StatusNoContent:
		return nil
	default:
		return fmt.Errorf("request failed, expected status code %d but got %d", fasthttp.StatusOK, resp.StatusCode())
	}

	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		return fmt.Errorf("request failed, expected content type application/json but got %s", contentType)
	}

	if in != nil {
		return json.Unmarshal(resp.Body(), in)
	}
	return nil
}

func makeGet(req *fasthttp.Request, timeout time.Duration, in any) error {
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.DoTimeout(req, resp, timeout); err != nil {
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

	if in != nil {
		return json.Unmarshal(resp.Body(), in)
	}
	return nil
}
