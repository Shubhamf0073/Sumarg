package money

import "math/bits"

// SplitInclusive splits a GST-inclusive gross amount into its taxable value and the tax, for a total GST rate r
// (18% = BasisPoints(1800)). It is what Previa's own subscription invoices use: the price per branch (₹4,000) is
// GST-inclusive (docs/05 §12):
//
//	taxable = round_half_up(gross × 100 / (100 + rate))     ₹4,000.00 → ₹3,389.83
//	tax     = gross − taxable                                 ₹610.17
//
// tax is the remainder, never rounded on its own, so taxable + tax == gross exactly. Rounding happens once per
// invoice on the invoice's gross total, never per line. gross must be ≥ 0 (credit notes negate the result).
func SplitInclusive(gross Paise, r BasisPoints) (taxable, tax Paise, err error) {
	if gross < 0 {
		return 0, 0, ErrInvalid
	}
	if r < 0 || r > Whole {
		return 0, 0, ErrRate
	}
	// gross × 10_000 / (10_000 + r) as a 128-bit product: gross can be any int64, the product can't overflow.
	div := uint64(Whole + r)
	hi, lo := bits.Mul64(uint64(gross), uint64(Whole))
	q, rem := bits.Div64(hi, lo, div) // hi < div because Whole ≤ div, so Div64 can't panic
	if 2*rem >= div {
		q++
	}
	// q ≤ gross: dividing by div ≥ Whole never makes the amount bigger, and rounding up only reaches gross itself.
	taxable = Paise(q)
	return taxable, gross - taxable, nil
}

// HalveTax splits an intra-state tax amount into CGST and SGST for Previa's inclusive invoices: CGST gets the lower
// half, SGST the remainder, so the two always add back to tax (₹610.17 → ₹305.08 + ₹305.09). tax must be ≥ 0.
//
// Tenant bills (LR freight) don't use this: there each component is ApplyRate at half the rate (docs/05 §3).
func HalveTax(tax Paise) (cgst, sgst Paise, err error) {
	if tax < 0 {
		return 0, 0, ErrInvalid
	}
	cgst = tax / 2
	return cgst, tax - cgst, nil
}
