package rsi

import "testing"

func TestCalcRSIIncreasingSeries(t *testing.T) {
	closes := []float64{1, 2, 3, 4, 5, 6, 7, 8}

	got := CalcRSI(closes, 7)
	if got != 100 {
		t.Fatalf("CalcRSI() = %.2f, want 100", got)
	}
}

func TestCalcStochRSIReturnsSmoothedValues(t *testing.T) {
	closes := []float64{
		44.34, 44.09, 44.15, 43.61, 44.33, 44.83, 45.1, 45.42, 45.84, 46.08,
		45.89, 46.03, 45.61, 46.28, 46.28, 46.0, 46.03, 46.41, 46.22, 45.64,
		46.21, 46.25, 45.71, 46.45, 45.78, 45.35, 44.03, 44.18, 44.22, 44.57,
		43.42, 42.66, 43.13,
	}

	values := CalcStochRSI(closes, 14, 14, 3, 3)
	if values.RSI <= 0 || values.RSI >= 100 {
		t.Fatalf("RSI out of range: %.4f", values.RSI)
	}
	if values.RawK < 0 || values.RawK > 100 {
		t.Fatalf("raw Stoch RSI out of range: %.4f", values.RawK)
	}
	if values.K < 0 || values.K > 100 {
		t.Fatalf("K out of range: %.4f", values.K)
	}
	if values.D < 0 || values.D > 100 {
		t.Fatalf("D out of range: %.4f", values.D)
	}
}
