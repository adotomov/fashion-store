package infrastructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/inventory/domain"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateItem(ctx context.Context, item domain.InventoryItem) (*domain.InventoryItem, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO inventory_items (variant_id, sku)
		VALUES ($1, $2)
		RETURNING id, variant_id, sku, quantity_on_hand, quantity_reserved, created_at, updated_at`,
		item.VariantID, item.SKU)

	created, err := scanItem(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505": // unique_violation
				if pgErr.ConstraintName == "inventory_items_sku_idx" {
					return nil, domain.ErrSKUConflict
				}
				return nil, domain.ErrItemAlreadyExists
			case "23503": // foreign_key_violation
				return nil, domain.ErrVariantNotFound
			}
		}
		return nil, err
	}
	return created, nil
}

const listItemsQuery = `
	SELECT i.id, i.variant_id, i.sku, i.quantity_on_hand, i.quantity_reserved, i.created_at, i.updated_at,
	       p.name AS product_name,
	       COALESCE(string_agg(av.value, ' / ' ORDER BY av.value), '') AS variant_label
	FROM inventory_items i
	JOIN product_variants v ON v.id = i.variant_id
	JOIN products p ON p.id = v.product_id
	LEFT JOIN variant_attribute_values vav ON vav.variant_id = v.id
	LEFT JOIN attribute_values av ON av.id = vav.attribute_value_id
	GROUP BY i.id, p.name`

func (r *PostgresRepository) ListItems(ctx context.Context) ([]domain.InventoryItem, error) {
	rows, err := r.db.Query(ctx, listItemsQuery+` ORDER BY i.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.InventoryItem
	for rows.Next() {
		item, err := scanItemWithDisplay(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.InventoryItem, error) {
	row := r.db.QueryRow(ctx, listItemsQuery+` HAVING i.id = $1`, id)
	return scanItemWithDisplay(row)
}

func (r *PostgresRepository) FindByVariantID(ctx context.Context, variantID uuid.UUID) (*domain.InventoryItem, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, variant_id, sku, quantity_on_hand, quantity_reserved, created_at, updated_at
		FROM inventory_items WHERE variant_id = $1`, variantID)
	return scanItem(row)
}

func (r *PostgresRepository) UpdateSKU(ctx context.Context, id uuid.UUID, sku string) (*domain.InventoryItem, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE inventory_items SET sku = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, variant_id, sku, quantity_on_hand, quantity_reserved, created_at, updated_at`,
		id, sku)

	updated, err := scanItem(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrSKUConflict
		}
		return nil, err
	}
	return updated, nil
}

func (r *PostgresRepository) AdjustStock(ctx context.Context, itemID uuid.UUID, movementType domain.MovementType, quantityDelta int, note string, createdBy *uuid.UUID) (*domain.InventoryItem, *domain.InventoryMovement, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	var currentOnHand int
	if err := tx.QueryRow(ctx, `SELECT quantity_on_hand FROM inventory_items WHERE id = $1 FOR UPDATE`, itemID).Scan(&currentOnHand); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, domain.ErrItemNotFound
		}
		return nil, nil, err
	}

	if currentOnHand+quantityDelta < 0 {
		return nil, nil, domain.ErrInsufficientStock
	}

	movementRow := tx.QueryRow(ctx, `
		INSERT INTO inventory_movements (inventory_item_id, type, quantity_delta, note, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, inventory_item_id, type, quantity_delta, COALESCE(note, ''), created_by, created_at`,
		itemID, movementType, quantityDelta, note, createdBy)

	movement, err := scanMovement(movementRow)
	if err != nil {
		return nil, nil, err
	}

	itemRow := tx.QueryRow(ctx, `
		UPDATE inventory_items SET quantity_on_hand = quantity_on_hand + $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, variant_id, sku, quantity_on_hand, quantity_reserved, created_at, updated_at`,
		itemID, quantityDelta)

	item, err := scanItem(itemRow)
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return item, movement, nil
}

func (r *PostgresRepository) ListMovements(ctx context.Context, itemID uuid.UUID) ([]domain.InventoryMovement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, inventory_item_id, type, quantity_delta, COALESCE(note, ''), created_by, created_at
		FROM inventory_movements WHERE inventory_item_id = $1 ORDER BY created_at DESC`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	movements := []domain.InventoryMovement{}
	for rows.Next() {
		m, err := scanMovement(rows)
		if err != nil {
			return nil, err
		}
		movements = append(movements, *m)
	}
	return movements, rows.Err()
}

// ReserveForVariants holds stock for each tracked line in one transaction:
// it locks each inventory_items row, checks available stock, and bails out
// (rolling back) the moment any tracked line can't be satisfied — no
// partial reservations. Untracked variants (no inventory_items row) are
// skipped entirely, same as the cart module's "untracked = unlimited"
// semantics.
func (r *PostgresRepository) ReserveForVariants(ctx context.Context, lines []application.ReserveLine, createdBy *uuid.UUID) (*domain.Reservation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	type trackedLine struct {
		inventoryItemID uuid.UUID
		quantity        int
	}
	var tracked []trackedLine

	for _, line := range lines {
		var itemID uuid.UUID
		var onHand, reserved int
		err := tx.QueryRow(ctx, `
			SELECT id, quantity_on_hand, quantity_reserved FROM inventory_items WHERE variant_id = $1 FOR UPDATE`,
			line.VariantID).Scan(&itemID, &onHand, &reserved)
		if errors.Is(err, pgx.ErrNoRows) {
			continue // untracked variant: no limit, nothing to reserve
		}
		if err != nil {
			return nil, err
		}
		if onHand-reserved < line.Quantity {
			return nil, domain.ErrInsufficientStock
		}
		tracked = append(tracked, trackedLine{inventoryItemID: itemID, quantity: line.Quantity})
	}

	var reservation domain.Reservation
	if err := tx.QueryRow(ctx, `
		INSERT INTO inventory_reservations (status) VALUES ($1)
		RETURNING id, status, expires_at, created_at, updated_at`,
		domain.ReservationPending).Scan(&reservation.ID, &reservation.Status, &reservation.ExpiresAt, &reservation.CreatedAt, &reservation.UpdatedAt); err != nil {
		return nil, err
	}

	for _, t := range tracked {
		var reservationItemID uuid.UUID
		if err := tx.QueryRow(ctx, `
			INSERT INTO inventory_reservation_items (reservation_id, inventory_item_id, quantity)
			VALUES ($1, $2, $3) RETURNING id`,
			reservation.ID, t.inventoryItemID, t.quantity).Scan(&reservationItemID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_items SET quantity_reserved = quantity_reserved + $2, updated_at = NOW() WHERE id = $1`,
			t.inventoryItemID, t.quantity); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO inventory_movements (inventory_item_id, type, quantity_delta, note, created_by)
			VALUES ($1, $2, $3, $4, $5)`,
			t.inventoryItemID, domain.MovementReservation, -t.quantity, "Checkout reservation", createdBy); err != nil {
			return nil, err
		}
		reservation.Items = append(reservation.Items, domain.ReservationItem{
			ID: reservationItemID, ReservationID: reservation.ID, InventoryItemID: t.inventoryItemID, Quantity: t.quantity,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &reservation, nil
}

func (r *PostgresRepository) reservationItems(ctx context.Context, tx pgx.Tx, reservationID uuid.UUID) ([]domain.ReservationItem, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, reservation_id, inventory_item_id, quantity FROM inventory_reservation_items WHERE reservation_id = $1`,
		reservationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.ReservationItem
	for rows.Next() {
		var item domain.ReservationItem
		if err := rows.Scan(&item.ID, &item.ReservationID, &item.InventoryItemID, &item.Quantity); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// CommitReservation permanently consumes held stock: quantity_on_hand and
// quantity_reserved both drop by the reserved quantity, since the items
// have now actually been sold.
func (r *PostgresRepository) CommitReservation(ctx context.Context, reservationID uuid.UUID, createdBy *uuid.UUID) error {
	return r.finalizeReservation(ctx, reservationID, createdBy, domain.ReservationCommitted)
}

// ReleaseReservation gives back held stock without touching
// quantity_on_hand — used when an order is never actually placed (e.g. a
// declined card_online charge).
func (r *PostgresRepository) ReleaseReservation(ctx context.Context, reservationID uuid.UUID, createdBy *uuid.UUID) error {
	return r.finalizeReservation(ctx, reservationID, createdBy, domain.ReservationReleased)
}

func (r *PostgresRepository) finalizeReservation(ctx context.Context, reservationID uuid.UUID, createdBy *uuid.UUID, outcome domain.ReservationStatus) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status domain.ReservationStatus
	if err := tx.QueryRow(ctx, `SELECT status FROM inventory_reservations WHERE id = $1 FOR UPDATE`, reservationID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrReservationNotFound
		}
		return err
	}
	if status != domain.ReservationPending {
		return domain.ErrReservationNotPending
	}

	items, err := r.reservationItems(ctx, tx, reservationID)
	if err != nil {
		return err
	}

	for _, item := range items {
		if outcome == domain.ReservationCommitted {
			if _, err := tx.Exec(ctx, `
				UPDATE inventory_items SET quantity_on_hand = quantity_on_hand - $2, quantity_reserved = quantity_reserved - $2, updated_at = NOW()
				WHERE id = $1`, item.InventoryItemID, item.Quantity); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO inventory_movements (inventory_item_id, type, quantity_delta, note, created_by)
				VALUES ($1, $2, $3, $4, $5)`,
				item.InventoryItemID, domain.MovementSaleCommitted, -item.Quantity, "Order placed", createdBy); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(ctx, `
				UPDATE inventory_items SET quantity_reserved = quantity_reserved - $2, updated_at = NOW()
				WHERE id = $1`, item.InventoryItemID, item.Quantity); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO inventory_movements (inventory_item_id, type, quantity_delta, note, created_by)
				VALUES ($1, $2, $3, $4, $5)`,
				item.InventoryItemID, domain.MovementReservationRelease, item.Quantity, "Reservation released", createdBy); err != nil {
				return err
			}
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE inventory_reservations SET status = $2, updated_at = NOW() WHERE id = $1`, reservationID, outcome); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func scanItem(row pgx.Row) (*domain.InventoryItem, error) {
	var i domain.InventoryItem
	err := row.Scan(&i.ID, &i.VariantID, &i.SKU, &i.QuantityOnHand, &i.QuantityReserved, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func scanItemWithDisplay(row pgx.Row) (*domain.InventoryItem, error) {
	var i domain.InventoryItem
	err := row.Scan(&i.ID, &i.VariantID, &i.SKU, &i.QuantityOnHand, &i.QuantityReserved, &i.CreatedAt, &i.UpdatedAt,
		&i.ProductName, &i.VariantLabel)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func scanMovement(row pgx.Row) (*domain.InventoryMovement, error) {
	var m domain.InventoryMovement
	err := row.Scan(&m.ID, &m.InventoryItemID, &m.Type, &m.QuantityDelta, &m.Note, &m.CreatedBy, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
