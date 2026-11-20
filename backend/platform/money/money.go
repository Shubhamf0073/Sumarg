// Package money is the only place amounts are created, parsed, formatted and rounded (docs/05 §3, §12).
//
// An amount is a whole number of paise in an int64: never a float, which can't hold 0.10 exactly and drifts when
// summed. In the database it is bigint; in JSON it is a plain integer of paise (exact in JavaScript up to about
// ₹90 lakh crore). Negative amounts exist (refunds, reversals); ParseRupees only accepts what a person types for
// a charge or payment, so it rejects signs.
//
// Go note: Paise is a named type. Passing a bare int64 where Paise is expected doesn't compile without an explicit
// Paise(x) conversion, which makes "is this rupees or paise?" mistakes visible in review.
package money

import (
	"errors"
	"math"
)

// Paise is an amount of money in paise (1 rupee = 100 paise).
type Paise int64

var (
	ErrInvalid  = errors.New("money: invalid amount")
	ErrOverflow = errors.New("money: amount out of range")
	ErrRate     = errors.New("money: rate must be between 0 and 100% (0 to 10000 basis points)")
)

// Rupees returns n whole rupees. It is for constants and tests; it panics if n*100 overflows. Amounts that
// come from people or files go through ParseRupees, which returns an error instead.
func Rupees(n int64) Paise {
	if n > math.MaxInt64/100 || n < math.MinInt64/100 {
		panic("money.Rupees: overflow")
	}
	return Paise(n * 100)
}

// ParseRupees parses a rupee amount as a person types it: digits, then optionally "." and one or two digits
// ("1234", "1234.5", "1234.50"). It rejects everything else: signs, commas, spaces, exponents, a leading or
// trailing ".", more than two decimals ("12.345" must not silently round), non-ASCII digits, and overflow.
//
// It is for amounts only: it accepts leading zeros ("007" is ₹7), so never reuse it for UTRs or reference
// numbers, where leading zeros matter. It accepts zero; a charge or payment handler must reject amounts <= 0
// itself.
func ParseRupees(s string) (Paise, error) {
	if len(s) == 0 || len(s) > 24 {
		return 0, ErrInvalid
	}
	whole, frac := s, ""
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			whole, frac = s[:i], s[i+1:]
			if len(frac) == 0 || len(frac) > 2 {
				return 0, ErrInvalid
			}
			break
		}
	}
	if len(whole) == 0 {
		return 0, ErrInvalid
	}
	var rupees int64
	for i := 0; i < len(whole); i++ {
		d, ok := digit(whole[i])
		if !ok {
			return 0, ErrInvalid
		}
		if rupees > (math.MaxInt64-d)/10 {
			return 0, ErrOverflow
		}
		rupees = rupees*10 + d
	}
	var paise int64
	for i := 0; i < 2; i++ {
		paise *= 10
		if i < len(frac) {
			d, ok := digit(frac[i])
			if !ok {
				return 0, ErrInvalid
			}
			paise += d
		}
	}
	if rupees > (math.MaxInt64-paise)/100 {
		return 0, ErrOverflow
	}
	return Paise(rupees*100 + paise), nil
}

func digit(c byte) (int64, bool) {
	if c < '0' || c > '9' {
		return 0, false
	}
	return int64(c - '0'), true
}

// Sum adds amounts, returning ErrOverflow instead of silently wrapping around.
func Sum(amounts ...Paise) (Paise, error) {
	var total Paise
	for _, a := range amounts {
		if (a > 0 && total > math.MaxInt64-a) || (a < 0 && total < math.MinInt64-a) {
			return 0, ErrOverflow
		}
		total += a
	}
	return total, nil
}

// Neg returns -p, for reversal entries. It returns ErrOverflow for math.MinInt64, whose negation doesn't fit in
// an int64 (a bare -p would silently give MinInt64 back).
func (p Paise) Neg() (Paise, error) {
	if p == math.MinInt64 {
		return 0, ErrOverflow
	}
	return -p, nil
}

// abs returns |p| as a uint64. It is correct even for math.MinInt64, whose magnitude (2^63) doesn't fit in an
// int64: in two's complement, -p wraps back to MinInt64, and converting that to uint64 gives exactly 2^63.
func (p Paise) abs() uint64 {
	if p < 0 {
		return uint64(-p)
	}
	return uint64(p)
}
