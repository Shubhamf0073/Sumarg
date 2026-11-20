package money

import "strconv"

// String formats p as rupees with exactly two decimals and no grouping: "1234.50", "-0.05". ParseRupees reads the
// output back (for non-negative amounts). fmt and slog use this, so logs show rupees, not raw paise.
func (p Paise) String() string {
	whole, frac := p.split()
	s := whole + "." + frac
	if p < 0 {
		return "-" + s
	}
	return s
}

// FormatINR formats p the way Indian documents show money: rupee sign and lakh/crore grouping (the last three
// digits, then groups of two): "₹1,23,456.78", "-₹1,000.00". Used on PDFs and exports.
func (p Paise) FormatINR() string {
	whole, frac := p.split()
	s := "₹" + groupIndian(whole) + "." + frac
	if p < 0 {
		return "-" + s
	}
	return s
}

// split returns the rupee digits and the two paise digits of |p|.
func (p Paise) split() (whole, frac string) {
	a := p.abs()
	frac = strconv.FormatUint(a%100, 10)
	if len(frac) == 1 {
		frac = "0" + frac
	}
	return strconv.FormatUint(a/100, 10), frac
}

// groupIndian inserts commas into a string of digits: 1234567 -> 12,34,567.
func groupIndian(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	head, tail := digits[:len(digits)-3], digits[len(digits)-3:]
	out := make([]byte, 0, len(digits)+len(digits)/2)
	if len(head)%2 == 1 {
		out = append(out, head[0], ',')
		head = head[1:]
	}
	for i := 0; i < len(head); i += 2 {
		out = append(out, head[i], head[i+1], ',')
	}
	return string(out) + tail
}
