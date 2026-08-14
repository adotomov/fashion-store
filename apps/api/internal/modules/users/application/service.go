package application

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/users/domain"
)

type Service struct {
	repo   Repository
	orders OrderCounter
}

// NewService takes an optional OrderCounter (may be nil, e.g. in tests that
// don't exercise the admin user list) to enrich admin views with order counts.
func NewService(repo Repository, orders OrderCounter) *Service {
	return &Service{repo: repo, orders: orders}
}

// AdminUserView pairs a user with the data the admin User Management page
// needs but that doesn't belong on domain.User itself.
type AdminUserView struct {
	User       domain.User
	OrderCount int
}

type ListUsersResult struct {
	Users []AdminUserView
	Total int
}

// EnsureUser returns the existing user for the given email, or creates one.
// Used by the auth module to provision a local user on first login.
func (s *Service) EnsureUser(ctx context.Context, input CreateUserInput) (*domain.User, error) {
	existing, err := s.repo.FindByEmail(ctx, input.Email)
	if err == nil {
		return existing, nil
	}
	if err != domain.ErrUserNotFound {
		return nil, err
	}
	return s.repo.Create(ctx, input)
}

// LocaleByEmail returns the stored preferred locale for the account with this
// email, or empty when there is no such user or no preference. Used to send a
// customer's transactional email in their own language.
func (s *Service) LocaleByEmail(ctx context.Context, email string) string {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil || user == nil {
		return ""
	}
	return user.Locale
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.repo.FindByID(ctx, userID)
}

// GetRoles returns the role names for a user. Used by the auth module to
// build the authenticated principal without trusting token claims.
func (s *Service) GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	roles := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roles = append(roles, string(r))
	}
	return roles, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if input.FullName != nil {
		user.FullName = *input.FullName
	}
	if input.Phone != nil {
		// A phone number is mandatory on the profile so orders always have a
		// reachable contact — reject a caller that tries to clear it.
		if strings.TrimSpace(*input.Phone) == "" {
			return nil, domain.ValidationError("phone is required")
		}
		user.Phone = *input.Phone
	}
	return s.repo.Update(ctx, *user)
}

func (s *Service) orderCountFor(ctx context.Context, userID uuid.UUID) (int, error) {
	if s.orders == nil {
		return 0, nil
	}
	return s.orders.CountOrdersByUser(ctx, userID)
}

func (s *Service) AdminListUsers(ctx context.Context, filter ListUsersFilter) (ListUsersResult, error) {
	users, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return ListUsersResult{}, err
	}

	views := make([]AdminUserView, 0, len(users))
	for _, u := range users {
		count, err := s.orderCountFor(ctx, u.ID)
		if err != nil {
			return ListUsersResult{}, err
		}
		views = append(views, AdminUserView{User: u, OrderCount: count})
	}
	return ListUsersResult{Users: views, Total: total}, nil
}

func (s *Service) AdminGetUser(ctx context.Context, userID uuid.UUID) (AdminUserView, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return AdminUserView{}, err
	}
	count, err := s.orderCountFor(ctx, userID)
	if err != nil {
		return AdminUserView{}, err
	}
	return AdminUserView{User: *user, OrderCount: count}, nil
}

// AdminSetRoles replaces a user's roles wholesale. Used to grant/revoke the
// admin role from the User Management page.
func (s *Service) AdminSetRoles(ctx context.Context, userID uuid.UUID, roles []domain.Role) (*domain.User, error) {
	if len(roles) == 0 {
		return nil, domain.ValidationError("at least one role is required")
	}
	for _, r := range roles {
		if r != domain.RoleUser && r != domain.RoleAdmin && r != domain.RoleAudit && r != domain.RoleAccountant {
			return nil, domain.ValidationError("invalid role: " + string(r))
		}
	}
	return s.repo.SetRoles(ctx, userID, roles)
}

func (s *Service) UserStats(ctx context.Context) (UserStats, error) {
	return s.repo.Stats(ctx)
}

func (s *Service) ListAddresses(ctx context.Context, userID uuid.UUID) ([]domain.Address, error) {
	return s.repo.ListAddresses(ctx, userID)
}

func (s *Service) AddAddress(ctx context.Context, userID uuid.UUID, input AddAddressInput) (*domain.Address, error) {
	addr := addressFromInput(input)
	addr.UserID = userID
	if err := addr.Validate(); err != nil {
		return nil, err
	}
	return s.repo.CreateAddress(ctx, addr)
}

func (s *Service) UpdateAddress(ctx context.Context, userID, addressID uuid.UUID, input UpdateAddressInput) (*domain.Address, error) {
	existing, err := s.repo.FindAddress(ctx, userID, addressID)
	if err != nil {
		return nil, err
	}

	addr := addressFromInput(input)
	addr.ID = existing.ID
	addr.UserID = existing.UserID
	addr.CreatedAt = existing.CreatedAt
	if err := addr.Validate(); err != nil {
		return nil, err
	}
	return s.repo.UpdateAddress(ctx, addr)
}

// addressFromInput maps a submitted structured address onto the domain type,
// defaulting the country to Bulgaria (the only supported market for now).
func addressFromInput(input AddressInput) domain.Address {
	countryCode := input.CountryCode
	if countryCode == "" {
		countryCode = "BG"
	}
	countryID := input.CountryID
	if countryID == 0 {
		countryID = 100
	}
	return domain.Address{
		Label:         input.Label,
		RecipientName: input.RecipientName,
		Phone:         input.Phone,
		CountryCode:   countryCode,
		CountryID:     countryID,
		SiteID:        input.SiteID,
		City:          input.City,
		PostCode:      input.PostCode,
		ComplexID:     input.ComplexID,
		ComplexName:   input.ComplexName,
		StreetID:      input.StreetID,
		StreetName:    input.StreetName,
		StreetNo:      input.StreetNo,
		BlockNo:       input.BlockNo,
		EntranceNo:    input.EntranceNo,
		FloorNo:       input.FloorNo,
		ApartmentNo:   input.ApartmentNo,
		IsDefault:     input.IsDefault,
	}
}

func (s *Service) DeleteAddress(ctx context.Context, userID, addressID uuid.UUID) error {
	return s.repo.DeleteAddress(ctx, userID, addressID)
}
