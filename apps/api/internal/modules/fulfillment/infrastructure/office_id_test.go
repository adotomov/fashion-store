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
