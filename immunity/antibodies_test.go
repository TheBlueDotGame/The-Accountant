package immunity

import (
	"context"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/transaction"
)

func generateTrasactionWithSize(dataSize int) transaction.Transaction {
	return transaction.Transaction{
		Subject: "",
		Data:    make([]byte, dataSize),
	}
}

func TestTransactionSizeAntibodyFailToCreateNew(t *testing.T) {
	testCases := []struct {
		min, max int
	}{
		{min: -1, max: 10},
		{min: 11, max: 10},
		{min: -100, max: -111},
		{min: 10, max: 0},
	}

	for i, c := range testCases {
		t.Run(fmt.Sprintf("test case %v max %v min %v", i, c.min, c.max), func(t *testing.T) {
			_, err := NewTransactionSizeAntibody(c.min, c.max)
			assert.ErrorContains(t, err, "min value")
		})
	}
}

func TestTransactionSizeAntibodySuccess(t *testing.T) {
	testCases := []struct {
		min, max, actual int
	}{
		{min: 0, max: 10, actual: 5},
		{min: 1, max: 11, actual: 1},
		{min: 2, max: 2, actual: 2},
		{min: 0, max: 10, actual: 10},
		{min: 9, max: 10, actual: 9},
	}

	for i, c := range testCases {
		t.Run(fmt.Sprintf("test case %v max %v min %v actual %v", i, c.min, c.max, c.actual), func(t *testing.T) {
			trx := generateTrasactionWithSize(c.actual)
			antibody, err := NewTransactionSizeAntibody(c.min, c.max)
			assert.NilError(t, err, fmt.Sprintf("test case %v", i))
			err = antibody.AnalyzeTransaction(context.TODO(), &trx)
			assert.NilError(t, err, fmt.Sprintf("test case %v", i))
		})
	}
}

func TestTransactionSizeAntibodyFailure(t *testing.T) {
	testCases := []struct {
		min, max, actual int
	}{
		{min: 0, max: 10, actual: 11},
		{min: 1, max: 11, actual: 0},
		{min: 2, max: 2, actual: 3},
		{min: 0, max: 10, actual: 10000},
		{min: 9, max: 10, actual: 8},
	}

	for i, c := range testCases {
		t.Run(fmt.Sprintf("test case %v max %v min %v actual %v", i, c.min, c.max, c.actual), func(t *testing.T) {
			trx := generateTrasactionWithSize(c.actual)
			antibody, err := NewTransactionSizeAntibody(c.min, c.max)
			assert.NilError(t, err)
			err = antibody.AnalyzeTransaction(context.TODO(), &trx)
			assert.ErrorContains(t, err, "got")
		})
	}
}

func BenchmarkTransactionSizeAntibody(b *testing.B) {
	trx := generateTrasactionWithSize(1000)
	antibody, err := NewTransactionSizeAntibody(0, 10000)
	assert.NilError(b, err)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = antibody.AnalyzeTransaction(context.TODO(), &trx)
	}
}

func TestBlockConsecutiveOrderAntibodyFailToCreate(t *testing.T) {
	var nilAntibody *BlockConsecutiveOrderAntibody
	antibody, err := NewBlockConsecutiveOrderAntibody(&block.Block{})
	assert.Equal(t, antibody, nilAntibody)
	assert.ErrorContains(t, err, "previous hash")
}

func TestBlockConsecutiveOrderAntibodySuccess(t *testing.T) {
	testCasesConsecutive := []*block.Block{
		{Hash: [32]byte{1}, Index: 0, PrevHash: [32]byte{0, 1}},
		{Hash: [32]byte{2}, Index: 1, PrevHash: [32]byte{1}},
		{Hash: [32]byte{3}, Index: 2, PrevHash: [32]byte{2}},
		{Hash: [32]byte{4}, Index: 3, PrevHash: [32]byte{3}},
		{Hash: [32]byte{5}, Index: 4, PrevHash: [32]byte{4}},
		{Hash: [32]byte{6}, Index: 5, PrevHash: [32]byte{5}},
		{Hash: [32]byte{7}, Index: 6, PrevHash: [32]byte{6}},
		{Hash: [32]byte{8}, Index: 7, PrevHash: [32]byte{7}},
		{Hash: [32]byte{9}, Index: 8, PrevHash: [32]byte{8}},
		{Hash: [32]byte{10}, Index: 9, PrevHash: [32]byte{9}},
		{Hash: [32]byte{11}, Index: 10, PrevHash: [32]byte{10}},
		{Hash: [32]byte{12}, Index: 11, PrevHash: [32]byte{11}},
		{Hash: [32]byte{13}, Index: 12, PrevHash: [32]byte{12}},
		{Hash: [32]byte{14}, Index: 13, PrevHash: [32]byte{13}},
		{Hash: [32]byte{15}, Index: 14, PrevHash: [32]byte{14}},
		{Hash: [32]byte{16}, Index: 15, PrevHash: [32]byte{15}},
		{Hash: [32]byte{17}, Index: 16, PrevHash: [32]byte{16}},
		{Hash: [32]byte{18}, Index: 17, PrevHash: [32]byte{17}},
		{Hash: [32]byte{19}, Index: 18, PrevHash: [32]byte{18}},
		{Hash: [32]byte{20}, Index: 19, PrevHash: [32]byte{19}},
		{Hash: [32]byte{21}, Index: 20, PrevHash: [32]byte{20}},
	}

	antibody, err := NewBlockConsecutiveOrderAntibody(testCasesConsecutive[0])
	assert.NilError(t, err)

	for i, c := range testCasesConsecutive[1:] {
		t.Run(fmt.Sprintf("index: %v", c.Index), func(t *testing.T) {
			err := antibody.AnalyzeBlock(context.TODO(), c, nil)
			assert.NilError(t, err, fmt.Sprintf("test case %v", i))
		})
	}
}

func TestBlockConsecutiveOrderAntibodyFailIndex(t *testing.T) {
	testCasesConsecutive := []*block.Block{
		{Hash: [32]byte{1}, Index: 0, PrevHash: [32]byte{0, 1}},
		{Hash: [32]byte{2}, Index: 2, PrevHash: [32]byte{1}},
		{Hash: [32]byte{3}, Index: 3, PrevHash: [32]byte{2}},
		{Hash: [32]byte{4}, Index: 4, PrevHash: [32]byte{3}},
		{Hash: [32]byte{5}, Index: 5, PrevHash: [32]byte{4}},
		{Hash: [32]byte{6}, Index: 6, PrevHash: [32]byte{5}},
		{Hash: [32]byte{7}, Index: 7, PrevHash: [32]byte{6}},
		{Hash: [32]byte{8}, Index: 8, PrevHash: [32]byte{7}},
		{Hash: [32]byte{9}, Index: 9, PrevHash: [32]byte{8}},
		{Hash: [32]byte{10}, Index: 10, PrevHash: [32]byte{9}},
		{Hash: [32]byte{11}, Index: 11, PrevHash: [32]byte{10}},
		{Hash: [32]byte{12}, Index: 12, PrevHash: [32]byte{11}},
		{Hash: [32]byte{13}, Index: 13, PrevHash: [32]byte{12}},
		{Hash: [32]byte{14}, Index: 14, PrevHash: [32]byte{13}},
		{Hash: [32]byte{15}, Index: 15, PrevHash: [32]byte{14}},
		{Hash: [32]byte{16}, Index: 16, PrevHash: [32]byte{15}},
		{Hash: [32]byte{17}, Index: 17, PrevHash: [32]byte{16}},
		{Hash: [32]byte{18}, Index: 18, PrevHash: [32]byte{17}},
		{Hash: [32]byte{19}, Index: 19, PrevHash: [32]byte{18}},
		{Hash: [32]byte{20}, Index: 20, PrevHash: [32]byte{19}},
		{Hash: [32]byte{21}, Index: 21, PrevHash: [32]byte{20}},
	}

	antibody, err := NewBlockConsecutiveOrderAntibody(testCasesConsecutive[0])
	assert.NilError(t, err)

	for _, c := range testCasesConsecutive[1:] {
		t.Run(fmt.Sprintf("index: %v", c.Index), func(t *testing.T) {
			err := antibody.AnalyzeBlock(context.TODO(), c, nil)
			assert.ErrorContains(t, err, "expected block index")
		})
	}
}

func TestBlockConsecutiveOrderAntibodyFailPreviousHash(t *testing.T) {
	testCasesConsecutive := []*block.Block{
		{Hash: [32]byte{1}, Index: 0, PrevHash: [32]byte{0, 1}},
		{Hash: [32]byte{2}, Index: 1, PrevHash: [32]byte{2}},
		{Hash: [32]byte{3}, Index: 2, PrevHash: [32]byte{3}},
		{Hash: [32]byte{4}, Index: 3, PrevHash: [32]byte{4}},
		{Hash: [32]byte{5}, Index: 4, PrevHash: [32]byte{5}},
		{Hash: [32]byte{6}, Index: 5, PrevHash: [32]byte{6}},
		{Hash: [32]byte{7}, Index: 6, PrevHash: [32]byte{7}},
		{Hash: [32]byte{8}, Index: 7, PrevHash: [32]byte{8}},
		{Hash: [32]byte{9}, Index: 8, PrevHash: [32]byte{9}},
		{Hash: [32]byte{10}, Index: 9, PrevHash: [32]byte{10}},
		{Hash: [32]byte{11}, Index: 10, PrevHash: [32]byte{11}},
		{Hash: [32]byte{12}, Index: 11, PrevHash: [32]byte{12}},
		{Hash: [32]byte{13}, Index: 12, PrevHash: [32]byte{13}},
		{Hash: [32]byte{14}, Index: 13, PrevHash: [32]byte{14}},
		{Hash: [32]byte{15}, Index: 14, PrevHash: [32]byte{15}},
		{Hash: [32]byte{16}, Index: 15, PrevHash: [32]byte{16}},
		{Hash: [32]byte{17}, Index: 16, PrevHash: [32]byte{17}},
		{Hash: [32]byte{18}, Index: 17, PrevHash: [32]byte{18}},
		{Hash: [32]byte{19}, Index: 18, PrevHash: [32]byte{19}},
		{Hash: [32]byte{20}, Index: 19, PrevHash: [32]byte{20}},
		{Hash: [32]byte{21}, Index: 20, PrevHash: [32]byte{21}},
	}

	antibody, err := NewBlockConsecutiveOrderAntibody(testCasesConsecutive[0])
	assert.NilError(t, err)

	for _, c := range testCasesConsecutive[1:] {
		t.Run(fmt.Sprintf("index: %v", c.Index), func(t *testing.T) {
			err := antibody.AnalyzeBlock(context.TODO(), c, nil)
			assert.ErrorContains(t, err, "expected")
		})
	}
}
