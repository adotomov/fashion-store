package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/domain"
)

func TestFakeSpeedyClient_TrackProgressesWithAge(t *testing.T) {
	base := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	client := NewFakeSpeedyClient()
	client.now = func() time.Time { return base }

	shipment, err := client.CreateShipment(context.Background(), application.CreateShipmentRequest{})
	if err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}
	if shipment.ParcelID == "" {
		t.Fatal("expected a fake parcel id")
	}

	cases := []struct {
		age  time.Duration
		want string
	}{
		{0, "picked_up"},
		{31 * time.Second, "in_transit"},
		{91 * time.Second, "out_for_delivery"},
		{151 * time.Second, "delivered"},
	}
	for _, tc := range cases {
		client.now = func() time.Time { return base.Add(tc.age) }
		tracked, err := client.Track(context.Background(), application.Credentials{}, []string{shipment.ParcelID})
		if err != nil {
			t.Fatalf("Track at age %s: %v", tc.age, err)
		}
		if len(tracked) != 1 {
			t.Fatalf("age %s: expected 1 tracked parcel, got %d", tc.age, len(tracked))
		}
		got := domain.FriendlyStatus(tracked[0].OperationCode, tracked[0].Description)
		if got != tc.want {
			t.Errorf("age %s: friendly status = %q, want %q", tc.age, got, tc.want)
		}
	}
}

func TestFakeSpeedyClient_TrackSkipsUnknownParcels(t *testing.T) {
	client := NewFakeSpeedyClient()
	tracked, err := client.Track(context.Background(), application.Credentials{}, []string{"REAL-123", "DEVP-notanumber-0001"})
	if err != nil {
		t.Fatalf("Track: %v", err)
	}
	if len(tracked) != 0 {
		t.Fatalf("expected non-fake parcel ids to be skipped, got %d", len(tracked))
	}
}

func TestFakeSpeedyClient_SearchOfficesReturnsCatalogue(t *testing.T) {
	client := NewFakeSpeedyClient()
	offices, err := client.SearchOffices(context.Background(), application.Credentials{}, "Plovdiv", "OFFICE")
	if err != nil {
		t.Fatalf("SearchOffices: %v", err)
	}
	if len(offices) == 0 {
		t.Fatal("expected a non-empty office catalogue")
	}
	for _, o := range offices {
		if o.Type != "OFFICE" {
			t.Errorf("office %s: type = %q, want OFFICE", o.ID, o.Type)
		}
	}
}
