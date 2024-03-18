package immunity

import (
	"context"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/bartossh/Computantis/src/transaction"
)

func generateTransactionWithSize(dataSize int) transaction.Transaction {
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
			trx := generateTransactionWithSize(c.actual)
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
			trx := generateTransactionWithSize(c.actual)
			antibody, err := NewTransactionSizeAntibody(c.min, c.max)
			assert.NilError(t, err)
			err = antibody.AnalyzeTransaction(context.TODO(), &trx)
			assert.ErrorContains(t, err, "got")
		})
	}
}

func BenchmarkTransactionSizeAntibody(b *testing.B) {
	trx := generateTransactionWithSize(1000)
	antibody, err := NewTransactionSizeAntibody(0, 10000)
	assert.NilError(b, err)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = antibody.AnalyzeTransaction(context.TODO(), &trx)
	}
}
