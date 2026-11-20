package money

import "math/bits"

// BasisPoints is a rate in hundredths of a percent: 18% is 1800, 2.5% is 250, 100% is 10_000. Integer basis
// points keep rates as exact as amounts, and a separate type means ApplyRate(p, 18) or swapped arguments don't
// compile by accident: write ApplyRate(p, BasisPoints(1800)). Tax rates themselves are configuration, not
// constants here: they belong to the accountant and change with GST reforms.
type BasisPoints int64

// Whole is 100%.
const Whole BasisPoints = 10_000

// ApplyRate returns p × r, rounded to the nearest paisa, half a paisa rounding away from zero (round half up on the
// magnitude): 2.5 → 3, 3.5 → 4, -2.5 → -3. Everything else rounds to the nearest paisa. r must be between 0 and
// 100%. The result is never larger in size than p, so it can't overflow.
//
// Rounding rule (docs/05 §3): half-up, the usual practice on Indian GST invoices; tax is computed per component
// (CGST and SGST each at half the rate) on the bill's taxable value, once per bill, never per line. The tenant's
// CA confirms this at onboarding (docs/09); this function is the only place to change it.
//
// Go note: p × r can exceed int64, so the product is computed as a 128-bit integer with math/bits
// (Mul64 gives the high and low 64 bits; Div64 divides that pair). No floats anywhere.
func ApplyRate(p Paise, r BasisPoints) (Paise, error) {
	if r < 0 || r > Whole {
		return 0, ErrRate
	}
	hi, lo := bits.Mul64(p.abs(), uint64(r))
	// hi < 10_000 because |p| ≤ 2^63 and r ≤ 10_000, so Div64 can't overflow (it panics if hi ≥ divisor).
	q, rem := bits.Div64(hi, lo, uint64(Whole))
	switch {
	case 2*rem >= uint64(Whole): // half or more: round up (away from zero once the sign is restored)
		q++
	}
	// q ≤ |p|: rounding up only happens when the exact result is below |p|, so q fits back in an int64
	// (for p = MinInt64 and 100%, q = 2^63 and the conversion below wraps to exactly MinInt64).
	if p < 0 {
		return -Paise(q), nil
	}
	return Paise(q), nil
}
