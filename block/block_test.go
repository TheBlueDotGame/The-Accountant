package block

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/wallet"
	"github.com/stretchr/testify/assert"
)

func TestBlockCreate(t *testing.T) {
	difficulty := uint64(1)

	issuer, err := wallet.New()
	assert.Nil(t, err)
	receiver, err := wallet.New()
	assert.Nil(t, err)

	verifier := wallet.Helper{}

	message := []byte("genesis transaction")
	trx, err := transaction.New("genesis", message, &issuer)
	assert.Nil(t, err)

	trxHash, err := trx.Sign(&receiver, verifier)
	assert.Nil(t, err)
	assert.NotEmpty(t, trxHash)

	blc := NewBlock(difficulty, 0, [32]byte{}, [][32]byte{trxHash})

	assert.NotEmpty(t, blc.Hash)

	blocksNum := 10000
	blockchain := make([]Block, 0, blocksNum)
	blockchain = append(blockchain, blc)

	for i := 1; i <= blocksNum; i++ {
		nextMsg := []byte(fmt.Sprintf("next message: %v", i))

		ntrx, err := transaction.New("text", nextMsg, &issuer)
		assert.Nil(t, err)

		ntrxHash, err := ntrx.Sign(&receiver, verifier)
		assert.Nil(t, err)
		newBlck := NewBlock(difficulty, uint64(i), blockchain[i-1].Hash, [][32]byte{ntrxHash})
		blockchain = append(blockchain, newBlck)
	}

	prevHash := blockchain[len(blockchain)-1].PrevHash
	for j := len(blockchain) - 2; j >= 0; j-- {
		ok := bytes.Equal(prevHash[:], blockchain[j].Hash[:])
		assert.True(t, ok)
		prevHash = blockchain[j].PrevHash
	}
}

func TestBlockValidateSuccess(t *testing.T) {
	difficulty := uint64(1)

	issuer, err := wallet.New()
	assert.Nil(t, err)
	receiver, err := wallet.New()
	assert.Nil(t, err)

	verifier := wallet.Helper{}

	message := []byte("genesis transaction")
	trx, err := transaction.New("genesis", message, &issuer)
	assert.Nil(t, err)

	trxHash, err := trx.Sign(&receiver, verifier)
	assert.Nil(t, err)
	assert.NotEmpty(t, trxHash)

	blc := NewBlock(difficulty, 0, [32]byte{}, [][32]byte{trxHash})

	assert.NotEmpty(t, blc.Hash)

	blocksNum := 10000
	blockchain := make([]Block, 0, blocksNum)
	blockchain = append(blockchain, blc)

	for i := 1; i <= blocksNum; i++ {
		nextMsg := []byte(fmt.Sprintf("next message: %v", i))

		ntrx, err := transaction.New("text", nextMsg, &issuer)
		assert.Nil(t, err)

		ntrxHash, err := ntrx.Sign(&receiver, verifier)
		assert.Nil(t, err)
		newBlck := NewBlock(difficulty, uint64(i), blockchain[i-1].Hash, [][32]byte{ntrxHash})
		blockchain = append(blockchain, newBlck)
	}

	prevHash := blockchain[len(blockchain)-1].PrevHash
	for j := len(blockchain) - 2; j >= 0; j-- {
		ok := bytes.Equal(prevHash[:], blockchain[j].Hash[:])
		assert.True(t, ok)
		prevHash = blockchain[j].PrevHash
		ok = blockchain[j].Validate(blockchain[j].TrxHashes)
		assert.True(t, ok)
	}
}

func TestBlockValidateFailure(t *testing.T) {
	difficulty := uint64(1)

	issuer, err := wallet.New()
	assert.Nil(t, err)
	receiver, err := wallet.New()
	assert.Nil(t, err)

	verifier := wallet.Helper{}

	message := []byte("genesis transaction")
	trx, err := transaction.New("genesis", message, &issuer)
	assert.Nil(t, err)

	trxHash, err := trx.Sign(&receiver, verifier)
	assert.Nil(t, err)
	assert.NotEmpty(t, trxHash)

	blc := NewBlock(difficulty, 0, [32]byte{}, [][32]byte{trxHash})

	assert.NotEmpty(t, blc.Hash)

	blocksNum := 10000
	blockchain := make([]Block, 0, blocksNum)
	blockchain = append(blockchain, blc)

	for i := 1; i <= blocksNum; i++ {
		nextMsg := []byte(fmt.Sprintf("next message: %v", i))

		ntrx, err := transaction.New("text", nextMsg, &issuer)
		assert.Nil(t, err)

		ntrxHash, err := ntrx.Sign(&receiver, verifier)
		assert.Nil(t, err)
		newBlck := NewBlock(difficulty, uint64(i), blockchain[i-1].Hash, [][32]byte{ntrxHash})
		blockchain = append(blockchain, newBlck)
	}

	prevHash := blockchain[len(blockchain)-1].PrevHash
	for j := len(blockchain) - 2; j >= 0; j-- {
		ok := bytes.Equal(prevHash[:], blockchain[j+1].Hash[:])
		assert.False(t, ok)
		prevHash = blockchain[j].PrevHash
		ok = blockchain[j].Validate(blockchain[j+1].TrxHashes)
		assert.False(t, ok)
	}
}

func Benchmark1000Blocks(b *testing.B) {
	difficulty := uint64(5)

	issuer, err := wallet.New()
	assert.Nil(b, err)
	receiver, err := wallet.New()
	assert.Nil(b, err)

	verifier := wallet.Helper{}

	for n := 0; n < b.N; n++ {
		message := []byte("genesis transaction")
		trx, err := transaction.New("genesis", message, &issuer)
		assert.Nil(b, err)

		trxHash, err := trx.Sign(&receiver, verifier)
		assert.Nil(b, err)
		assert.NotEmpty(b, trxHash)

		blc := NewBlock(difficulty, 0, [32]byte{}, [][32]byte{trxHash})

		assert.NotEmpty(b, blc.Hash)

		blocksNum := 1000
		blockchain := make([]Block, 0, blocksNum)
		blockchain = append(blockchain, blc)

		for i := 1; i <= blocksNum; i++ {
			nextMsg := []byte(fmt.Sprintf("next message: %v", i))

			ntrx, err := transaction.New("text", nextMsg, &issuer)
			assert.Nil(b, err)

			ntrxHash, err := ntrx.Sign(&receiver, verifier)
			assert.Nil(b, err)
			newBlck := NewBlock(difficulty, 1, blockchain[i-1].Hash, [][32]byte{ntrxHash})
			blockchain = append(blockchain, newBlck)
		}
	}
}

func Benchmark1_Block(b *testing.B) {
	difficulty := uint64(5)

	issuer, err := wallet.New()
	assert.Nil(b, err)
	receiver, err := wallet.New()
	assert.Nil(b, err)

	verifier := wallet.Helper{}

	for n := 0; n < b.N; n++ {
		message := []byte("genesis transaction")
		trx, err := transaction.New("genesis", message, &issuer)
		assert.Nil(b, err)

		trxHash, err := trx.Sign(&receiver, verifier)
		assert.Nil(b, err)
		assert.NotEmpty(b, trxHash)

		NewBlock(difficulty, 0, [32]byte{}, [][32]byte{trxHash})
	}
}
