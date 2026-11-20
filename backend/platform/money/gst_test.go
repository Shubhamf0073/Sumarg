package money

import (
	"errors"
	"math"
	"math/big"
	"testing"
)

func TestSplitInclusive(t *testing.T) {
	tests := []struct {
		name         string
		gross        Paise
		rate         BasisPoints
		taxable, tax Paise
		cgst, sgst   Paise // HalveTax(tax)
	}{
		// docs/05 §12 and docs/10: the price per branch, GST-inclusive.
		{"one branch", Rupees(4000), 1800, 338983, 61017, 30508, 30509},
		{"two branches (MTC)", Rupees(8000), 1800, 677966, 122034, 61017, 61017},
		{"prorated 15 of 30 days", Rupees(2000), 1800, 169492, 30508, 15254, 15254},
		{"zero", 0, 1800, 0, 0, 0, 0},
		{"zero rate", Rupees(4000), 0, Rupees(4000), 0, 0, 0},
		{"one paisa", 1, 1800, 1, 0, 0, 0}, // 0.847 → 1: tax can't be negative
		{"exact half rounds up", 59, 1800, 50, 9, 4, 5},
	}
	for _, tt := range tests {
		taxable, tax, err := SplitInclusive(tt.gross, tt.rate)
		if err != nil || taxable != tt.taxable || tax != tt.tax {
			t.Errorf("%s: SplitInclusive(%d) = %d + %d, %v; want %d + %d", tt.name, tt.gross, taxable, tax, err,
				tt.taxable, tt.tax)
			continue
		}
		cgst, sgst, err := HalveTax(tax)
		if err != nil || cgst != tt.cgst || sgst != tt.sgst || cgst+sgst != tax {
			t.Errorf("%s: HalveTax(%d) = %d + %d, %v; want %d + %d", tt.name, tax, cgst, sgst, err, tt.cgst, tt.sgst)
		}
	}
	if _, _, err := SplitInclusive(-1, 1800); !errors.Is(err, ErrInvalid) {
		t.Errorf("negative gross should be ErrInvalid, got %v", err)
	}
	for _, r := range []BasisPoints{-1, Whole + 1} {
		if _, _, err := SplitInclusive(100, r); !errors.Is(err, ErrRate) {
			t.Errorf("rate %d should be ErrRate, got %v", r, err)
		}
	}
	if _, _, err := HalveTax(-2); !errors.Is(err, ErrInvalid) {
		t.Errorf("negative tax should be ErrInvalid, got %v", err)
	}
}

// referenceInclusive is SplitInclusive's taxable part with big.Int, written independently so the two check each other.
func referenceInclusive(gross Paise, r BasisPoints) Paise {
	num := new(big.Int).Mul(big.NewInt(int64(gross)), big.NewInt(int64(Whole)))
	den := big.NewInt(int64(Whole + r))
	q, rem := new(big.Int).QuoRem(num, den, new(big.Int))
	if new(big.Int).Mul(rem, big.NewInt(2)).Cmp(den) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	return Paise(q.Int64())
}

func FuzzSplitInclusive(f *testing.F) {
	for _, s := range [][2]int64{{400000, 1800}, {59, 1800}, {math.MaxInt64, 1800}, {1, 0}} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, g, raw int64) {
		if g < 0 {
			g = -(g + 1) // map onto 0..MaxInt64
		}
		r := BasisPoints(raw % (int64(Whole) + 1))
		if r < 0 {
			r = -r
		}
		taxable, tax, err := SplitInclusive(Paise(g), r)
		if err != nil {
			t.Fatalf("SplitInclusive(%d, %d): %v", g, r, err)
		}
		if want := referenceInclusive(Paise(g), r); taxable != want || taxable+tax != Paise(g) || tax < 0 {
			t.Fatalf("SplitInclusive(%d, %d) = %d + %d; reference taxable %d", g, r, taxable, tax, want)
		}
	})
}
