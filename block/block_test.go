package block

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/bartossh/Computantis/spice"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/wallet"
)

// TODO: (to improve security and for certification) get test vectors for test, fuzzy testing

func TestBlockCreate(t *testing.T) {
	difficulty := uint64(1)

	issuer, err := wallet.New()
	assert.NilError(t, err)
	receiver, err := wallet.New()
	assert.NilError(t, err)

	verifier := wallet.Helper{}

	message := []byte("genesis transaction")
	trx, err := transaction.New("genesis", spice.New(math.MaxInt64, 0), message, receiver.Address(), &issuer)
	assert.NilError(t, err)

	trxHash, err := trx.Sign(&receiver, verifier)
	assert.NilError(t, err)
	assert.Equal(t, len(trxHash), 32)

	blc := New(difficulty, 0, [32]byte{}, [][32]byte{trxHash})

	assert.Equal(t, len(blc.Hash), 32)

	blocksNum := 10
	blockchain := make([]Block, 0, blocksNum)
	blockchain = append(blockchain, blc)

	for i := 1; i <= blocksNum; i++ {
		nextMsg := []byte(fmt.Sprintf("next message: %v", i))

		ntrx, err := transaction.New("text", spice.New(math.MaxInt64, 0), nextMsg, receiver.Address(), &issuer)
		assert.NilError(t, err)

		ntrxHash, err := ntrx.Sign(&receiver, verifier)
		assert.NilError(t, err)
		newBlck := New(difficulty, uint64(i), blockchain[i-1].Hash, [][32]byte{ntrxHash})
		blockchain = append(blockchain, newBlck)
	}

	prevHash := blockchain[len(blockchain)-1].PrevHash
	for j := len(blockchain) - 2; j >= 0; j-- {
		ok := bytes.Equal(prevHash[:], blockchain[j].Hash[:])
		assert.Equal(t, ok, true)
		prevHash = blockchain[j].PrevHash
	}
}

func TestBlockValidateSuccess(t *testing.T) {
	difficulty := uint64(1)

	issuer, err := wallet.New()
	assert.NilError(t, err)
	receiver, err := wallet.New()
	assert.NilError(t, err)

	verifier := wallet.Helper{}

	message := []byte("genesis transaction")
	trx, err := transaction.New("genesis", spice.New(math.MaxInt64, 0), message, receiver.Address(), &issuer)
	assert.NilError(t, err)

	trxHash, err := trx.Sign(&receiver, verifier)
	assert.NilError(t, err)
	assert.Equal(t, len(trxHash), 32)

	blc := New(difficulty, 0, [32]byte{}, [][32]byte{trxHash})

	assert.Equal(t, len(blc.Hash), 32)

	blocksNum := 10
	blockchain := make([]Block, 0, blocksNum)
	blockchain = append(blockchain, blc)

	for i := 1; i <= blocksNum; i++ {
		trxs := make([][32]byte, 0, 1000)
		for j := 0; j < 1000; j++ {
			nextMsg := []byte(fmt.Sprintf("next message: %v%v", i, j))

			ntrx, err := transaction.New("text", spice.New(math.MaxInt64, 0), nextMsg, receiver.Address(), &issuer)
			assert.NilError(t, err)

			ntrxHash, err := ntrx.Sign(&receiver, verifier)
			assert.NilError(t, err)

			trxs = append(trxs, ntrxHash)
		}
		newBlck := New(difficulty, uint64(i), blockchain[i-1].Hash, trxs)

		blockchain = append(blockchain, newBlck)
	}

	prevHash := blockchain[len(blockchain)-1].PrevHash
	for j := len(blockchain) - 2; j >= 0; j-- {
		ok := bytes.Equal(prevHash[:], blockchain[j].Hash[:])
		assert.Equal(t, ok, true)
		prevHash = blockchain[j].PrevHash
		err := blockchain[j].Validate(blockchain[j].TrxHashes)
		assert.NilError(t, err)
	}
}

func TestBlockValidateFailure(t *testing.T) {
	difficulty := uint64(1)

	issuer, err := wallet.New()
	assert.NilError(t, err)
	receiver, err := wallet.New()
	assert.NilError(t, err)

	verifier := wallet.Helper{}

	message := []byte("genesis transaction")
	trx, err := transaction.New("genesis", spice.New(math.MaxInt64, 0), message, receiver.Address(), &issuer)
	assert.NilError(t, err)

	trxHash, err := trx.Sign(&receiver, verifier)
	assert.NilError(t, err)
	assert.Equal(t, len(trxHash), 32)

	blc := New(difficulty, 0, [32]byte{}, [][32]byte{trxHash})

	assert.Equal(t, len(blc.Hash), 32)

	blocksNum := 10
	blockchain := make([]Block, 0, blocksNum)
	blockchain = append(blockchain, blc)

	for i := 1; i <= blocksNum; i++ {
		trxs := make([][32]byte, 0, 1000)
		for j := 0; j < 1000; j++ {
			nextMsg := []byte(fmt.Sprintf("next message: %v%v", i, j))

			ntrx, err := transaction.New("text", spice.New(math.MaxInt64, 0), nextMsg, receiver.Address(), &issuer)
			assert.NilError(t, err)

			ntrxHash, err := ntrx.Sign(&receiver, verifier)
			assert.NilError(t, err)

			trxs = append(trxs, ntrxHash)
		}
		newBlck := New(difficulty, uint64(i), blockchain[i-1].Hash, trxs)
		blockchain = append(blockchain, newBlck)
	}

	prevHash := blockchain[len(blockchain)-1].PrevHash
	for j := len(blockchain) - 2; j >= 0; j-- {
		ok := bytes.Equal(prevHash[:], blockchain[j+1].Hash[:])
		assert.Equal(t, ok, false)
		prevHash = blockchain[j].PrevHash
		err := blockchain[j].Validate(blockchain[j+1].TrxHashes)
		if err == nil {
			t.Error("error is nil")
		}
	}
}

func Benchmark1000Blocks(b *testing.B) {
	difficulty := uint64(5)

	issuer, err := wallet.New()
	assert.NilError(b, err)
	receiver, err := wallet.New()
	assert.NilError(b, err)

	verifier := wallet.Helper{}

	for n := 0; n < b.N; n++ {
		message := []byte("genesis transaction")
		trx, err := transaction.New("genesis", spice.New(math.MaxInt64, 0), message, receiver.Address(), &issuer)
		assert.NilError(b, err)

		trxHash, err := trx.Sign(&receiver, verifier)
		assert.NilError(b, err)
		assert.Equal(b, len(trxHash), 32)

		blc := New(difficulty, 0, [32]byte{}, [][32]byte{trxHash})

		assert.Equal(b, len(blc.Hash), 32)

		blocksNum := 1000
		blockchain := make([]Block, 0, blocksNum)
		blockchain = append(blockchain, blc)

		for i := 1; i <= blocksNum; i++ {
			nextMsg := []byte(fmt.Sprintf("next message: %v", i))

			ntrx, err := transaction.New("text", spice.New(math.MaxInt64, 0), nextMsg, receiver.Address(), &issuer)
			assert.NilError(b, err)

			ntrxHash, err := ntrx.Sign(&receiver, verifier)
			assert.NilError(b, err)
			newBlck := New(difficulty, 1, blockchain[i-1].Hash, [][32]byte{ntrxHash})
			blockchain = append(blockchain, newBlck)
		}
	}
}

func Benchmark1_Block(b *testing.B) {
	difficulty := uint64(5)

	issuer, err := wallet.New()
	assert.NilError(b, err)
	receiver, err := wallet.New()
	assert.NilError(b, err)

	verifier := wallet.Helper{}

	for n := 0; n < b.N; n++ {
		message := []byte("genesis transaction")
		trx, err := transaction.New("genesis", spice.New(math.MaxInt64, 0), message, receiver.Address(), &issuer)
		assert.NilError(b, err)

		trxHash, err := trx.Sign(&receiver, verifier)
		assert.NilError(b, err)
		assert.Equal(b, len(trxHash), 32)

		New(difficulty, 0, [32]byte{}, [][32]byte{trxHash})
	}
}

func BenchmarkDifficulty(b *testing.B) {
	difficulties := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	issuer, err := wallet.New()
	assert.NilError(b, err)
	receiver, err := wallet.New()
	assert.NilError(b, err)

	verifier := wallet.Helper{}

	for _, difficulty := range difficulties {
		b.Run(fmt.Sprintf("difficulty: %v", difficulty), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				message := []byte("genesis transaction")
				trx, err := transaction.New("genesis", spice.New(math.MaxInt64, 0), message, receiver.Address(), &issuer)
				assert.NilError(b, err)

				trxHash, err := trx.Sign(&receiver, verifier)
				assert.NilError(b, err)
				assert.Equal(b, len(trxHash), 32)

				New(difficulty, 0, [32]byte{}, [][32]byte{trxHash})
			}
		})
	}
}

// This tests is has large difficulty and will take a lot of time to run
// The large difficulty is used to test for validation of small set of nonces that proves the work
func TestProofOfWorkSuccess(t *testing.T) {
	var difficulty uint64 = 23

	trxHashes := [][32]byte{{1, 2, 3, 4, 5}, {6, 7, 8, 9, 10}, {11, 12, 13, 14, 15}}

	for _, trxH := range trxHashes {
		t.Run(fmt.Sprintf("trx hashes %d", trxH[:5]), func(t *testing.T) {
			blc := Block{
				Difficulty: difficulty,
			}

			p := newProof(&blc)

			blc.Nonce, blc.Hash = p.run(trxH)

			ok := p.validate(trxH)
			assert.Equal(t, ok, true)
		})
	}
}

// This test will fail for small difficulty as it is much more possible nonces to be found
// It tests the case when the nonce is not valid for the given trx hashes and high difficulty
// It is tuned for difficulty 3, anything less will fail
func TestProofOfWorkFail(t *testing.T) {
	var difficulty uint64 = 3

	trxHashes := [][32]byte{{1, 2, 3, 4, 5}, {6, 7, 8, 9, 10}, {11, 12, 13, 14, 15}}

	for _, trxH := range trxHashes {
		t.Run(fmt.Sprintf("trx hash: %v", trxH[:5]), func(t *testing.T) {
			blc := Block{
				Difficulty: difficulty,
			}

			p := newProof(&blc)

			p.block.Nonce, p.block.Hash = p.run(trxH)

			trxH[0] = 255

			ok := p.validate(trxH)
			assert.Equal(t, ok, false)
		})
	}
}

// Fuzzing will take a long long time to run
func FuzzProofOfWorkSuccess(f *testing.F) {
	var difficulty uint64 = 3

	trxHashes := [][]byte{
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		{6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 7, 8, 9, 10},
		{11, 12, 13, 14, 15, 1, 2, 3, 4, 5, 6, 7, 8, 9, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 11, 12, 13, 14, 15, 16, 17, 18},
	}

	for _, trxH := range trxHashes {
		f.Add(trxH)
	}

	f.Fuzz(func(t *testing.T, trxH []byte) {
		blc := Block{
			Difficulty: difficulty,
		}

		p := newProof(&blc)

		var h [32]byte
		copy(h[:], trxH)

		p.block.Nonce, p.block.Hash = p.run(h)

		ok := p.validate(h)
		assert.Equal(t, ok, true)
	})
}
