package spice

import (
	"fmt"
	"math"
	"testing"

	"gotest.tools/v3/assert"
)

func TestStringifyMelange(t *testing.T) {
	testcases := []Melange{
		{Currency: 10, SupplementaryCurrency: maxAmoutnPerSupplementaryCurrency},
		{Currency: 100, SupplementaryCurrency: maxAmoutnPerSupplementaryCurrency - 1},
		{Currency: 1000, SupplementaryCurrency: maxAmoutnPerSupplementaryCurrency - 10},
		{Currency: 10000, SupplementaryCurrency: maxAmoutnPerSupplementaryCurrency - 100},
		{Currency: 100000, SupplementaryCurrency: maxAmoutnPerSupplementaryCurrency - 1000},
		{Currency: 1000000, SupplementaryCurrency: maxAmoutnPerSupplementaryCurrency - 10000},
		{Currency: 1, SupplementaryCurrency: maxAmoutnPerSupplementaryCurrency - 100000},
		{Currency: 0, SupplementaryCurrency: 0},
		{Currency: 0, SupplementaryCurrency: maxAmoutnPerSupplementaryCurrency / 10},
		{Currency: 0, SupplementaryCurrency: maxAmoutnPerSupplementaryCurrency / 100},
		{Currency: 0, SupplementaryCurrency: maxAmoutnPerSupplementaryCurrency / 1000},
		{Currency: 0, SupplementaryCurrency: maxAmoutnPerSupplementaryCurrency / 10000},
	}

	results := []string{
		"10.",
		"100.999999999999999999",
		"1000.99999999999999999",
		"10000.9999999999999999",
		"100000.999999999999999",
		"1000000.99999999999999",
		"1.9999999999999",
		".",
		".1",
		".01",
		".001",
		".0001",
	}

	for i, c := range testcases {
		t.Run(fmt.Sprintf("test case %v", i), func(t *testing.T) {
			assert.Equal(t, c.String(), results[i])
		})
	}
}

func TestMelangeTransferSuccess(t *testing.T) {
	testcases := []struct {
		from, to, ammount, fromResult, toResult Melange
	}{
		{from: New(1, 10), to: New(1, 10), ammount: New(1, 10), fromResult: New(0, 0), toResult: New(2, 20)},
		{from: New(100, 1000), to: New(1, 10), ammount: New(1, 10), fromResult: New(99, 990), toResult: New(2, 20)},
		{from: New(1000, 1000), to: New(100, 1000), ammount: New(100, 1000), fromResult: New(900, 0), toResult: New(200, 2000)},
		{
			from: New(100, maxAmoutnPerSupplementaryCurrency-1), to: New(100, 1), ammount: New(100, maxAmoutnPerSupplementaryCurrency-1),
			fromResult: New(0, 0), toResult: New(201, 0),
		},
		{
			from: New(200, 0), to: New(100, maxAmoutnPerSupplementaryCurrency-1), ammount: New(100, maxAmoutnPerSupplementaryCurrency-1),
			fromResult: New(99, 1), toResult: New(201, maxAmoutnPerSupplementaryCurrency-2),
		},
		{
			from: New(200, 0), to: New(100, maxAmoutnPerSupplementaryCurrency-10), ammount: New(100, maxAmoutnPerSupplementaryCurrency-10),
			fromResult: New(99, 10), toResult: New(201, maxAmoutnPerSupplementaryCurrency-20),
		},
		{
			from: New(1, 0), to: New(100, maxAmoutnPerSupplementaryCurrency-10), ammount: New(0, maxAmoutnPerSupplementaryCurrency-10),
			fromResult: New(0, 10), toResult: New(101, maxAmoutnPerSupplementaryCurrency-20),
		},
	}

	for i, c := range testcases {
		t.Run(fmt.Sprintf("test case %v", i), func(t *testing.T) {
			err := Transfer(c.ammount, &c.from, &c.to)
			assert.NilError(t, err, fmt.Sprintf("cases number %v", i))
			assert.Equal(t, c.from.Currency, c.fromResult.Currency)
			assert.Equal(t, c.to.Currency, c.toResult.Currency)
			assert.Equal(t, c.from.SupplementaryCurrency, c.fromResult.SupplementaryCurrency)
			assert.Equal(t, c.to.SupplementaryCurrency, c.toResult.SupplementaryCurrency)
		})
	}
}

func TestMelangeTransferFailure(t *testing.T) {
	testcases := []struct {
		err                                     error
		from, to, ammount, fromResult, toResult Melange
	}{
		{
			from: New(math.MaxUint64, 10), to: New(1, 10), ammount: New(math.MaxUint64, 10),
			fromResult: New(math.MaxUint64, 10), toResult: New(1, 10), err: ErrValueOverflow,
		},
		{
			from: New(math.MaxUint64, 10), to: New(math.MaxUint64, 10), ammount: New(1, 10),
			fromResult: New(math.MaxUint64, 10), toResult: New(math.MaxUint64, 10), err: ErrValueOverflow,
		},
		{
			from: New(math.MaxUint64, 10), to: New(math.MaxUint64, maxAmoutnPerSupplementaryCurrency-1), ammount: New(0, 10),
			fromResult: New(math.MaxUint64, 10), toResult: New(math.MaxUint64, maxAmoutnPerSupplementaryCurrency-1), err: ErrValueOverflow,
		},
		{
			from: New(10, 10), to: New(1, 10), ammount: New(11, 10),
			fromResult: New(10, 10), toResult: New(1, 10), err: ErrNoSufficientFounds,
		},
		{
			from: New(0, 10), to: New(1, 10), ammount: New(0, 11),
			fromResult: New(0, 10), toResult: New(1, 10), err: ErrNoSufficientFounds,
		},
		{
			from: New(10000000000000, 10), to: New(1, 10), ammount: New(10000000000000, 11),
			fromResult: New(10000000000000, 10), toResult: New(1, 10), err: ErrNoSufficientFounds,
		},
	}

	for i, c := range testcases {
		t.Run(fmt.Sprintf("test case %v", i), func(t *testing.T) {
			err := Transfer(c.ammount, &c.from, &c.to)
			assert.ErrorIs(t, err, c.err)
			assert.Equal(t, c.from.Currency, c.fromResult.Currency)
			assert.Equal(t, c.to.Currency, c.toResult.Currency)
			assert.Equal(t, c.from.SupplementaryCurrency, c.fromResult.SupplementaryCurrency)
			assert.Equal(t, c.to.SupplementaryCurrency, c.toResult.SupplementaryCurrency)
		})
	}
}

func TestMelangeTransferAccounting(t *testing.T) {
	testcase := struct {
		from, to, ammount Melange
	}{
		from: New(math.MaxUint64, maxAmoutnPerSupplementaryCurrency), to: New(0, 0), ammount: New(1, 1),
	}
	transfersNum := 1000000

	toCp := testcase.to.clone()
	fromCp := testcase.from.clone()

	for i := 0; i < transfersNum; i++ {
		err := Transfer(testcase.ammount, &testcase.from, &testcase.to)
		assert.NilError(t, err, fmt.Sprintf("loop index %v", i))
	}
	assert.Equal(t, testcase.from.Currency, fromCp.Currency-uint64(transfersNum))
	assert.Equal(t, testcase.from.SupplementaryCurrency, fromCp.SupplementaryCurrency-uint64(transfersNum))
	assert.Equal(t, testcase.to.Currency, toCp.Currency+uint64(transfersNum))
	assert.Equal(t, testcase.to.SupplementaryCurrency, toCp.SupplementaryCurrency+uint64(transfersNum))
}

func BenchmarkMelangeTransfer(b *testing.B) {
	testcase := struct {
		from, to, ammount Melange
	}{
		from: New(math.MaxUint64, maxAmoutnPerSupplementaryCurrency), to: New(0, 0), ammount: New(1, 1),
	}

	for n := 0; n < b.N; n++ {
		Transfer(testcase.ammount, &testcase.from, &testcase.to)
	}
}
