package domain

import "time"

// Provider identifies a logistics carrier integration. Speedy is the pilot;
// more carriers are added by introducing new constants and config schemas,
// not by changing this module's shape.
const ProviderSpeedy = "speedy"

// ProviderSettings is an admin-configured logistics provider: whether it's
// enabled (which gates the delivery methods checkout offers) and a free-form
// config map (credentials, default service IDs, etc.) whose keys are
// provider-specific.
type ProviderSettings struct {
	Provider  string
	Enabled   bool
	Config    map[string]string
	UpdatedAt time.Time
}

// Speedy config keys captured by the admin settings form. Sender details are
// deliberately omitted — Speedy defaults to the authenticated account's own
// registered pickup address, so there's nothing for us to configure there.
const (
	SpeedyConfigUsername                = "username"
	SpeedyConfigPassword                = "password"
	SpeedyConfigLanguage                = "language"
	SpeedyConfigClientSystemID          = "client_system_id"
	SpeedyConfigDefaultCourierServiceID = "default_courier_service_id"
	SpeedyConfigDefaultOfficeServiceID  = "default_office_service_id"
	SpeedyConfigDefaultLockerServiceID  = "default_locker_service_id"
	SpeedyConfigDefaultParcelWeightKg   = "default_parcel_weight_kg"
	// Speedy requires a content description and a package type on every shipment
	// (ShipmentContent.contents / .package, both mandatory). These configure the
	// store-wide defaults; blank falls back to the constants in the service.
	SpeedyConfigDefaultParcelContents = "default_parcel_contents"
	SpeedyConfigDefaultParcelPackage  = "default_parcel_package"
)

// Fallback values for the mandatory Speedy content fields when the provider
// config leaves them blank, so shipments always carry the required description.
const (
	DefaultParcelContents = "Дрехи и аксесоари"
	DefaultParcelPackage  = "Кутия"
)
