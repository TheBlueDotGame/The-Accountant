package emulator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/signerservice"
	"github.com/pterm/pterm"
	"github.com/valyala/fasthttp"
)

type publisher struct {
	timeout       time.Duration
	signerAPIRoot string
	random        bool
	position      int
}

// RunPublisher runs publisher emulator that emulates data in a buffer.
// Running emmulator is stopped by canceling context.
func RunPublisher(ctx context.Context, config Config, data [][]byte) error {
	if config.TimeoutSeconds < 1 || config.TimeoutSeconds > 20 {
		return fmt.Errorf("wrong timeout_seconds parameter, expected value between 1 and 20 inclusive")
	}

	if config.TickSeconds < 1 || config.TickSeconds > 60 {
		return fmt.Errorf("wrong tick_seconds parameter, expected value between 1 and 60 inclusive")
	}

	p := publisher{
		timeout:       time.Second * time.Duration(config.TimeoutSeconds),
		signerAPIRoot: config.SignerServiceURL,
		random:        config.Random,
	}

	var alive signerservice.AliveResponse
	if err := p.makeGet(signerservice.Alive, &alive); err != nil {
		return err
	}
	if alive.APIVersion != server.ApiVersion || alive.APIHeader != server.Header {
		return fmt.Errorf(
			"emulation not possible due to wrong headers and/or version, expected header %s, version %s, received header %s, version %s",
			server.Header, server.ApiVersion, alive.APIHeader, alive.APIVersion)
	}

	var addr signerservice.AddressResponse
	if err := p.makeGet(signerservice.Address, &addr); err != nil {
		return fmt.Errorf("cannot read public address, %s", err)
	}

	t := time.NewTicker(time.Duration(config.TickSeconds) * time.Second)
	defer t.Stop()
	spinner, _ := pterm.DefaultSpinner.Start("Emulating transaction publisher...")
	defer spinner.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			spinner, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("Making [ %d ] transaction emulation.\n", p.position+1))
			if err := p.emulate(ctx, addr.Address, data); err != nil {
				spinner.Warning()
				return err
			}
			spinner.Success()
		}
	}
}

func (p *publisher) emulate(ctx context.Context, receiver string, data [][]byte) error {
	switch p.random {
	case true:
		p.position = rand.Intn(len(data))
	default:
		p.position++
	}

	if p.position >= len(data) {
		p.position = 0
	}

	t := time.NewTimer(time.Second * time.Duration(p.timeout))
	defer t.Stop()
	d := make(chan struct{}, 1)

	var err error
	go func() {
		defer func() {
			d <- struct{}{}
		}()
		req := signerservice.IssueTransactionRequest{
			ReceiverAddress: receiver,
			Subject:         "emulator-test",
			Data:            data[p.position],
		}
		var resp signerservice.IssueTransactionResponse
		err = p.makePost(signerservice.IssueTransaction, req, &resp)
		if resp.Err != "" {
			err = errors.New(resp.Err)
			return
		}
		if !resp.Ok {
			err = errors.New("unexpected error")
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case <-d:
		return err
	case <-t.C:
		return errors.New("timeout")
	}
}

func (p *publisher) makePost(path string, out, in any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(path)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	raw, err := json.Marshal(out)
	if err != nil {
		return err
	}
	req.SetBody(raw)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.DoTimeout(req, resp, p.timeout); err != nil {
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

func (p *publisher) makeGet(path string, out any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(fmt.Sprintf("%s/%s", p.signerAPIRoot, path))
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.DoTimeout(req, resp, p.timeout); err != nil {
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
