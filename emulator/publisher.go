package emulator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/pterm/pterm"

	"github.com/bartossh/Computantis/httpclient"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/walletapi"
)

type publisher struct {
	clientURL string
	timeout   time.Duration
	position  int
	random    bool
}

// RunPublisher runs publisher emulator that emulates data in a buffer.
// Running emmulator is stopped by canceling context.
func RunPublisher(ctx context.Context, cancel context.CancelFunc, config Config, data []byte) error {
	defer cancel()

	var measurtements []Measurement
	if err := json.Unmarshal(data, &measurtements); err != nil {
		return fmt.Errorf("cannot unmarshal data, %s", err)
	}

	if config.TimeoutSeconds < 1 || config.TimeoutSeconds > 20 {
		return fmt.Errorf("wrong timeout_seconds parameter, expected value between 1 and 20 inclusive")
	}

	if config.TickSeconds < 1 || config.TickSeconds > 60 {
		return fmt.Errorf("wrong tick_seconds parameter, expected value between 1 and 60 inclusive")
	}

	p := publisher{
		timeout:   time.Second * time.Duration(config.TimeoutSeconds),
		clientURL: config.ClientURL,
		random:    config.Random,
	}

	var alive walletapi.AliveResponse
	url := fmt.Sprintf("%s%s", p.clientURL, walletapi.Alive)
	if err := httpclient.MakeGet(p.timeout, url, &alive); err != nil {
		return err
	}
	if alive.APIVersion != server.ApiVersion || alive.APIHeader != server.Header {
		return fmt.Errorf(
			"emulation not possible due to wrong headers and/or version, expected header %s, version %s, received header %s, version %s",
			server.Header, server.ApiVersion, alive.APIHeader, alive.APIVersion)
	}

	var addr walletapi.AddressResponse
	url = fmt.Sprintf("%s%s", p.clientURL, walletapi.Address)
	if err := httpclient.MakeGet(p.timeout, url, &addr); err != nil {
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
			if err := p.emulate(ctx, addr.Address, measurtements); err != nil {
				spinner.Warning()
				return err
			}
			spinner.Success()
		}
	}
}

func (p *publisher) emulate(ctx context.Context, receiver string, measurements []Measurement) error {
	switch p.random {
	case true:
		p.position = rand.Intn(len(measurements))
	default:
		p.position++
	}

	if p.position >= len(measurements) {
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
		var data []byte
		data, err = json.Marshal(measurements[p.position])
		if err != nil {
			return
		}
		req := walletapi.IssueTransactionRequest{
			ReceiverAddress: receiver,
			Subject:         "emulator-test",
			Data:            data,
		}
		var resp walletapi.IssueTransactionResponse
		url := fmt.Sprintf("%s%s", p.clientURL, walletapi.IssueTransaction)
		err = httpclient.MakePost(p.timeout, url, req, &resp)
		if resp.Err != "" {
			err = errors.New(resp.Err)
			return
		}
		if !resp.Ok {
			err = errors.New("unexpected error")
		}

		pterm.Info.Printf("Emulated measuremnt: %#v.", measurements[p.position])
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
