-- +goose Up

-- Seed starter content for the new About and Size Guide pages, which reuse the
-- translatable store_documents (Markdown) system. ON CONFLICT DO NOTHING so an
-- admin's later edits are never overwritten on re-run.

-- +goose StatementBegin
INSERT INTO store_documents (type, locale, content_md) VALUES
('about', 'en', '# About Verani

Verani is a Bulgarian boutique for considered fashion — clothing, jewelry, bags, and accessories chosen with care and made to last.

We believe getting dressed should feel effortless and personal. That is why we curate a tight, thoughtful edit rather than an endless catalog: pieces that work together, wear beautifully, and earn their place in your wardrobe season after season.

From the materials we source to the way your order arrives at your door, we sweat the details so you do not have to. Thank you for being here — we are glad you found us.'),
('about', 'bg', '# За Verani

Verani е български бутик за обмислена мода — дрехи, бижута, чанти и аксесоари, подбрани с внимание и създадени да издържат във времето.

Вярваме, че обличането трябва да е леко и лично. Затова подбираме стегната и премислена колекция вместо безкраен каталог: артикули, които се съчетават помежду си, носят се красиво и остават любими сезон след сезон.

От материалите, които избираме, до начина, по който поръчката ви пристига у дома — грижим се за детайлите, за да не се налага вие да го правите. Благодарим ви, че сте тук.')
ON CONFLICT (type, locale) DO NOTHING;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO store_documents (type, locale, content_md) VALUES
('size_guide', 'en', '# Size Guide

Use the charts below as a general guide. Measurements are in centimeters. If you are between two sizes, we recommend sizing up.

## Women — Clothing

| Size | EU | Bust (cm) | Waist (cm) | Hips (cm) |
| --- | --- | --- | --- | --- |
| XS | 34 | 80–83 | 62–65 | 88–91 |
| S | 36 | 84–87 | 66–69 | 92–95 |
| M | 38 | 88–92 | 70–74 | 96–100 |
| L | 40 | 93–97 | 75–80 | 101–105 |
| XL | 42 | 98–103 | 81–86 | 106–111 |

## Men — Clothing

| Size | Chest (cm) | Waist (cm) |
| --- | --- | --- |
| S | 88–94 | 74–80 |
| M | 95–101 | 81–87 |
| L | 102–108 | 88–94 |
| XL | 109–115 | 95–101 |

## How to measure

- **Bust / Chest:** measure around the fullest part, keeping the tape level.
- **Waist:** measure around the narrowest part of your waistline.
- **Hips:** measure around the fullest part of your hips.

Still unsure? Contact us and we will help you find the right fit.'),
('size_guide', 'bg', '# Таблица с размери

Използвайте таблиците по-долу като общ ориентир. Мерките са в сантиметри. Ако сте между два размера, препоръчваме да изберете по-големия.

## Жени — Облекло

| Размер | EU | Бюст (см) | Талия (см) | Ханш (см) |
| --- | --- | --- | --- | --- |
| XS | 34 | 80–83 | 62–65 | 88–91 |
| S | 36 | 84–87 | 66–69 | 92–95 |
| M | 38 | 88–92 | 70–74 | 96–100 |
| L | 40 | 93–97 | 75–80 | 101–105 |
| XL | 42 | 98–103 | 81–86 | 106–111 |

## Мъже — Облекло

| Размер | Гръдна обиколка (см) | Талия (см) |
| --- | --- | --- |
| S | 88–94 | 74–80 |
| M | 95–101 | 81–87 |
| L | 102–108 | 88–94 |
| XL | 109–115 | 95–101 |

## Как да измерите

- **Бюст / Гръдна обиколка:** измерете около най-широката част, като държите сантиметъра хоризонтално.
- **Талия:** измерете около най-тясната част на талията.
- **Ханш:** измерете около най-широката част на ханша.

Все още се колебаете? Свържете се с нас и ще ви помогнем да намерите правилния размер.')
ON CONFLICT (type, locale) DO NOTHING;
-- +goose StatementEnd

-- Bulgarian translations for the trust bar (previously English-only) and the
-- new About / Contact / Size Guide page chrome. English baselines for these
-- keys live in defaultUIStrings (cmd/api/modules.go); this fills the bg locale.
INSERT INTO ui_strings (key, locale, value) VALUES
('trust.shipping_title',     'bg', 'Безплатна доставка'),
('trust.shipping_subtitle',  'bg', 'При поръчки над 100 лв.'),
('trust.returns_title',      'bg', 'Лесно връщане'),
('trust.returns_subtitle',   'bg', '30 дни за размисъл'),
('trust.secure_title',       'bg', 'Сигурно плащане'),
('trust.secure_subtitle',    'bg', 'Криптирани трансакции'),
('trust.authentic_title',    'bg', 'Автентични продукти'),
('trust.authentic_subtitle', 'bg', 'Гарантирано качество'),
('about.contact_cta',        'bg', 'Свържете се с нас'),
('about.unavailable',        'bg', 'Скоро ще споделим нашата история.'),
('contact.title',            'bg', 'Свържете се с нас'),
('contact.intro',            'bg', 'Ще се радваме да се свържете с нас. Заповядайте в магазина или ни пишете по всяко време.'),
('contact.details_heading',  'bg', 'Контакти'),
('contact.hours_heading',    'bg', 'Работно време'),
('contact.find_us_heading',  'bg', 'Как да ни намерите'),
('contact.email_label',      'bg', 'Имейл'),
('contact.phone_label',      'bg', 'Телефон'),
('contact.address_label',    'bg', 'Адрес'),
('contact.unavailable',      'bg', 'Информацията за контакт ще бъде добавена скоро.'),
('sizeguide.title',          'bg', 'Таблица с размери'),
('sizeguide.unavailable',    'bg', 'Таблицата с размери ще бъде добавена скоро.')
ON CONFLICT (key, locale) DO NOTHING;

-- +goose Down

DELETE FROM store_documents WHERE type IN ('about', 'size_guide');
DELETE FROM ui_strings WHERE locale = 'bg' AND (
    key LIKE 'trust.%' OR key LIKE 'about.%' OR key LIKE 'contact.%' OR key LIKE 'sizeguide.%'
);
