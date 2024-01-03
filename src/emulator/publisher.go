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

	"github.com/bartossh/Computantis/src/protobufcompiled"
	"github.com/bartossh/Computantis/src/spice"
)

type publisher struct {
	conn       *grpc.ClientConn
	client     protobufcompiled.WalletClientAPIClient
	position   int
	random     bool
	knownNodes []string
}

// RunPublisher runs publisher emulator that emulates data in a buffer.
// Running emulator is stopped by canceling context.
func RunPublisher(ctx context.Context, cancel context.CancelFunc, config Config, data []byte) error {
	defer cancel()

	var measurtements []Measurement
	if err := json.Unmarshal(data, &measurtements); err != nil {
		return fmt.Errorf("cannot unmarshal data, %s", err)
	}

	if config.TickMillisecond == 0 || config.TickMillisecond > 60000 {
		return fmt.Errorf("wrong tick_millisecond parameter, expected value between 1 and 60000 inclusive")
	}

	opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
	conn, err := grpc.Dial(config.ClientURL, opts)
	if err != nil {
		return fmt.Errorf("dial failed, %s", err)
	}
	defer conn.Close()
	client := protobufcompiled.NewWalletClientAPIClient(conn)

	p := publisher{
		conn:       conn,
		client:     client,
		random:     config.Random,
		knownNodes: config.NotaryNodes,
	}

	t := time.NewTicker(time.Duration(config.TickMillisecond) * time.Millisecond)
	tb := time.NewTicker(time.Duration(config.TickMillisecond*50) * time.Millisecond)
	defer t.Stop()
	defer tb.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			p.emulate(ctx, config.ReceiverPublicAddr, measurtements)
		case <-tb.C:
			addr, err := p.client.WalletPublicAddress(ctx, &emptypb.Empty{})
			if err != nil {
				pterm.Error.Printf("Publisher cannot validate public address, %s\n", err)
				continue
			}
			b, err := p.checkBalance(ctx)
			if err != nil {
				pterm.Error.Printf("Publisher [ %s ] emulator cannot check balance, %s\n", addr.Public, err)
				continue
			}
			pterm.Info.Printf(
				"Publisher emulator balance of account [ %s ] is %s \n",
				addr.Public,
				b.String(),
			)

		}
	}
}

func (p *publisher) emulate(ctx context.Context, receiver string, measurements []Measurement) {
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
		pterm.Error.Printf("Emulator cannot marshal data position [ %d ]\n", p.position+1)
		return
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
		pterm.Error.Printf("Emulator cannot send data position [ %d ], %s\n", p.position+1, err)
		return
	}
	m := measurements[p.position]
	pterm.Info.Printf("Emulated and published [ %d ] transaction [ %v | %v | %v ].\n", p.position+1, m.Mamps, m.Power, m.Volts)
}

func (pub *publisher) getRandomNodeURLFromList(notaryNodeURL string) string {
	if len(pub.knownNodes) > 0 {
		idx := rand.Intn(len(pub.knownNodes))
		notaryNodeURL = pub.knownNodes[idx]
	}
	return notaryNodeURL
}

func (pub *publisher) checkBalance(ctx context.Context) (spice.Melange, error) {
	s, err := pub.client.Balance(ctx, &emptypb.Empty{})
	if err != nil {
		return spice.Melange{}, err
	}
	return spice.Melange{Currency: s.Currency, SupplementaryCurrency: s.SuplementaryCurrency}, nil
}
