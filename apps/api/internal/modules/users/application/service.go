package application

import (
	"context"

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
		if r != domain.RoleUser && r != domain.RoleAdmin {
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
	addr := domain.Address{
		UserID:        userID,
		Label:         input.Label,
		RecipientName: input.RecipientName,
		Phone:         input.Phone,
		Line1:         input.Line1,
		Line2:         input.Line2,
		City:          input.City,
		Region:        input.Region,
		PostalCode:    input.PostalCode,
		CountryCode:   input.CountryCode,
		IsDefault:     input.IsDefault,
	}
	if err := addr.Validate(); err != nil {
		return nil, err
	}
	return s.repo.CreateAddress(ctx, addr)
}

func (s *Service) UpdateAddress(ctx context.Context, userID, addressID uuid.UUID, input UpdateAddressInput) (*domain.Address, error) {
	addr, err := s.repo.FindAddress(ctx, userID, addressID)
	if err != nil {
		return nil, err
	}

	if input.Label != nil {
		addr.Label = *input.Label
	}
	if input.RecipientName != nil {
		addr.RecipientName = *input.RecipientName
	}
	if input.Phone != nil {
		addr.Phone = *input.Phone
	}
	if input.Line1 != nil {
		addr.Line1 = *input.Line1
	}
	if input.Line2 != nil {
		addr.Line2 = *input.Line2
	}
	if input.City != nil {
		addr.City = *input.City
	}
	if input.Region != nil {
		addr.Region = *input.Region
	}
	if input.PostalCode != nil {
		addr.PostalCode = *input.PostalCode
	}
	if input.CountryCode != nil {
		addr.CountryCode = *input.CountryCode
	}
	if input.IsDefault != nil {
		addr.IsDefault = *input.IsDefault
	}

	if err := addr.Validate(); err != nil {
		return nil, err
	}

	return s.repo.UpdateAddress(ctx, *addr)
}

func (s *Service) DeleteAddress(ctx context.Context, userID, addressID uuid.UUID) error {
	return s.repo.DeleteAddress(ctx, userID, addressID)
}
