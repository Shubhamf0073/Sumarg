package money

import "strings"

var (
	onesWords = [...]string{"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten",
		"Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen", "Seventeen", "Eighteen", "Nineteen"}
	tensWords = [...]string{"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety"}
)

// InWords spells an amount the way Indian bills print it, with lakh and crore grouping (docs/05 §8):
//
//	₹3,150.00      → "Rupees Three Thousand One Hundred Fifty Only"
//	₹3,389.83      → "Rupees Three Thousand Three Hundred Eighty Nine and Eighty Three Paise Only"
//	₹12,50,000.00  → "Rupees Twelve Lakh Fifty Thousand Only"
//
// Documents pass this string to the Typst template as-is (templates never compute).
func InWords(p Paise) string {
	var b strings.Builder
	n := int64(p)
	if n < 0 {
		b.WriteString("Minus ")
		if n == -n { // MinInt64 has no positive counterpart; one paisa off is fine for words
			n++
		}
		n = -n
	}
	rupees, paise := n/100, n%100
	b.WriteString("Rupees ")
	if rupees == 0 {
		b.WriteString("Zero")
	} else {
		b.WriteString(indianWords(rupees))
	}
	if paise > 0 {
		b.WriteString(" and ")
		b.WriteString(indianWords(paise))
		b.WriteString(" Paise")
	}
	b.WriteString(" Only")
	return b.String()
}

// indianWords spells n > 0 using crore (10^7), lakh (10^5), thousand and hundred. Amounts of 100 crore and more
// repeat the scale ("One Hundred Crore"), which is how Indian banks write them.
func indianWords(n int64) string {
	var parts []string
	if c := n / 10_000_000; c > 0 {
		parts = append(parts, indianWords(c)+" Crore")
		n %= 10_000_000
	}
	for _, s := range []struct {
		div  int64
		name string
	}{{100_000, "Lakh"}, {1_000, "Thousand"}, {100, "Hundred"}} {
		if q := n / s.div; q > 0 {
			parts = append(parts, belowHundred(q)+" "+s.name)
			n %= s.div
		}
	}
	if n > 0 {
		parts = append(parts, belowHundred(n))
	}
	return strings.Join(parts, " ")
}

func belowHundred(n int64) string {
	if n < 20 {
		return onesWords[n]
	}
	if n%10 == 0 {
		return tensWords[n/10]
	}
	return tensWords[n/10] + " " + onesWords[n%10]
}
