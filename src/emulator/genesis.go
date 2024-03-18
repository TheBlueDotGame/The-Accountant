package emulator

import (
	"context"
	"fmt"
	"time"

	"github.com/bartossh/Computantis/src/protobufcompiled"
	"github.com/bartossh/Computantis/src/spice"
	"github.com/pterm/pterm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type genesis struct {
	conn       *grpc.ClientConn
	client     protobufcompiled.WalletClientAPIClient
	knownNodes []string
}

func RunGenesis(ctx context.Context, cancel context.CancelFunc, config Config) error {
	if config.SleepInSecBeforeStart > 0 {
		time.Sleep(time.Duration(config.SleepInSecBeforeStart))
	}
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
					"Genesis emulator sending %v_spice to [ %s ] failed, %s.\n",
					config.SpicePerTransaction,
					config.ReceiverPublicAddr,
					err,
				)
				continue
			}
			pterm.Info.Printf(
				"Genesis emulator sending %v_spice to [ %s ] succedded.\n",
				config.SpicePerTransaction,
				config.ReceiverPublicAddr,
			)
			addr, err := g.client.WalletPublicAddress(ctx, &emptypb.Empty{})
			if err != nil {
				pterm.Error.Printf("Genesis cannot validate public address, %s\n", err)
				continue
			}
			b, err := g.checkBalance(ctx)
			if err != nil {
				pterm.Error.Printf("Genesis [ %s ] emulator cannot check balance, %s\n", addr.Public, err)
				continue
			}
			pterm.Info.Printf(
				"Genesis emulator balance of account [ %s ] is %s \n",
				addr.Public,
				b.String(),
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
			Currency:              spice,
			SupplementaryCurrency: 0,
		},
	})
	return err
}

func (g genesis) checkBalance(ctx context.Context) (spice.Melange, error) {
	s, err := g.client.Balance(ctx, &emptypb.Empty{})
	if err != nil {
		return spice.Melange{}, err
	}
	return spice.Melange{Currency: s.Currency, SupplementaryCurrency: s.SupplementaryCurrency}, nil
}
