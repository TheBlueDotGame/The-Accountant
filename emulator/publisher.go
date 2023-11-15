package emulator

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/pterm/pterm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bartossh/Computantis/protobufcompiled"
)

type publisher struct {
	conn     *grpc.ClientConn
	client   protobufcompiled.WalletClientAPIClient
	position int
	random   bool
}

// RunPublisher runs publisher emulator that emulates data in a buffer.
// Running emmulator is stopped by canceling context.
func RunPublisher(ctx context.Context, cancel context.CancelFunc, config Config, data []byte) error {
	defer cancel()

	var measurtements []Measurement
	if err := json.Unmarshal(data, &measurtements); err != nil {
		return fmt.Errorf("cannot unmarshal data, %s", err)
	}

	if config.TickSeconds < 1 || config.TickSeconds > 60 {
		return fmt.Errorf("wrong tick_seconds parameter, expected value between 1 and 60 inclusive")
	}

	opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
	conn, err := grpc.Dial(config.ClientURL, opts)
	if err != nil {
		return fmt.Errorf("dial failed, %s", err)
	}
	defer conn.Close()
	client := protobufcompiled.NewWalletClientAPIClient(conn)

	p := publisher{
		conn:   conn,
		client: client,
		random: config.Random,
	}

	address, err := p.client.WalletPublicAddress(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}

	t := time.NewTicker(time.Duration(config.TickSeconds) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := p.emulate(ctx, address.Public, measurtements); err != nil {
				return err
			}
			pterm.Info.Printf("Emulated and published [ %d ] transaction from the given dataset.\n", p.position+1)
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

	var err error
	var data []byte
	data, err = json.Marshal(measurements[p.position])
	if err != nil {
		return err
	}
	if _, err := p.client.Issue(ctx, &protobufcompiled.IssueTrx{
		Subject:         fmt.Sprintf("measurement %v", p.position),
		ReceiverAddress: receiver,
		Data:            data,
		Spice: &protobufcompiled.Spice{
			Currency:             0,
			SuplementaryCurrency: 0,
		},
	}); err != nil {
		return err
	}
	return nil
}
