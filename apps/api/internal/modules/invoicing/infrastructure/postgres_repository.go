package infrastructure

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/invoicing/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/invoicing/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/shared/money"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// ── Invoice ──────────────────────────────────────────────────────────────────

const invoiceColumns = `
	id, invoice_number, document_type, order_id, storno_of_invoice_id,
	order_number, placed_at, payment_method,
	card_provider, card_provider_reference, courier_name, courier_identifier,
	company_name, company_legal_type, company_eik, company_address, company_email, company_phone,
	nra_store_number, vat_number, vat_rate,
	recipient_name, recipient_address, recipient_email,
	subtotal_excl_vat_minor, vat_amount_minor, total_incl_vat_minor, currency,
	delivery_fee_minor, discount_amount_minor,
	created_at`

func (r *PostgresRepository) CreateInvoice(ctx context.Context, inv domain.Invoice) (*domain.Invoice, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var discountMinor *int64
	if inv.DiscountAmount != nil {
		discountMinor = &inv.DiscountAmount.AmountMinor
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO invoices (
			document_type, order_id, storno_of_invoice_id,
			order_number, placed_at, payment_method,
			card_provider, card_provider_reference, courier_name, courier_identifier,
			company_name, company_legal_type, company_eik, company_address, company_email, company_phone,
			nra_store_number, vat_number, vat_rate,
			recipient_name, recipient_address, recipient_email,
			subtotal_excl_vat_minor, vat_amount_minor, total_incl_vat_minor, currency,
			delivery_fee_minor, discount_amount_minor
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23,$24,$25,$26,$27,$28
		) RETURNING `+invoiceColumns,
		string(inv.DocumentType), inv.OrderID, inv.StornoOfInvoiceID,
		inv.OrderNumber, inv.PlacedAt, inv.PaymentMethod,
		inv.CardProvider, inv.CardProviderReference, inv.CourierName, inv.CourierIdentifier,
		inv.CompanyName, inv.CompanyLegalType, inv.CompanyEIK, inv.CompanyAddress, inv.CompanyEmail, inv.CompanyPhone,
		inv.NRAStoreNumber, inv.VATNumber, inv.VATRate,
		inv.RecipientName, inv.RecipientAddress, inv.RecipientEmail,
		inv.SubtotalExclVAT.AmountMinor, inv.VATAmount.AmountMinor, inv.TotalInclVAT.AmountMinor, inv.TotalInclVAT.Currency,
		inv.DeliveryFee.AmountMinor, discountMinor,
	)

	created, err := scanInvoice(row)
	if err != nil {
		return nil, err
	}

	for _, item := range inv.LineItems {
		itemRow := tx.QueryRow(ctx, `
			INSERT INTO invoice_line_items (
				invoice_id, product_name, variant_label, quantity,
				unit_price_incl_vat_minor, unit_price_excl_vat_minor, vat_per_unit_minor,
				line_total_incl_vat_minor, line_total_excl_vat_minor, line_vat_amount_minor,
				vat_rate, sort_order
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			RETURNING id, invoice_id, product_name, variant_label, quantity,
				unit_price_incl_vat_minor, unit_price_excl_vat_minor, vat_per_unit_minor,
				line_total_incl_vat_minor, line_total_excl_vat_minor, line_vat_amount_minor,
				vat_rate, sort_order, created_at`,
			created.ID, item.ProductName, item.VariantLabel, item.Quantity,
			item.UnitPriceInclVAT.AmountMinor, item.UnitPriceExclVAT.AmountMinor, item.VATPerUnit.AmountMinor,
			item.LineTotalInclVAT.AmountMinor, item.LineTotalExclVAT.AmountMinor, item.LineVATAmount.AmountMinor,
			item.VATRate, item.SortOrder,
		)
		scannedItem, err := scanLineItem(itemRow, created.TotalInclVAT.Currency)
		if err != nil {
			return nil, err
		}
		created.LineItems = append(created.LineItems, *scannedItem)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	row := r.db.QueryRow(ctx, `SELECT `+invoiceColumns+` FROM invoices WHERE id = $1`, id)
	inv, err := scanInvoice(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.attachLineItems(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

func (r *PostgresRepository) FindByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Invoice, error) {
	row := r.db.QueryRow(ctx, `SELECT `+invoiceColumns+` FROM invoices WHERE order_id = $1 AND document_type = 'фактура' LIMIT 1`, orderID)
	inv, err := scanInvoice(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.attachLineItems(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

func (r *PostgresRepository) List(ctx context.Context, filter application.ListFilter) ([]domain.Invoice, error) {
	query := `SELECT ` + invoiceColumns + ` FROM invoices WHERE
		($1::timestamptz IS NULL OR created_at >= $1) AND
		($2::timestamptz IS NULL OR created_at <= $2) AND
		($3::text IS NULL OR document_type = $3) AND
		($4::text IS NULL OR payment_method = $4) AND
		($5::text = '' OR invoice_number ILIKE $5 || '%' OR order_number ILIKE $5 || '%')
		ORDER BY created_at DESC
		LIMIT $6 OFFSET $7`

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.Query(ctx, query,
		filter.From, filter.To, filter.DocumentType, filter.PaymentMethod,
		filter.Search, limit, filter.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []domain.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, *inv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// No line items attached in list view for performance
	return invoices, nil
}

func (r *PostgresRepository) attachLineItems(ctx context.Context, inv *domain.Invoice) error {
	rows, err := r.db.Query(ctx, `
		SELECT id, invoice_id, product_name, variant_label, quantity,
			unit_price_incl_vat_minor, unit_price_excl_vat_minor, vat_per_unit_minor,
			line_total_incl_vat_minor, line_total_excl_vat_minor, line_vat_amount_minor,
			vat_rate, sort_order, created_at
		FROM invoice_line_items WHERE invoice_id = $1 ORDER BY sort_order`, inv.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanLineItem(rows, inv.TotalInclVAT.Currency)
		if err != nil {
			return err
		}
		inv.LineItems = append(inv.LineItems, *item)
	}
	return rows.Err()
}

// ── Settings ─────────────────────────────────────────────────────────────────

func (r *PostgresRepository) GetSettings(ctx context.Context) (domain.InvoiceSettings, error) {
	var s domain.InvoiceSettings
	err := r.db.QueryRow(ctx, `
		SELECT company_name, company_legal_type, company_eik,
		       company_address_street, company_address_city, company_address_postal_code, company_address_country,
		       company_email, company_phone, nra_store_number, vat_number, vat_rate
		FROM invoice_settings LIMIT 1`,
	).Scan(&s.CompanyName, &s.CompanyLegalType, &s.CompanyEIK,
		&s.CompanyAddressStreet, &s.CompanyAddressCity, &s.CompanyAddressPostalCode, &s.CompanyAddressCountry,
		&s.CompanyEmail, &s.CompanyPhone, &s.NRAStoreNumber, &s.VATNumber, &s.VATRate)
	return s, err
}

func (r *PostgresRepository) SaveSettings(ctx context.Context, s domain.InvoiceSettings) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO invoice_settings (id, company_name, company_legal_type, company_eik,
		    company_address_street, company_address_city, company_address_postal_code, company_address_country,
		    company_email, company_phone, nra_store_number, vat_number, vat_rate, updated_at)
		VALUES (TRUE, $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, NOW())
		ON CONFLICT (id) DO UPDATE SET
		    company_name = EXCLUDED.company_name,
		    company_legal_type = EXCLUDED.company_legal_type,
		    company_eik = EXCLUDED.company_eik,
		    company_address_street = EXCLUDED.company_address_street,
		    company_address_city = EXCLUDED.company_address_city,
		    company_address_postal_code = EXCLUDED.company_address_postal_code,
		    company_address_country = EXCLUDED.company_address_country,
		    company_email = EXCLUDED.company_email,
		    company_phone = EXCLUDED.company_phone,
		    nra_store_number = EXCLUDED.nra_store_number,
		    vat_number = EXCLUDED.vat_number,
		    vat_rate = EXCLUDED.vat_rate,
		    updated_at = NOW()`,
		s.CompanyName, s.CompanyLegalType, s.CompanyEIK,
		s.CompanyAddressStreet, s.CompanyAddressCity, s.CompanyAddressPostalCode, s.CompanyAddressCountry,
		s.CompanyEmail, s.CompanyPhone, s.NRAStoreNumber, s.VATNumber, s.VATRate,
	)
	return err
}

// ── Couriers ─────────────────────────────────────────────────────────────────

func (r *PostgresRepository) ListCouriers(ctx context.Context) ([]domain.Courier, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, identifier, is_active, sort_order, created_at
		FROM invoice_couriers ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var couriers []domain.Courier
	for rows.Next() {
		var c domain.Courier
		if err := rows.Scan(&c.ID, &c.Name, &c.Identifier, &c.IsActive, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, err
		}
		couriers = append(couriers, c)
	}
	return couriers, rows.Err()
}

func (r *PostgresRepository) CreateCourier(ctx context.Context, c domain.Courier) (*domain.Courier, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO invoice_couriers (id, name, identifier, is_active, sort_order)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, name, identifier, is_active, sort_order, created_at`,
		c.ID, c.Name, c.Identifier, c.IsActive, c.SortOrder,
	).Scan(&c.ID, &c.Name, &c.Identifier, &c.IsActive, &c.SortOrder, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) UpdateCourier(ctx context.Context, id uuid.UUID, c domain.Courier) (*domain.Courier, error) {
	err := r.db.QueryRow(ctx, `
		UPDATE invoice_couriers SET name=$2, identifier=$3, is_active=$4, sort_order=$5
		WHERE id=$1
		RETURNING id, name, identifier, is_active, sort_order, created_at`,
		id, c.Name, c.Identifier, c.IsActive, c.SortOrder,
	).Scan(&c.ID, &c.Name, &c.Identifier, &c.IsActive, &c.SortOrder, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrInvoiceNotFound
	}
	return &c, err
}

func (r *PostgresRepository) DeleteCourier(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM invoice_couriers WHERE id = $1`, id)
	return err
}

// ── Tax groups ───────────────────────────────────────────────────────────────

func (r *PostgresRepository) ListTaxGroups(ctx context.Context) ([]domain.TaxGroup, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, identifier, vat_rate, created_at, updated_at
		FROM tax_groups ORDER BY identifier`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []domain.TaxGroup
	for rows.Next() {
		var g domain.TaxGroup
		if err := rows.Scan(&g.ID, &g.Identifier, &g.VATRate, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (r *PostgresRepository) CreateTaxGroup(ctx context.Context, g domain.TaxGroup) (*domain.TaxGroup, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO tax_groups (id, identifier, vat_rate)
		VALUES ($1,$2,$3)
		RETURNING id, identifier, vat_rate, created_at, updated_at`,
		g.ID, g.Identifier, g.VATRate,
	).Scan(&g.ID, &g.Identifier, &g.VATRate, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *PostgresRepository) UpdateTaxGroup(ctx context.Context, id uuid.UUID, g domain.TaxGroup) (*domain.TaxGroup, error) {
	err := r.db.QueryRow(ctx, `
		UPDATE tax_groups SET identifier=$2, vat_rate=$3, updated_at=NOW()
		WHERE id=$1
		RETURNING id, identifier, vat_rate, created_at, updated_at`,
		id, g.Identifier, g.VATRate,
	).Scan(&g.ID, &g.Identifier, &g.VATRate, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrInvoiceNotFound
	}
	return &g, err
}

func (r *PostgresRepository) DeleteTaxGroup(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM tax_groups WHERE id = $1`, id)
	return err
}

// ── Audit log ────────────────────────────────────────────────────────────────

func (r *PostgresRepository) LogAuditEvent(ctx context.Context, invoiceNumber, eventType, actor string, metadata map[string]any) error {
	var metaJSON []byte
	if len(metadata) > 0 {
		var err error
		metaJSON, err = json.Marshal(metadata)
		if err != nil {
			return err
		}
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO invoice_audit_log (invoice_number, event_type, actor, metadata) VALUES ($1,$2,$3,$4)`,
		invoiceNumber, eventType, actor, metaJSON,
	)
	return err
}

// ── Scanners ─────────────────────────────────────────────────────────────────

func scanInvoice(row pgx.Row) (*domain.Invoice, error) {
	var inv domain.Invoice
	var docType string
	var discountMinor *int64
	var vatRate float64

	err := row.Scan(
		&inv.ID, &inv.InvoiceNumber, &docType, &inv.OrderID, &inv.StornoOfInvoiceID,
		&inv.OrderNumber, &inv.PlacedAt, &inv.PaymentMethod,
		&inv.CardProvider, &inv.CardProviderReference, &inv.CourierName, &inv.CourierIdentifier,
		&inv.CompanyName, &inv.CompanyLegalType, &inv.CompanyEIK, &inv.CompanyAddress, &inv.CompanyEmail, &inv.CompanyPhone,
		&inv.NRAStoreNumber, &inv.VATNumber, &vatRate,
		&inv.RecipientName, &inv.RecipientAddress, &inv.RecipientEmail,
		&inv.SubtotalExclVAT.AmountMinor, &inv.VATAmount.AmountMinor, &inv.TotalInclVAT.AmountMinor, &inv.TotalInclVAT.Currency,
		&inv.DeliveryFee.AmountMinor, &discountMinor,
		&inv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	inv.DocumentType = domain.DocumentType(docType)
	inv.VATRate = vatRate
	inv.SubtotalExclVAT.Currency = inv.TotalInclVAT.Currency
	inv.VATAmount.Currency = inv.TotalInclVAT.Currency
	inv.DeliveryFee.Currency = inv.TotalInclVAT.Currency
	if discountMinor != nil {
		m := money.Money{AmountMinor: *discountMinor, Currency: inv.TotalInclVAT.Currency}
		inv.DiscountAmount = &m
	}
	return &inv, nil
}

func scanLineItem(row pgx.Row, currency string) (*domain.InvoiceLineItem, error) {
	var item domain.InvoiceLineItem
	var createdAt interface{}
	err := row.Scan(
		&item.ID, &item.InvoiceID,
		&item.ProductName, &item.VariantLabel, &item.Quantity,
		&item.UnitPriceInclVAT.AmountMinor, &item.UnitPriceExclVAT.AmountMinor, &item.VATPerUnit.AmountMinor,
		&item.LineTotalInclVAT.AmountMinor, &item.LineTotalExclVAT.AmountMinor, &item.LineVATAmount.AmountMinor,
		&item.VATRate, &item.SortOrder, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	item.UnitPriceInclVAT.Currency = currency
	item.UnitPriceExclVAT.Currency = currency
	item.VATPerUnit.Currency = currency
	item.LineTotalInclVAT.Currency = currency
	item.LineTotalExclVAT.Currency = currency
	item.LineVATAmount.Currency = currency
	return &item, nil
}
