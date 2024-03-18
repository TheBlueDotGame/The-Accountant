package spice

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	msgpackv2 "github.com/shamaton/msgpack/v2"
	"github.com/vmihailenco/msgpack"
)

const (
	MaxAmountPerSupplementaryCurrency = 1000000000000000000
)

const (
	Currency = iota
	SupplementaryCurrency
)

var (
	ErrValueOverflow      = errors.New("value overflow")
	ErrNoSufficientFounds = errors.New("no sufficient founds to process transaction")
)

func convertFloatToInt(d float64) (int, int) {
	intPart := int(d)

	parts := strings.SplitN(fmt.Sprintf("%.10f", d), ".", 2)
	if len(parts) < 2 {
		return intPart, 0
	}
	p := parts[1]

	var isPassedLeading bool
	var buf strings.Builder
	for i := 0; i < 18; i++ {
		if i >= len(p) {
			buf.WriteRune('0')
			continue
		}
		if p[i] != '0' || isPassedLeading {
			buf.WriteByte(p[i])
			isPassedLeading = true
		}
	}

	fractionalPart, _ := strconv.Atoi(buf.String())
	return intPart, fractionalPart
}

// Melange is an asset that is digitally transferable between two wallets.
type Melange struct {
	Currency              uint64 `yaml:"currency"               msgpack:"currency"`
	SupplementaryCurrency uint64 `yaml:"supplementary_currency" msgpack:"supplementary_currency"`
}

// New creates a new spice Melange from given currency and supplementary currency values.
func New(currency, supplementaryCurrency uint64) Melange {
	if supplementaryCurrency >= MaxAmountPerSupplementaryCurrency {
		currency += 1
		supplementaryCurrency -= MaxAmountPerSupplementaryCurrency
	}
	return Melange{
		Currency:              currency,
		SupplementaryCurrency: supplementaryCurrency,
	}
}

// From float crates a new spice Melange from floating point number.
// Note this will give a precision up to nine decimal places.
func FromFloat(n float64) Melange {
	if n <= 0.0 {
		return Melange{}
	}
	cur, supl := convertFloatToInt(n)
	return New(uint64(cur), uint64(supl))
}

// Supply supplies spice of the given amount from the source to the entity.
func (m *Melange) Supply(amount Melange) error {
	mCp := m.Clone()
	for _, unit := range []byte{Currency, SupplementaryCurrency} {
		switch unit {
		case Currency:
			if math.MaxUint64-amount.Currency < m.Currency {
				return ErrValueOverflow
			}
			m.Currency += amount.Currency
		case SupplementaryCurrency:
			if MaxAmountPerSupplementaryCurrency-amount.SupplementaryCurrency < m.SupplementaryCurrency {
				if m.Currency == math.MaxUint64 {
					m.copyFrom(mCp)
					return ErrValueOverflow
				}
			}
			m.SupplementaryCurrency += amount.SupplementaryCurrency

			if m.SupplementaryCurrency >= MaxAmountPerSupplementaryCurrency {
				m.Currency += 1
				m.SupplementaryCurrency -= MaxAmountPerSupplementaryCurrency
			}
		}
	}
	return nil
}

// Drain drains amount from the function pointer receiver to the sink.
func (m *Melange) Drain(amount Melange, sink *Melange) error {
	return Transfer(amount, m, sink)
}

// Empty verifies if is spice empty.
func (m *Melange) Empty() bool {
	return m.Currency == 0 && m.SupplementaryCurrency == 0
}

// Transfer transfers given amount from one Melange asset to the other if possible or returns error otherwise.
func Transfer(amount Melange, from, to *Melange) error {
	toCp := to.Clone()
	fromCp := from.Clone()
	for _, unit := range []byte{Currency, SupplementaryCurrency} {
		switch unit {
		case Currency:
			if amount.Currency > from.Currency {
				return ErrNoSufficientFounds
			}
			if math.MaxUint64-amount.Currency < to.Currency {
				return ErrValueOverflow
			}
			to.Currency += amount.Currency
			from.Currency -= amount.Currency
		case SupplementaryCurrency:
			if MaxAmountPerSupplementaryCurrency-amount.SupplementaryCurrency < to.SupplementaryCurrency {
				if to.Currency == math.MaxUint64 {
					to.copyFrom(toCp)
					from.copyFrom(fromCp)
					return ErrValueOverflow
				}
			}
			if amount.SupplementaryCurrency > from.SupplementaryCurrency {
				if from.Currency == 0 {
					to.copyFrom(toCp)
					from.copyFrom(fromCp)
					return ErrNoSufficientFounds
				}
				from.Currency -= 1
				from.SupplementaryCurrency = from.SupplementaryCurrency + MaxAmountPerSupplementaryCurrency - amount.SupplementaryCurrency
				to.SupplementaryCurrency += amount.SupplementaryCurrency

				if to.SupplementaryCurrency >= MaxAmountPerSupplementaryCurrency {
					to.Currency += 1
					to.SupplementaryCurrency -= MaxAmountPerSupplementaryCurrency
				}
				continue
			}
			from.SupplementaryCurrency -= amount.SupplementaryCurrency
			to.SupplementaryCurrency += amount.SupplementaryCurrency

			if to.SupplementaryCurrency >= MaxAmountPerSupplementaryCurrency {
				to.Currency += 1
				to.SupplementaryCurrency -= MaxAmountPerSupplementaryCurrency
			}
		}
	}
	return nil
}

// String returns string representation of spice Melange.
func (m Melange) String() string {
	supplementary := fmt.Sprintf("%v", m.SupplementaryCurrency)
	zeros := 18 - len(supplementary)
	if zeros < 0 {
		supplementary = "0"
	}
	supplementary = strings.Trim(supplementary, "0")

	var buf strings.Builder
	if len(supplementary) != 0 {
		for i := 0; i < zeros; i++ {
			buf.WriteString("0")
		}
	}
	buf.WriteString(supplementary)
	supp := buf.String()
	if len(supp) == 0 {
		supp = "0"
	}
	curr := fmt.Sprintf("%v", m.Currency)
	return fmt.Sprintf("%s.%s", curr, supp)
}

func (m Melange) Clone() Melange {
	return Melange{
		Currency:              m.Currency,
		SupplementaryCurrency: m.SupplementaryCurrency,
	}
}

func (m *Melange) copyFrom(c Melange) {
	m.Currency = c.Currency
	m.SupplementaryCurrency = c.SupplementaryCurrency
}

func (s *Melange) Encode() ([]byte, error) {
	buf, err := msgpack.Marshal(s)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func Decode(buf []byte) (Melange, error) {
	var s Melange
	err := msgpackv2.Unmarshal(buf, &s)
	return s, err
}
