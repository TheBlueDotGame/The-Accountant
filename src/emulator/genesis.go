package emulator

import (
	"context"
	"fmt"
	"time"

	"github.com/bartossh/Computantis/src/protobufcompiled"
	"github.com/pterm/pterm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type genesis struct {
	conn       *grpc.ClientConn
	client     protobufcompiled.WalletClientAPIClient
	knownNodes []string
}

func RunGenesis(ctx context.Context, cancel context.CancelFunc, config Config) error {
	defer cancel()

	if config.TickMillisecond == 0 || config.TickMillisecond > 120000 {
		return fmt.Errorf("wrong tick_millisecond parameter, expected value between 1 and 120000 inclusive")
	}

	opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
	conn, err := grpc.Dial(config.ClientURL, opts)
	if err != nil {
		return fmt.Errorf("dial failed, %s", err)
	}
	defer conn.Close()
	client := protobufcompiled.NewWalletClientAPIClient(conn)

	g := genesis{
		conn:       conn,
		client:     client,
		knownNodes: config.NotaryNodes,
	}

	t := time.NewTicker(time.Duration(config.TickMillisecond) * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := g.sendSpice(ctx, uint64(config.SpicePerTransaction), config.ReceiverPublicAddr); err != nil {
				pterm.Error.Printf(
					"Genesis emulator sending %v_spice to [ %s ] failed, %s.\n, err",
					config.SpicePerTransaction,
					config.ReceiverPublicAddr,
				)
				continue
			}
			pterm.Info.Printf(
				"Genesis emulator sending %v_spice to [ %s ] succedded.\n",
				config.SpicePerTransaction,
				config.ReceiverPublicAddr,
			)
		}
	}
}

func (g genesis) sendSpice(ctx context.Context, spice uint64, receiver string) error {
	_, err := g.client.Issue(ctx, &protobufcompiled.IssueTrx{
		Subject:         fmt.Sprintf("Spice transfer %v for %s", spice, receiver),
		ReceiverAddress: receiver,
		Data:            []byte{},
		Spice: &protobufcompiled.Spice{
			Currency:             spice,
			SuplementaryCurrency: 0,
		},
	})
	return err
}
