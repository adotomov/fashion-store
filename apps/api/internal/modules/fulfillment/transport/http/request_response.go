package http

import (
	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/domain"
)

const timeFormat = "2006-01-02T15:04:05Z07:00"

// knownProviders is the registry of carriers the admin Logistics page can
// show. Adding a new carrier later means adding an entry here (plus its own
// SpeedyClient-shaped port/client) — the page itself is data-driven.
var knownProviders = []struct {
	Code string
	Name string
}{
	{Code: domain.ProviderSpeedy, Name: "Speedy"},
}

// maskedConfigKeys are never echoed back verbatim once set, so the admin
// form doesn't render a stored password in plain text.
var maskedConfigKeys = map[string]bool{
	domain.SpeedyConfigPassword: true,
}

type providerResponse struct {
	Provider  string            `json:"provider"`
	Name      string            `json:"name"`
	Enabled   bool              `json:"enabled"`
	Config    map[string]string `json:"config"`
	UpdatedAt string            `json:"updated_at,omitempty"`
}

func toProviderResponse(code, name string, settings *domain.ProviderSettings) providerResponse {
	resp := providerResponse{Provider: code, Name: name, Config: map[string]string{}}
	if settings == nil {
		return resp
	}
	resp.Enabled = settings.Enabled
	resp.UpdatedAt = settings.UpdatedAt.Format(timeFormat)
	for k, v := range settings.Config {
		if maskedConfigKeys[k] && v != "" {
			resp.Config[k] = "********"
			continue
		}
		resp.Config[k] = v
	}
	return resp
}

type saveProviderRequest struct {
	Enabled bool              `json:"enabled"`
	Config  map[string]string `json:"config"`
}

type officeResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func toOfficeResponses(offices []application.Office) []officeResponse {
	resp := make([]officeResponse, 0, len(offices))
	for _, o := range offices {
		resp = append(resp, officeResponse{ID: o.ID, Name: o.Name, Type: o.Type})
	}
	return resp
}
