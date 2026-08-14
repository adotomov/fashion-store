package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/fulfillment/application"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
	// otelhttp transport emits a client span per outbound call and propagates
	// trace context; it is a no-op when tracing is disabled.
	return &SpeedyHTTPClient{httpClient: &http.Client{
		Timeout:   15 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}}
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

// speedyAddress is Speedy's structured (code-based) address. We send resolved
// location IDs — siteId (city), complexId (кв./жк.), streetId — plus the
// free-text house details, so Speedy never has to guess-resolve typed text.
type speedyAddress struct {
	CountryID   int64  `json:"countryId,omitempty"`
	SiteID      int64  `json:"siteId,omitempty"`
	ComplexID   int64  `json:"complexId,omitempty"`
	StreetID    int64  `json:"streetId,omitempty"`
	StreetNo    string `json:"streetNo,omitempty"`
	BlockNo     string `json:"blockNo,omitempty"`
	EntranceNo  string `json:"entranceNo,omitempty"`
	FloorNo     string `json:"floorNo,omitempty"`
	ApartmentNo string `json:"apartmentNo,omitempty"`
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
			CountryID:   req.Recipient.CountryID,
			SiteID:      req.Recipient.SiteID,
			ComplexID:   req.Recipient.ComplexID,
			StreetID:    req.Recipient.StreetID,
			StreetNo:    req.Recipient.StreetNo,
			BlockNo:     req.Recipient.BlockNo,
			EntranceNo:  req.Recipient.EntranceNo,
			FloorNo:     req.Recipient.FloorNo,
			ApartmentNo: req.Recipient.ApartmentNo,
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

type calculateContent struct {
	ParcelsCount int     `json:"parcelsCount"`
	TotalWeight  float64 `json:"totalWeight"`
}

// calculateRecipient is a minimal recipient for pricing — just the destination
// site. Unlike a shipment it carries no contact name/phone. Note the
// destination is keyed "addressLocation" here, not "address" as in the
// shipment API — the /calculate endpoint rejects "address" with code 120.
type calculateRecipient struct {
	PrivatePerson bool           `json:"privatePerson"`
	Address       *speedyAddress `json:"addressLocation,omitempty"`
}

// calculateService wraps the requested service IDs. The /calculate endpoint
// expects an object { "serviceIds": [...] } and rejects a bare array with a
// deserialization error, unlike the shipment API's scalar "service".
type calculateService struct {
	ServiceIDs []int64 `json:"serviceIds"`
}

type calculateRequest struct {
	speedyAuth
	// Sender omitted on purpose — Speedy uses the account's default pickup.
	Recipient calculateRecipient `json:"recipient"`
	Service   calculateService   `json:"service"`
	Content   calculateContent   `json:"content"`
	Payment   speedyPayment      `json:"payment"`
}

type speedyPrice struct {
	Total    float64 `json:"total"`
	Currency string  `json:"currency"`
}

type speedyCalculation struct {
	ServiceID int64        `json:"serviceId"`
	Price     *speedyPrice `json:"price"`
	Error     *speedyError `json:"error"`
}

type calculateResponse struct {
	Calculations []speedyCalculation `json:"calculations"`
	Error        *speedyError        `json:"error"`
}

func (c *SpeedyHTTPClient) Calculate(ctx context.Context, req application.CalculateRequest) ([]application.CalculationResult, error) {
	body := calculateRequest{
		speedyAuth: authFromCreds(req.Creds),
		Recipient: calculateRecipient{
			PrivatePerson: true,
			Address:       &speedyAddress{CountryID: speedyCountryBG, SiteID: req.SiteID},
		},
		Service: calculateService{ServiceIDs: req.ServiceIDs},
		Content: calculateContent{ParcelsCount: 1, TotalWeight: req.WeightKg},
		Payment: speedyPayment{CourierServicePayer: "SENDER"},
	}

	var resp calculateResponse
	if err := c.post(ctx, "/calculate", body, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("speedy calculate failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	results := make([]application.CalculationResult, 0, len(resp.Calculations))
	for _, calc := range resp.Calculations {
		// A single request can return both priced and errored services; keep
		// only the ones Speedy could actually price.
		if calc.Error != nil || calc.Price == nil {
			continue
		}
		results = append(results, application.CalculationResult{
			ServiceID: calc.ServiceID,
			Amount:    money.Money{AmountMinor: int64(math.Round(calc.Price.Total * 100)), Currency: calc.Price.Currency},
		})
	}
	return results, nil
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

// speedyCountryBG is Speedy's numeric country code for Bulgaria; the store
// ships domestically only for now, so every location lookup is scoped to it.
const speedyCountryBG = 100

type officeSearchRequest struct {
	speedyAuth
	CountryID int64  `json:"countryId,omitempty"`
	SiteID    int64  `json:"siteId,omitempty"`
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
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

func (c *SpeedyHTTPClient) SearchOffices(ctx context.Context, creds application.Credentials, siteID int64, name, officeType string) ([]application.Office, error) {
	var resp officeSearchResponse
	req := officeSearchRequest{speedyAuth: authFromCreds(creds), CountryID: speedyCountryBG, SiteID: siteID, Name: name, Type: officeType}
	if err := c.post(ctx, "/location/office", req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("speedy office search failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	// Offices and EasyBox lockers are returned by the same endpoint,
	// distinguished only by Type ("OFFICE" vs "APT"). We pass Type as a request
	// filter above, but defensively re-filter here too: if Speedy ever ignores
	// the request-side filter we must not leak lockers into the office picker
	// (or vice versa), since the caller relies on a single-type list.
	offices := make([]application.Office, 0, len(resp.Offices))
	for _, o := range resp.Offices {
		if officeType != "" && o.Type != officeType {
			continue
		}
		offices = append(offices, application.Office{ID: o.ID, Name: o.Name, Type: o.Type})
	}
	return offices, nil
}

type siteSearchRequest struct {
	speedyAuth
	CountryID int64  `json:"countryId"`
	Name      string `json:"name,omitempty"`
}

type speedySite struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Municipality string `json:"municipality"`
	Region       string `json:"region"`
	PostCode     string `json:"postCode"`
}

type siteSearchResponse struct {
	Sites []speedySite `json:"sites"`
	Error *speedyError `json:"error"`
}

func (c *SpeedyHTTPClient) SearchSites(ctx context.Context, creds application.Credentials, name string) ([]application.Site, error) {
	var resp siteSearchResponse
	req := siteSearchRequest{speedyAuth: authFromCreds(creds), CountryID: speedyCountryBG, Name: name}
	if err := c.post(ctx, "/location/site", req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("speedy site search failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	sites := make([]application.Site, 0, len(resp.Sites))
	for _, s := range resp.Sites {
		sites = append(sites, application.Site{
			ID: s.ID, Name: s.Name, Type: s.Type,
			Municipality: s.Municipality, Region: s.Region, PostCode: s.PostCode,
		})
	}
	return sites, nil
}

type complexSearchRequest struct {
	speedyAuth
	SiteID int64  `json:"siteId"`
	Name   string `json:"name,omitempty"`
}

type speedyComplex struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type complexSearchResponse struct {
	Complexes []speedyComplex `json:"complexes"`
	Error     *speedyError    `json:"error"`
}

func (c *SpeedyHTTPClient) SearchComplexes(ctx context.Context, creds application.Credentials, siteID int64, name string) ([]application.Complex, error) {
	var resp complexSearchResponse
	req := complexSearchRequest{speedyAuth: authFromCreds(creds), SiteID: siteID, Name: name}
	if err := c.post(ctx, "/location/complex", req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("speedy complex search failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	complexes := make([]application.Complex, 0, len(resp.Complexes))
	for _, x := range resp.Complexes {
		complexes = append(complexes, application.Complex{ID: x.ID, Name: x.Name, Type: x.Type})
	}
	return complexes, nil
}

type streetSearchRequest struct {
	speedyAuth
	SiteID int64  `json:"siteId"`
	Name   string `json:"name,omitempty"`
}

type speedyStreet struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type streetSearchResponse struct {
	Streets []speedyStreet `json:"streets"`
	Error   *speedyError   `json:"error"`
}

func (c *SpeedyHTTPClient) SearchStreets(ctx context.Context, creds application.Credentials, siteID int64, name string) ([]application.Street, error) {
	var resp streetSearchResponse
	req := streetSearchRequest{speedyAuth: authFromCreds(creds), SiteID: siteID, Name: name}
	if err := c.post(ctx, "/location/street", req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("speedy street search failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	streets := make([]application.Street, 0, len(resp.Streets))
	for _, s := range resp.Streets {
		streets = append(streets, application.Street{ID: s.ID, Name: s.Name, Type: s.Type})
	}
	return streets, nil
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
