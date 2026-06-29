package money

// Money represents an amount in integer minor units (e.g. cents) to avoid
// floating-point rounding errors. Never use float64 for money.
type Money struct {
	AmountMinor int64
	Currency    string
}
