package httpclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"time"
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
	raw, err := json.Marshal(out)
	if err != nil {
		return err
	}
	req.SetBody(raw)

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
		return errors.Join(
			ErrStatusCodeMismatch,
			fmt.Errorf("expected status code %d but got %d", fasthttp.StatusOK, resp.StatusCode()))
	}

	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		return errors.Join(
			ErrContentTypeMismatch,
			fmt.Errorf("expected content type application/json but got %s", contentType))
	}

	return json.Unmarshal(resp.Body(), in)
}

func MakeGet(timeout time.Duration, url string, out any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.DoTimeout(req, resp, timeout); err != nil {
		return err
	}

	switch resp.StatusCode() {
	case fasthttp.StatusOK:
	case fasthttp.StatusNoContent:
		return nil
	default:
		return errors.Join(
			ErrStatusCodeMismatch,
			fmt.Errorf("expected status code %d but got %d", fasthttp.StatusOK, resp.StatusCode()))
	}

	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		return errors.Join(
			ErrContentTypeMismatch,
			fmt.Errorf("expected content type application/json but got %s", contentType))
	}

	return json.Unmarshal(resp.Body(), out)
}
