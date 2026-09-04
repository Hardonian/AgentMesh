package cost

import (
	"math"
	"testing"
)

func TestCostIntelligence_MicroUSD(t *testing.T) {
	ci := NewIntelligence()

	// 1000 input tokens at $0.00125/1k, 1000 output tokens at $0.005/1k = 0.00625 USD = 6,250 MicroUSD
	micro := ci.CalculateCostMicroUSD("gemini-1.5-pro", 1000, 1000, 0)
	if micro != 6250 {
		t.Fatalf("expected 6250 MicroUSD, got %d", micro)
	}

	usd := micro.ToUSD()
	if math.Abs(usd-0.00625) > 1e-9 {
		t.Fatalf("expected 0.00625 USD, got %f", usd)
	}

	// Test negative rejection
	_, err := ToMicroUSD(-1.0)
	if err == nil {
		t.Fatal("expected error for negative USD, got nil")
	}

	// Test NaN / Inf rejection
	_, err = ToMicroUSD(math.NaN())
	if err == nil {
		t.Fatal("expected error for NaN USD, got nil")
	}

	_, err = ToMicroUSD(math.Inf(1))
	if err == nil {
		t.Fatal("expected error for Inf USD, got nil")
	}

	// Negative token counts in CalculateCost
	c := ci.CalculateCost("gemini-1.5-pro", -500, -500, -500)
	if c != 0.0 {
		t.Fatalf("expected 0.0 cost for negative tokens, got %f", c)
	}
}
