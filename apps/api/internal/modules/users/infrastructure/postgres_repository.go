package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/users/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/users/domain"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Stats(ctx context.Context) (application.UserStats, error) {
	var stats application.UserStats
	now := time.Now()

	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*),
			COUNT(*) FILTER (WHERE created_at >= $1),
			COUNT(*) FILTER (WHERE created_at >= $2),
			COUNT(*) FILTER (WHERE created_at >= $3)
		FROM users`,
		now.AddDate(0, 0, -1), now.AddDate(0, 0, -7), now.AddDate(0, 0, -30),
	).Scan(&stats.TotalUsers, &stats.New24h, &stats.New7d, &stats.New30d); err != nil {
		return stats, err
	}

	roleRows, err := r.db.Query(ctx, `SELECT role, COUNT(*) FROM user_roles GROUP BY role ORDER BY COUNT(*) DESC`)
	if err != nil {
		return stats, err
	}
	stats.RoleBreakdown, err = scanCountBreakdown(roleRows)
	if err != nil {
		return stats, err
	}

	countryRows, err := r.db.Query(ctx, `
		SELECT country_code, COUNT(*) FROM user_addresses WHERE is_default AND country_code <> ''
		GROUP BY country_code ORDER BY COUNT(*) DESC`)
	if err != nil {
		return stats, err
	}
	stats.ByCountry, err = scanCountBreakdown(countryRows)
	if err != nil {
		return stats, err
	}

	dailyRows, err := r.db.Query(ctx, `
		SELECT date_trunc('day', created_at) AS day, COUNT(*)
		FROM users WHERE created_at >= $1 GROUP BY day ORDER BY day`, now.AddDate(0, 0, -30))
	if err != nil {
		return stats, err
	}
	defer dailyRows.Close()
	for dailyRows.Next() {
		var d application.DailyUserCount
		if err := dailyRows.Scan(&d.Date, &d.Count); err != nil {
			return stats, err
		}
		stats.DailyRegistrations = append(stats.DailyRegistrations, d)
	}
	if err := dailyRows.Err(); err != nil {
		return stats, err
	}

	return stats, nil
}

func scanCountBreakdown(rows pgx.Rows) ([]application.CountBreakdown, error) {
	defer rows.Close()
	result := []application.CountBreakdown{}
	for rows.Next() {
		var b application.CountBreakdown
		if err := rows.Scan(&b.Label, &b.Count); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, email, COALESCE(full_name, ''), COALESCE(phone, ''), created_at, updated_at
		FROM users WHERE id = $1`, id)

	user, err := scanUser(row)
	if err != nil {
		return nil, err
	}

	roles, err := r.rolesFor(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	return user, nil
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, email, COALESCE(full_name, ''), COALESCE(phone, ''), created_at, updated_at
		FROM users WHERE lower(email) = lower($1)`, email)

	user, err := scanUser(row)
	if err != nil {
		return nil, err
	}

	roles, err := r.rolesFor(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	return user, nil
}

func (r *PostgresRepository) Create(ctx context.Context, input application.CreateUserInput) (*domain.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO users (email, full_name)
		VALUES ($1, $2)
		RETURNING id, email, COALESCE(full_name, ''), COALESCE(phone, ''), created_at, updated_at`,
		input.Email, input.FullName)

	user, err := scanUser(row)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`,
		user.ID, domain.RoleUser); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	user.Roles = []domain.Role{domain.RoleUser}

	return user, nil
}

func (r *PostgresRepository) Update(ctx context.Context, user domain.User) (*domain.User, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE users SET full_name = $2, phone = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, COALESCE(full_name, ''), COALESCE(phone, ''), created_at, updated_at`,
		user.ID, user.FullName, user.Phone)

	updated, err := scanUser(row)
	if err != nil {
		return nil, err
	}

	roles, err := r.rolesFor(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	updated.Roles = roles

	return updated, nil
}

// List supports the admin User Management page: free-text search over name
// and email, paginated, ordered newest-first.
func (r *PostgresRepository) List(ctx context.Context, filter application.ListUsersFilter) ([]domain.User, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	search := "%" + filter.Search + "%"

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE $1 = '' OR full_name ILIKE $2 OR email ILIKE $2`,
		filter.Search, search).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, email, COALESCE(full_name, ''), COALESCE(phone, ''), created_at, updated_at
		FROM users
		WHERE $1 = '' OR full_name ILIKE $2 OR email ILIKE $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`,
		filter.Search, search, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	for i := range users {
		roles, err := r.rolesFor(ctx, users[i].ID)
		if err != nil {
			return nil, 0, err
		}
		users[i].Roles = roles
	}

	return users, total, nil
}

func (r *PostgresRepository) SetRoles(ctx context.Context, userID uuid.UUID, roles []domain.Role) (*domain.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
		return nil, err
	}
	for _, role := range roles {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, userID, role); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.FindByID(ctx, userID)
}

func (r *PostgresRepository) rolesFor(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	rows, err := r.db.Query(ctx, `SELECT role FROM user_roles WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, domain.Role(role))
	}
	return roles, rows.Err()
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.FullName, &u.Phone, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *PostgresRepository) ListAddresses(ctx context.Context, userID uuid.UUID) ([]domain.Address, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, label, recipient_name, phone, line1, line2, city, region,
		       postal_code, country_code, is_default, created_at, updated_at
		FROM user_addresses WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []domain.Address
	for rows.Next() {
		addr, err := scanAddress(rows)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, *addr)
	}
	return addresses, rows.Err()
}

func (r *PostgresRepository) CreateAddress(ctx context.Context, address domain.Address) (*domain.Address, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO user_addresses (user_id, label, recipient_name, phone, line1, line2,
			city, region, postal_code, country_code, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, user_id, label, recipient_name, phone, line1, line2, city, region,
		          postal_code, country_code, is_default, created_at, updated_at`,
		address.UserID, address.Label, address.RecipientName, address.Phone, address.Line1,
		address.Line2, address.City, address.Region, address.PostalCode, address.CountryCode,
		address.IsDefault)

	return scanAddress(row)
}

func (r *PostgresRepository) UpdateAddress(ctx context.Context, address domain.Address) (*domain.Address, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE user_addresses SET
			label = $3, recipient_name = $4, phone = $5, line1 = $6, line2 = $7,
			city = $8, region = $9, postal_code = $10, country_code = $11,
			is_default = $12, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, label, recipient_name, phone, line1, line2, city, region,
		          postal_code, country_code, is_default, created_at, updated_at`,
		address.ID, address.UserID, address.Label, address.RecipientName, address.Phone,
		address.Line1, address.Line2, address.City, address.Region, address.PostalCode,
		address.CountryCode, address.IsDefault)

	return scanAddress(row)
}

func (r *PostgresRepository) DeleteAddress(ctx context.Context, userID, addressID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM user_addresses WHERE id = $1 AND user_id = $2`, addressID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAddressNotFound
	}
	return nil
}

func (r *PostgresRepository) FindAddress(ctx context.Context, userID, addressID uuid.UUID) (*domain.Address, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, label, recipient_name, phone, line1, line2, city, region,
		       postal_code, country_code, is_default, created_at, updated_at
		FROM user_addresses WHERE id = $1 AND user_id = $2`, addressID, userID)

	return scanAddress(row)
}

func scanAddress(row pgx.Row) (*domain.Address, error) {
	var a domain.Address
	err := row.Scan(&a.ID, &a.UserID, &a.Label, &a.RecipientName, &a.Phone, &a.Line1, &a.Line2,
		&a.City, &a.Region, &a.PostalCode, &a.CountryCode, &a.IsDefault, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAddressNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
