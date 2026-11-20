package money

import (
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"testing"
)

func TestParseRupees(t *testing.T) {
	valid := map[string]Paise{
		"0":                    0,
		"0.01":                 1,
		"0.5":                  50,
		"1":                    100,
		"1.5":                  150,
		"1.50":                 150,
		"007":                  700,
		"1234.05":              123405,
		"92233720368547758.07": math.MaxInt64,
	}
	for in, want := range valid {
		if got, err := ParseRupees(in); err != nil || got != want {
			t.Errorf("ParseRupees(%q) = %d, %v; want %d", in, got, err, want)
		}
	}

	invalid := map[string]error{
		"":                          ErrInvalid,
		" ":                         ErrInvalid,
		"12.345":                    ErrInvalid, // must not silently round
		"1.234":                     ErrInvalid,
		"1,234":                     ErrInvalid,
		"1 234":                     ErrInvalid,
		"-5":                        ErrInvalid,
		"+5":                        ErrInvalid,
		".5":                        ErrInvalid,
		"5.":                        ErrInvalid,
		"1.2.3":                     ErrInvalid,
		"1e3":                       ErrInvalid,
		"0x10":                      ErrInvalid,
		"१२":                        ErrInvalid, // Devanagari digits
		"₹100":                      ErrInvalid,
		"100 ":                      ErrInvalid,
		"1234567890123456789012345": ErrInvalid, // longer than 24 characters
		"92233720368547758.08":      ErrOverflow,
		"99999999999999999999":      ErrOverflow,
	}
	for in, want := range invalid {
		if got, err := ParseRupees(in); !errors.Is(err, want) || got != 0 {
			t.Errorf("ParseRupees(%q) = %d, %v; want 0, %v", in, got, err, want)
		}
	}
}

func TestString(t *testing.T) {
	tests := map[Paise]string{
		0:             "0.00",
		5:             "0.05",
		50:            "0.50",
		100:           "1.00",
		123405:        "1234.05",
		-150:          "-1.50",
		-5:            "-0.05",
		math.MaxInt64: "92233720368547758.07",
		math.MinInt64: "-92233720368547758.08",
	}
	for p, want := range tests {
		if got := p.String(); got != want {
			t.Errorf("Paise(%d).String() = %q, want %q", int64(p), got, want)
		}
	}
}

func TestFormatINR(t *testing.T) {
	tests := map[Paise]string{
		0:                "₹0.00",
		5:                "₹0.05",
		99900:            "₹999.00",
		100000:           "₹1,000.00",
		1234500:          "₹12,345.00",
		Rupees(100000):   "₹1,00,000.00", // 1 lakh
		12345678:         "₹1,23,456.78",
		Rupees(10000000): "₹1,00,00,000.00", // 1 crore
		-12345678:        "-₹1,23,456.78",
		math.MinInt64:    "-₹92,23,37,20,36,85,47,758.08",
	}
	for p, want := range tests {
		if got := p.FormatINR(); got != want {
			t.Errorf("Paise(%d).FormatINR() = %q, want %q", int64(p), got, want)
		}
	}
}

func TestJSONIsIntegerPaise(t *testing.T) {
	b, err := json.Marshal(struct{ Total Paise }{Total: 123405})
	if err != nil || string(b) != `{"Total":123405}` {
		t.Fatalf("got %s, %v", b, err)
	}
}

func TestRupees(t *testing.T) {
	if Rupees(12) != 1200 || Rupees(-3) != -300 {
		t.Fatal("Rupees conversion wrong")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("Rupees must panic on overflow")
		}
	}()
	Rupees(math.MaxInt64 / 10)
}

func TestSum(t *testing.T) {
	if got, err := Sum(100, 250, -50); err != nil || got != 300 {
		t.Fatalf("Sum = %d, %v", got, err)
	}
	if got, err := Sum(); err != nil || got != 0 {
		t.Fatalf("empty Sum = %d, %v", got, err)
	}
	for _, in := range [][]Paise{{math.MaxInt64, 1}, {math.MinInt64, -1}, {math.MaxInt64 - 5, 3, 3}} {
		if _, err := Sum(in...); !errors.Is(err, ErrOverflow) {
			t.Errorf("Sum(%v) should overflow, got %v", in, err)
		}
	}
}

func TestApplyRate(t *testing.T) {
	tests := []struct {
		p    Paise
		bp   BasisPoints
		want Paise
	}{
		{Rupees(100), 1800, 1800}, // 18% of ₹100 = ₹18
		{Rupees(100), 250, 250},   // 2.5%
		{3, 1800, 1},              // 0.54 paisa -> 1
		{2, 1800, 0},              // 0.36 paisa -> 0
		// exactly half a paisa: away from zero (half-up on the magnitude, docs/05 §3)
		{25, 1000, 3},   // 2.5 -> 3
		{35, 1000, 4},   // 3.5 -> 4
		{15, 1000, 2},   // 1.5 -> 2
		{5, 1000, 1},    // 0.5 -> 1
		{-25, 1000, -3}, // -2.5 -> -3
		{-35, 1000, -4}, // -3.5 -> -4
		{-3, 1800, -1},  // -0.54 -> -1
		{1, 5000, 1},    // 0.5 -> 1
		{3, 5000, 2},    // 1.5 -> 2
		// just either side of half
		{2501, 1, 0}, // 0.2501 -> 0
		{5001, 1, 1}, // 0.5001 -> 1
		{4999, 1, 0}, // 0.4999 -> 0
		// bounds
		{Rupees(100), 0, 0},
		{Rupees(100), 10000, Rupees(100)},
		{math.MaxInt64, 10000, math.MaxInt64},
		{math.MinInt64, 10000, math.MinInt64},
		{math.MaxInt64, 5000, 4611686018427387904}, // ...903.5 -> up ...904
		{math.MinInt64, 5000, -4611686018427387904},
	}
	for _, tt := range tests {
		if got, err := ApplyRate(tt.p, tt.bp); err != nil || got != tt.want {
			t.Errorf("ApplyRate(%d, %d) = %d, %v; want %d", int64(tt.p), tt.bp, got, err, tt.want)
		}
	}
	for _, bp := range []BasisPoints{-1, 10001, math.MaxInt64, math.MinInt64} {
		if _, err := ApplyRate(100, bp); !errors.Is(err, ErrRate) {
			t.Errorf("ApplyRate(100, %d) should be ErrRate, got %v", bp, err)
		}
	}
}

// Pins docs/05 §3: CGST and SGST are each rounded at half the rate, so on tiny amounts their total can differ from
// IGST at the full rate. That is the documented rule, not a bug.
func TestSplitTaxDiffersFromWholeRate(t *testing.T) {
	cgst, _ := ApplyRate(3, 900)  // 0.27 -> 0
	igst, _ := ApplyRate(3, 1800) // 0.54 -> 1
	if cgst+cgst != 0 || igst != 1 {
		t.Fatalf("9%%+9%% = %d, 18%% = %d; want 0 and 1", cgst+cgst, igst)
	}
}

func TestNeg(t *testing.T) {
	for _, p := range []Paise{0, 150, -150, math.MaxInt64} {
		if got, err := p.Neg(); err != nil || got != -p {
			t.Errorf("Neg(%d) = %d, %v", int64(p), got, err)
		}
	}
	if _, err := Paise(math.MinInt64).Neg(); !errors.Is(err, ErrOverflow) {
		t.Errorf("Neg(MinInt64) should be ErrOverflow, got %v", err)
	}
}

// referenceRate computes p × bp / 10000, half away from zero, using exact big-integer arithmetic.
// It is written differently from ApplyRate (big.Int instead of 128-bit math/bits) so the two check each other.
func referenceRate(p Paise, bp BasisPoints) Paise {
	num := new(big.Int).Mul(big.NewInt(int64(p)), big.NewInt(int64(bp)))
	den := big.NewInt(int64(Whole))
	q, r := new(big.Int).QuoRem(num, den, new(big.Int)) // truncates toward zero
	twice := new(big.Int).Abs(new(big.Int).Mul(r, big.NewInt(2)))
	step := big.NewInt(int64(num.Sign()))
	if twice.Cmp(den) >= 0 {
		q.Add(q, step)
	}
	return Paise(q.Int64())
}

func FuzzApplyRate(f *testing.F) {
	for _, s := range [][2]int64{{25, 1000}, {35, 1000}, {math.MaxInt64, 5000}, {math.MinInt64, 10000}, {-3, 1800}} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, p, raw int64) {
		bp := BasisPoints(raw % (int64(Whole) + 1))
		if bp < 0 {
			bp = -bp
		}
		got, err := ApplyRate(Paise(p), bp)
		if err != nil {
			t.Fatalf("ApplyRate(%d, %d): %v", p, bp, err)
		}
		if want := referenceRate(Paise(p), bp); got != want {
			t.Fatalf("ApplyRate(%d, %d) = %d, reference = %d", p, bp, got, want)
		}
	})
}

func FuzzParseRupees(f *testing.F) {
	for _, s := range []string{"0", "1.5", "1234.05", "12.345", "-1", "92233720368547758.07", "1e3"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		p, err := ParseRupees(s)
		if err != nil {
			return
		}
		if p < 0 {
			t.Fatalf("ParseRupees(%q) returned a negative amount %d", s, p)
		}
		back, err := ParseRupees(p.String())
		if err != nil || back != p {
			t.Fatalf("round trip of %q: String() = %q, parsed back to %d, %v", s, p.String(), back, err)
		}
	})
}
