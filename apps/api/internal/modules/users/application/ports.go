package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/users/domain"
)

// Repository persists users and their addresses.
type Repository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, input CreateUserInput) (*domain.User, error)
	Update(ctx context.Context, user domain.User) (*domain.User, error)

	// List supports the admin User Management page: a free-text search over
	// name/email plus pagination. The int return is the total match count
	// (ignoring pagination) so the frontend can render page controls.
	List(ctx context.Context, filter ListUsersFilter) ([]domain.User, int, error)
	SetRoles(ctx context.Context, userID uuid.UUID, roles []domain.Role) (*domain.User, error)

	ListAddresses(ctx context.Context, userID uuid.UUID) ([]domain.Address, error)
	CreateAddress(ctx context.Context, address domain.Address) (*domain.Address, error)
	UpdateAddress(ctx context.Context, address domain.Address) (*domain.Address, error)
	DeleteAddress(ctx context.Context, userID, addressID uuid.UUID) error
	FindAddress(ctx context.Context, userID, addressID uuid.UUID) (*domain.Address, error)

	Stats(ctx context.Context) (UserStats, error)
}

// OrderCounter is implemented by an adapter onto the orders module so the
// admin User Management page can show a per-user order count without this
// module importing orders' domain or repository directly.
type OrderCounter interface {
	CountOrdersByUser(ctx context.Context, userID uuid.UUID) (int, error)
}

type CreateUserInput struct {
	Email    string
	FullName string
}

type ListUsersFilter struct {
	Search   string
	Page     int
	PageSize int
}

// CountBreakdown is a generic (label, count) pair, e.g. for role or
// customer-country breakdowns on the admin dashboard.
type CountBreakdown struct {
	Label string
	Count int
}

// DailyUserCount is one point in the admin dashboard's daily registrations
// chart.
type DailyUserCount struct {
	Date  time.Time
	Count int
}

// UserStats summarizes the user base for the admin dashboard. ByCountry is
// derived from each user's default shipping address as a proxy, since
// users have no dedicated demographic fields.
type UserStats struct {
	TotalUsers         int
	New24h             int
	New7d              int
	New30d             int
	RoleBreakdown      []CountBreakdown
	ByCountry          []CountBreakdown
	DailyRegistrations []DailyUserCount
}
