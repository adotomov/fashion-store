package infrastructure

import (
	"encoding/json"
	"strings"
	"testing"
)

// Speedy's shipment schema names the pickup-point field "pickupOfficeId" and
// types it as an integer; a string "officeId" is silently ignored, so the
// parcel would never route to the chosen office. Lock the wire shape.
func TestSpeedyRecipientPickupOfficeMarshal(t *testing.T) {
	b, err := json.Marshal(speedyRecipient{PrivatePerson: true, ClientName: "Ivan", PickupOfficeID: 2966})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, `"pickupOfficeId":2966`) {
		t.Errorf("expected integer pickupOfficeId in body, got %s", got)
	}
	if strings.Contains(got, "officeId") && !strings.Contains(got, "pickupOfficeId") {
		t.Errorf("unexpected legacy officeId field: %s", got)
	}
	// When no pickup office is set the field must be omitted (address delivery).
	b2, _ := json.Marshal(speedyRecipient{PrivatePerson: true, ClientName: "Ivan"})
	if strings.Contains(string(b2), "pickupOfficeId") {
		t.Errorf("pickupOfficeId should be omitted when zero, got %s", string(b2))
	}
}

// The create-shipment body must match Speedy's schema: serviceId is an integer,
// content carries totalWeight, and the COD additional service is nested under
// `service` (not top-level) with a `currencyCode` key. Getting any of these
// wrong means Speedy rejects the shipment or, worse, silently drops the COD and
// delivers without collecting money.
func TestCreateShipmentBodyShape(t *testing.T) {
	body := createShipmentRequest{
		Recipient: speedyRecipient{PrivatePerson: true, ClientName: "Ivan", PickupOfficeID: 2966},
		Service:   speedyService{ServiceID: 505, AutoAdjustPickupDate: true},
		Content:   speedyContent{ParcelsCount: 1, TotalWeight: 1.5, Contents: "Дрехи и аксесоари", Package: "Кутия"},
		Payment:   speedyPayment{CourierServicePayer: "SENDER"},
	}
	body.Service.AdditionalServices = &speedyAdditionalServices{COD: &speedyCOD{
		Amount: 49.90, CurrencyCode: "BGN", ProcessingType: "CASH",
	}}

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := m["additionalServices"]; ok {
		t.Errorf("additionalServices must not be top-level; it belongs under service: %s", raw)
	}
	service, _ := m["service"].(map[string]any)
	if service == nil {
		t.Fatalf("missing service object: %s", raw)
	}
	if sid, ok := service["serviceId"].(float64); !ok || sid != 505 {
		t.Errorf("service.serviceId should be integer 505, got %v (%s)", service["serviceId"], raw)
	}
	// autoAdjustPickupDate must ride along so after-cutoff/weekend bookings roll to
	// the next pickup day instead of Speedy rejecting them with code 500.
	if adj, ok := service["autoAdjustPickupDate"].(bool); !ok || !adj {
		t.Errorf("service.autoAdjustPickupDate should be true, got %v (%s)", service["autoAdjustPickupDate"], raw)
	}
	content, _ := m["content"].(map[string]any)
	if tw, ok := content["totalWeight"].(float64); !ok || tw != 1.5 {
		t.Errorf("content.totalWeight should be 1.5, got %v (%s)", content["totalWeight"], raw)
	}
	// contents + package are mandatory; Speedy 600s the create without them.
	if c, ok := content["contents"].(string); !ok || c == "" {
		t.Errorf("content.contents (description) is required, got %v (%s)", content["contents"], raw)
	}
	if p, ok := content["package"].(string); !ok || p == "" {
		t.Errorf("content.package is required, got %v (%s)", content["package"], raw)
	}
	// A parcels[] array requires a sequential seqNo per entry; sending it without
	// one is what triggered Speedy code 1. For a single parcel we must omit it.
	if _, bad := content["parcels"]; bad {
		t.Errorf("content.parcels must be omitted for a single parcel: %s", raw)
	}
	as, _ := service["additionalServices"].(map[string]any)
	cod, _ := as["cod"].(map[string]any)
	if cod == nil {
		t.Fatalf("COD must be nested under service.additionalServices.cod: %s", raw)
	}
	if _, bad := cod["currency"]; bad {
		t.Errorf("COD must use currencyCode, not currency: %s", raw)
	}
	if cc, ok := cod["currencyCode"].(string); !ok || cc != "BGN" {
		t.Errorf("cod.currencyCode should be BGN, got %v (%s)", cod["currencyCode"], raw)
	}
}

// Speedy accepts only digits + an optional leading "+", with spaces as the sole
// separator, starting with "0" or "+" — anything else 100s the create. Lock the
// normalization that gets customer-entered numbers into that shape.
func TestNormalizePhone(t *testing.T) {
	cases := []struct{ in, want string }{
		{"0888 123 456", "0888123456"},        // spaces stripped
		{"0888-123-456", "0888123456"},        // dashes stripped
		{"(088) 812 3456", "0888123456"},      // parentheses stripped
		{"+359 88 812 3456", "+359888123456"}, // leading + preserved
		{"00359888123456", "+359888123456"},   // 00 international prefix -> +
		{"888123456", "0888123456"},           // bare 9-digit national gets leading 0
		{"  ", ""},                            // blank stays blank
	}
	for _, tc := range cases {
		if got := normalizePhone(tc.in); got != tc.want {
			t.Errorf("normalizePhone(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Speedy returns office ids as JSON numbers. Modeling speedyOffice.ID as a
// plain string used to 500 the office picker on prod ("cannot unmarshal number
// into Go struct field ... of type string"). These cases lock in that the
// custom unmarshaler decodes number, quoted string, and null.
func TestSpeedyOfficeDecode(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"numeric id", `{"offices":[{"id":2966,"name":"Sofia","type":"OFFICE"}]}`, "2966"},
		{"string id", `{"offices":[{"id":"2966","name":"Sofia","type":"OFFICE"}]}`, "2966"},
		{"null id", `{"offices":[{"id":null,"name":"Sofia","type":"OFFICE"}]}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var resp officeSearchResponse
			if err := json.Unmarshal([]byte(tc.body), &resp); err != nil {
				t.Fatalf("unexpected decode error: %v", err)
			}
			if len(resp.Offices) != 1 {
				t.Fatalf("expected 1 office, got %d", len(resp.Offices))
			}
			if got := string(resp.Offices[0].ID); got != tc.want {
				t.Fatalf("office id = %q, want %q", got, tc.want)
			}
		})
	}
}
