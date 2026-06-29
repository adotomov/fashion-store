package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/application"
)

const speedyBaseURL = "https://api.speedy.bg/v1"

// SpeedyHTTPClient implements application.SpeedyClient against the real
// Speedy Web API, per the documented contract in .ai/speedy-docs/. The exact
// JSON field names below follow the field lists given in 09-data-models.md
// and 01-shipment-api.md; that documentation is an AI-summarized rewrite
// rather than Speedy's literal OpenAPI spec, so these should be checked
// against a real account/sandbox response before going live.
type SpeedyHTTPClient struct {
	httpClient *http.Client
}

func NewSpeedyHTTPClient() *SpeedyHTTPClient {
	return &SpeedyHTTPClient{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

type speedyAuth struct {
	UserName       string `json:"userName"`
	Password       string `json:"password"`
	Language       string `json:"language,omitempty"`
	ClientSystemID string `json:"clientSystemId,omitempty"`
}

func authFromCreds(creds application.Credentials) speedyAuth {
	return speedyAuth{UserName: creds.Username, Password: creds.Password, Language: creds.Language, ClientSystemID: creds.ClientSystemID}
}

type speedyPhone struct {
	Number string `json:"number"`
}

type speedyAddress struct {
	CountryCode  string `json:"countryCode,omitempty"`
	City         string `json:"city,omitempty"`
	PostCode     string `json:"postCode,omitempty"`
	AddressLine1 string `json:"addressLine1,omitempty"`
	AddressLine2 string `json:"addressLine2,omitempty"`
}

type speedyRecipient struct {
	ClientName string         `json:"clientName"`
	Phone1     speedyPhone    `json:"phone1"`
	Email      string         `json:"email,omitempty"`
	Address    *speedyAddress `json:"address,omitempty"`
	OfficeID   string         `json:"officeId,omitempty"`
}

type speedyService struct {
	ServiceID string `json:"serviceId"`
}

type speedyParcel struct {
	Weight float64 `json:"weight"`
}

type speedyContent struct {
	ParcelsCount int            `json:"parcelsCount"`
	Parcels      []speedyParcel `json:"parcels"`
}

type speedyCOD struct {
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	ProcessingType string  `json:"processingType"`
}

type speedyAdditionalServices struct {
	COD *speedyCOD `json:"cod,omitempty"`
}

type speedyPayment struct {
	CourierServicePayer string `json:"courierServicePayer"`
}

type createShipmentRequest struct {
	speedyAuth
	Recipient          speedyRecipient           `json:"recipient"`
	Service            speedyService             `json:"service"`
	Content            speedyContent             `json:"content"`
	Payment            speedyPayment             `json:"payment"`
	AdditionalServices *speedyAdditionalServices `json:"additionalServices,omitempty"`
	Ref1               string                    `json:"ref1,omitempty"`
}

type speedyParcelInfo struct {
	ParcelID string `json:"parcelId"`
}

type speedyError struct {
	ID      string `json:"id"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type createShipmentResponse struct {
	ID      string             `json:"id"`
	Parcels []speedyParcelInfo `json:"parcels"`
	Error   *speedyError       `json:"error"`
}

func (c *SpeedyHTTPClient) CreateShipment(ctx context.Context, req application.CreateShipmentRequest) (application.ShipmentResult, error) {
	body := createShipmentRequest{
		speedyAuth: authFromCreds(req.Creds),
		Recipient: speedyRecipient{
			ClientName: req.Recipient.ContactName,
			Phone1:     speedyPhone{Number: req.Recipient.Phone},
			Email:      req.Recipient.Email,
		},
		Service: speedyService{ServiceID: req.ServiceID},
		Content: speedyContent{ParcelsCount: 1, Parcels: []speedyParcel{{Weight: req.ParcelWeightKg}}},
		Payment: speedyPayment{CourierServicePayer: "SENDER"},
		Ref1:    req.Ref1,
	}

	if req.Recipient.OfficeID != "" {
		body.Recipient.OfficeID = req.Recipient.OfficeID
	} else {
		body.Recipient.Address = &speedyAddress{
			CountryCode:  req.Recipient.CountryCode,
			City:         req.Recipient.City,
			PostCode:     req.Recipient.PostalCode,
			AddressLine1: req.Recipient.Line1,
			AddressLine2: req.Recipient.Line2,
		}
	}

	if req.RequireCOD {
		body.AdditionalServices = &speedyAdditionalServices{COD: &speedyCOD{
			Amount:         float64(req.CODAmount.AmountMinor) / 100,
			Currency:       req.CODAmount.Currency,
			ProcessingType: "CASH",
		}}
	}

	var resp createShipmentResponse
	if err := c.post(ctx, "/shipment", body, &resp); err != nil {
		return application.ShipmentResult{}, err
	}
	if resp.Error != nil {
		return application.ShipmentResult{}, fmt.Errorf("speedy create shipment failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}
	if len(resp.Parcels) == 0 {
		return application.ShipmentResult{}, fmt.Errorf("speedy create shipment returned no parcels")
	}
	return application.ShipmentResult{ShipmentID: resp.ID, ParcelID: resp.Parcels[0].ParcelID}, nil
}

type trackRequest struct {
	speedyAuth
	Parcels           []speedyParcelInfo `json:"parcels"`
	LastOperationOnly bool               `json:"lastOperationOnly"`
}

type speedyTrackingOperation struct {
	OperationCode int    `json:"operationCode"`
	Description   string `json:"description"`
}

type speedyTrackedParcel struct {
	ParcelID   string                    `json:"parcelId"`
	Operations []speedyTrackingOperation `json:"operations"`
}

type trackResponse struct {
	Parcels []speedyTrackedParcel `json:"parcels"`
	Error   *speedyError          `json:"error"`
}

func (c *SpeedyHTTPClient) Track(ctx context.Context, creds application.Credentials, parcelIDs []string) ([]application.TrackedParcel, error) {
	refs := make([]speedyParcelInfo, 0, len(parcelIDs))
	for _, id := range parcelIDs {
		refs = append(refs, speedyParcelInfo{ParcelID: id})
	}

	var resp trackResponse
	if err := c.post(ctx, "/track", trackRequest{speedyAuth: authFromCreds(creds), Parcels: refs, LastOperationOnly: true}, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("speedy track failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	result := make([]application.TrackedParcel, 0, len(resp.Parcels))
	for _, p := range resp.Parcels {
		if len(p.Operations) == 0 {
			continue
		}
		last := p.Operations[len(p.Operations)-1]
		result = append(result, application.TrackedParcel{
			ParcelID:      p.ParcelID,
			OperationCode: last.OperationCode,
			Description:   last.Description,
		})
	}
	return result, nil
}

type officeSearchRequest struct {
	speedyAuth
	CountryCode string `json:"countryCode,omitempty"`
	City        string `json:"city,omitempty"`
	Type        string `json:"type,omitempty"`
}

type speedyOffice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type officeSearchResponse struct {
	Offices []speedyOffice `json:"offices"`
	Error   *speedyError   `json:"error"`
}

func (c *SpeedyHTTPClient) SearchOffices(ctx context.Context, creds application.Credentials, city, officeType string) ([]application.Office, error) {
	var resp officeSearchResponse
	if err := c.post(ctx, "/location/office", officeSearchRequest{speedyAuth: authFromCreds(creds), City: city, Type: officeType}, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("speedy office search failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	offices := make([]application.Office, 0, len(resp.Offices))
	for _, o := range resp.Offices {
		offices = append(offices, application.Office{ID: o.ID, Name: o.Name, Type: o.Type})
	}
	return offices, nil
}

func (c *SpeedyHTTPClient) post(ctx context.Context, path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, speedyBaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("speedy api returned status %d: %s", resp.StatusCode, strconv.Quote(string(respBody)))
	}
	return json.Unmarshal(respBody, out)
}
