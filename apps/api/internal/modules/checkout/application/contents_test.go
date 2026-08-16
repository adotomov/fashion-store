package application

import "testing"

func TestShipmentContentsFromItems(t *testing.T) {
	cases := []struct {
		name  string
		items []OrderResultItem
		want  string
	}{
		{"empty", nil, ""},
		{
			"joins names",
			[]OrderResultItem{{ProductName: "Тениска"}, {ProductName: "Дънки"}},
			"Тениска, Дънки",
		},
		{
			"dedupes and trims, skips blanks",
			[]OrderResultItem{{ProductName: " Тениска "}, {ProductName: "Тениска"}, {ProductName: ""}, {ProductName: "Шапка"}},
			"Тениска, Шапка",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shipmentContentsFromItems(tc.items); got != tc.want {
				t.Errorf("shipmentContentsFromItems() = %q, want %q", got, tc.want)
			}
		})
	}
}
