package application

import "testing"

func TestVATExclusive(t *testing.T) {
	tests := []struct {
		name    string
		incl    int64
		rate    float64
		wantVAL int64
	}{
		{"20 percent matches legacy 100/120 split", 12000, 20, 10000},
		{"9 percent reduced rate", 10900, 9, 10000},
		{"0 percent leaves amount untouched", 5000, 0, 5000},
		{"rounds down (floor)", 100, 20, 83}, // 100*100/120 = 83.33 -> 83
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vatExclusive(tt.incl, tt.rate); got != tt.wantVAL {
				t.Errorf("vatExclusive(%d, %v) = %d, want %d", tt.incl, tt.rate, got, tt.wantVAL)
			}
		})
	}
}
