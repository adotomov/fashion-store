package infrastructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/payments/domain"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.PaymentMethod, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, brand, last4, exp_month, exp_year, is_default, created_at, updated_at
		FROM payment_methods WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	methods := []domain.PaymentMethod{}
	for rows.Next() {
		m, err := scanPaymentMethod(rows)
		if err != nil {
			return nil, err
		}
		methods = append(methods, *m)
	}
	return methods, rows.Err()
}

func (r *PostgresRepository) Find(ctx context.Context, userID, id uuid.UUID) (*domain.PaymentMethod, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, brand, last4, exp_month, exp_year, is_default, created_at, updated_at
		FROM payment_methods WHERE id = $1 AND user_id = $2`, id, userID)
	return scanPaymentMethod(row)
}

func (r *PostgresRepository) Create(ctx context.Context, method domain.PaymentMethod) (*domain.PaymentMethod, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if method.IsDefault {
		if _, err := tx.Exec(ctx, `UPDATE payment_methods SET is_default = FALSE WHERE user_id = $1`, method.UserID); err != nil {
			return nil, err
		}
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO payment_methods (user_id, brand, last4, exp_month, exp_year, is_default)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, brand, last4, exp_month, exp_year, is_default, created_at, updated_at`,
		method.UserID, method.Brand, method.Last4, method.ExpMonth, method.ExpYear, method.IsDefault)

	created, err := scanPaymentMethod(row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *PostgresRepository) Update(ctx context.Context, method domain.PaymentMethod) (*domain.PaymentMethod, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if method.IsDefault {
		if _, err := tx.Exec(ctx, `UPDATE payment_methods SET is_default = FALSE WHERE user_id = $1 AND id != $2`,
			method.UserID, method.ID); err != nil {
			return nil, err
		}
	}

	row := tx.QueryRow(ctx, `
		UPDATE payment_methods SET brand = $3, last4 = $4, exp_month = $5, exp_year = $6, is_default = $7, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, brand, last4, exp_month, exp_year, is_default, created_at, updated_at`,
		method.ID, method.UserID, method.Brand, method.Last4, method.ExpMonth, method.ExpYear, method.IsDefault)

	updated, err := scanPaymentMethod(row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM payment_methods WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPaymentMethodNotFound
	}
	return nil
}

func scanPaymentMethod(row pgx.Row) (*domain.PaymentMethod, error) {
	var m domain.PaymentMethod
	err := row.Scan(&m.ID, &m.UserID, &m.Brand, &m.Last4, &m.ExpMonth, &m.ExpYear, &m.IsDefault, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPaymentMethodNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
