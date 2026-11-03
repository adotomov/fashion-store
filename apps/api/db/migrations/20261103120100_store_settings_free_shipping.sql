-- +goose Up

-- The free-shipping threshold, in EUR minor units: an order at or above it pays no delivery fee. 5100 is €51,
-- the euro equivalent of the 100 лв the store promised before the euro, at the fixed rate.
ALTER TABLE store_settings ADD COLUMN free_shipping_threshold_minor BIGINT NOT NULL DEFAULT 5100
	CHECK (free_shipping_threshold_minor >= 0);

-- The shipping copy shows the threshold. The storefront fills in {amount} from the setting above.
UPDATE ui_strings SET value = 'При поръчки над {amount}' WHERE key = 'trust.shipping_subtitle' AND locale = 'bg';
UPDATE ui_strings SET value = 'On orders over {amount}' WHERE key = 'trust.shipping_subtitle' AND locale = 'en';
UPDATE ui_strings SET value = 'Free shipping over {amount}' WHERE key = 'footer.free_shipping' AND locale = 'en';
UPDATE ui_strings SET value = 'Безплатна доставка над {amount}' WHERE key = 'footer.free_shipping' AND locale = 'bg';

-- +goose Down

UPDATE ui_strings SET value = 'При поръчки над 100 лв.' WHERE key = 'trust.shipping_subtitle' AND locale = 'bg';
UPDATE ui_strings SET value = 'On orders over $100' WHERE key = 'trust.shipping_subtitle' AND locale = 'en';
UPDATE ui_strings SET value = 'Free shipping over $100' WHERE key = 'footer.free_shipping' AND locale = 'en';
UPDATE ui_strings SET value = 'Безплатна доставка над 100 лв.' WHERE key = 'footer.free_shipping' AND locale = 'bg';

ALTER TABLE store_settings DROP COLUMN free_shipping_threshold_minor;
