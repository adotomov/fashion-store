package application

type CreatePaymentMethodInput struct {
	Brand     string
	Last4     string
	ExpMonth  int
	ExpYear   int
	IsDefault bool
}

type UpdatePaymentMethodInput struct {
	Brand     *string
	Last4     *string
	ExpMonth  *int
	ExpYear   *int
	IsDefault *bool
}
