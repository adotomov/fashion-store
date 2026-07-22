-- +goose Up
-- email_templates holds the per-locale copy for each transactional email, keyed
-- the same way as the i18n ui_strings table. Stored in the database rather than
-- embedded in the binary so wording can be corrected without a deploy — the same
-- reasoning behind admin-editable UI strings and legal documents.
--
-- html_body/text_body are Go html/template (resp. text/template) fragments
-- holding only the INNER content; the renderer wraps them in a shared branded
-- layout and injects store branding (StoreName, LogoURL, StorefrontURL, Year).
-- Monetary and date values arrive already formatted by the producer, so
-- templates never have to do locale-aware formatting.
CREATE TABLE email_templates (
	template_key TEXT NOT NULL,
	locale TEXT NOT NULL,
	subject TEXT NOT NULL,
	html_body TEXT NOT NULL,
	text_body TEXT NOT NULL DEFAULT '',
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (template_key, locale)
);

-- +goose StatementBegin
INSERT INTO email_templates (template_key, locale, subject, html_body, text_body) VALUES

('welcome', 'en', $sub$Welcome to {{.StoreName}}$sub$, $html$
<h1>Welcome, {{.CustomerName}}</h1>
<p>Thanks for creating an account at {{.StoreName}}. Your account is ready.</p>
<p>From your account you can track orders, save favourites and check out faster.</p>
<p><a class="btn" href="{{.StorefrontURL}}/shop">Start shopping</a></p>
$html$, $text$Welcome, {{.CustomerName}}

Thanks for creating an account at {{.StoreName}}. Your account is ready.
From your account you can track orders, save favourites and check out faster.

Start shopping: {{.StorefrontURL}}/shop
$text$),

('welcome', 'bg', $sub$Добре дошли в {{.StoreName}}$sub$, $html$
<h1>Добре дошли, {{.CustomerName}}</h1>
<p>Благодарим ви, че създадохте профил в {{.StoreName}}. Профилът ви е готов.</p>
<p>От вашия профил можете да проследявате поръчки, да запазвате любими продукти и да поръчвате по-бързо.</p>
<p><a class="btn" href="{{.StorefrontURL}}/shop">Към магазина</a></p>
$html$, $text$Добре дошли, {{.CustomerName}}

Благодарим ви, че създадохте профил в {{.StoreName}}. Профилът ви е готов.

Към магазина: {{.StorefrontURL}}/shop
$text$),

('order_confirmation', 'en', $sub${{.StoreName}} — order {{.OrderNumber}} confirmed$sub$, $html$
<h1>Thank you for your order</h1>
<p>Hi {{.CustomerName}}, we have received order <strong>{{.OrderNumber}}</strong> and it is now being prepared.</p>
<table class="items">
	<tr><th align="left">Item</th><th align="right">Qty</th><th align="right">Total</th></tr>
	{{range .Items}}
	<tr>
		<td align="left">{{.Name}}{{if .Variant}} — {{.Variant}}{{end}}</td>
		<td align="right">{{.Quantity}}</td>
		<td align="right">{{.LineTotal}}</td>
	</tr>
	{{end}}
	<tr><td align="left">Delivery</td><td></td><td align="right">{{.DeliveryFee}}</td></tr>
	<tr class="total"><td align="left"><strong>Total</strong></td><td></td><td align="right"><strong>{{.OrderTotal}}</strong></td></tr>
</table>
<p><strong>Delivery method:</strong> {{.DeliveryMethod}}<br>
<strong>Payment:</strong> {{.PaymentMethod}}</p>
<p>{{.ShippingAddress}}</p>
$html$, $text$Thank you for your order

Hi {{.CustomerName}}, we have received order {{.OrderNumber}} and it is now being prepared.

{{range .Items}}- {{.Name}}{{if .Variant}} ({{.Variant}}){{end}} x{{.Quantity}}  {{.LineTotal}}
{{end}}Delivery: {{.DeliveryFee}}
Total: {{.OrderTotal}}

Delivery method: {{.DeliveryMethod}}
Payment: {{.PaymentMethod}}
{{.ShippingAddress}}
$text$),

('order_confirmation', 'bg', $sub${{.StoreName}} — поръчка {{.OrderNumber}} е потвърдена$sub$, $html$
<h1>Благодарим за поръчката</h1>
<p>Здравейте, {{.CustomerName}}. Получихме поръчка <strong>{{.OrderNumber}}</strong> и вече я подготвяме.</p>
<table class="items">
	<tr><th align="left">Продукт</th><th align="right">Бр.</th><th align="right">Сума</th></tr>
	{{range .Items}}
	<tr>
		<td align="left">{{.Name}}{{if .Variant}} — {{.Variant}}{{end}}</td>
		<td align="right">{{.Quantity}}</td>
		<td align="right">{{.LineTotal}}</td>
	</tr>
	{{end}}
	<tr><td align="left">Доставка</td><td></td><td align="right">{{.DeliveryFee}}</td></tr>
	<tr class="total"><td align="left"><strong>Общо</strong></td><td></td><td align="right"><strong>{{.OrderTotal}}</strong></td></tr>
</table>
<p><strong>Начин на доставка:</strong> {{.DeliveryMethod}}<br>
<strong>Плащане:</strong> {{.PaymentMethod}}</p>
<p>{{.ShippingAddress}}</p>
$html$, $text$Благодарим за поръчката

Здравейте, {{.CustomerName}}. Получихме поръчка {{.OrderNumber}} и вече я подготвяме.

{{range .Items}}- {{.Name}}{{if .Variant}} ({{.Variant}}){{end}} x{{.Quantity}}  {{.LineTotal}}
{{end}}Доставка: {{.DeliveryFee}}
Общо: {{.OrderTotal}}

Начин на доставка: {{.DeliveryMethod}}
Плащане: {{.PaymentMethod}}
{{.ShippingAddress}}
$text$),

('shipping_update', 'en', $sub$Your order {{.OrderNumber}} is on its way$sub$, $html$
<h1>Your order has shipped</h1>
<p>Hi {{.CustomerName}}, order <strong>{{.OrderNumber}}</strong> has been handed to {{.Carrier}}.</p>
{{if .TrackingNumber}}<p><strong>Tracking number:</strong> {{.TrackingNumber}}</p>{{end}}
{{if .TrackingURL}}<p><a class="btn" href="{{.TrackingURL}}">Track your parcel</a></p>{{end}}
$html$, $text$Your order has shipped

Hi {{.CustomerName}}, order {{.OrderNumber}} has been handed to {{.Carrier}}.
{{if .TrackingNumber}}Tracking number: {{.TrackingNumber}}
{{end}}{{if .TrackingURL}}Track your parcel: {{.TrackingURL}}
{{end}}$text$),

('shipping_update', 'bg', $sub$Поръчка {{.OrderNumber}} е изпратена$sub$, $html$
<h1>Поръчката ви е изпратена</h1>
<p>Здравейте, {{.CustomerName}}. Поръчка <strong>{{.OrderNumber}}</strong> беше предадена на {{.Carrier}}.</p>
{{if .TrackingNumber}}<p><strong>Товарителница:</strong> {{.TrackingNumber}}</p>{{end}}
{{if .TrackingURL}}<p><a class="btn" href="{{.TrackingURL}}">Проследи пратката</a></p>{{end}}
$html$, $text$Поръчката ви е изпратена

Здравейте, {{.CustomerName}}. Поръчка {{.OrderNumber}} беше предадена на {{.Carrier}}.
{{if .TrackingNumber}}Товарителница: {{.TrackingNumber}}
{{end}}{{if .TrackingURL}}Проследи пратката: {{.TrackingURL}}
{{end}}$text$),

('payment_failed', 'en', $sub$Payment could not be completed for order {{.OrderNumber}}$sub$, $html$
<h1>We could not take payment</h1>
<p>Hi {{.CustomerName}}, the card payment for order <strong>{{.OrderNumber}}</strong> ({{.OrderTotal}}) did not go through, so the order has not been placed.</p>
<p>No money has been taken. Your basket is still saved — you can try again with another card or choose to pay on delivery.</p>
<p><a class="btn" href="{{.StorefrontURL}}/cart">Return to your basket</a></p>
$html$, $text$We could not take payment

Hi {{.CustomerName}}, the card payment for order {{.OrderNumber}} ({{.OrderTotal}}) did not go through, so the order has not been placed.
No money has been taken. Your basket is still saved.

Return to your basket: {{.StorefrontURL}}/cart
$text$),

('payment_failed', 'bg', $sub$Неуспешно плащане за поръчка {{.OrderNumber}}$sub$, $html$
<h1>Плащането не беше успешно</h1>
<p>Здравейте, {{.CustomerName}}. Плащането с карта за поръчка <strong>{{.OrderNumber}}</strong> ({{.OrderTotal}}) не беше успешно и поръчката не е направена.</p>
<p>Не са удържани средства. Количката ви е запазена — можете да опитате с друга карта или да изберете плащане при доставка.</p>
<p><a class="btn" href="{{.StorefrontURL}}/cart">Към количката</a></p>
$html$, $text$Плащането не беше успешно

Здравейте, {{.CustomerName}}. Плащането с карта за поръчка {{.OrderNumber}} ({{.OrderTotal}}) не беше успешно и поръчката не е направена.
Не са удържани средства. Количката ви е запазена.

Към количката: {{.StorefrontURL}}/cart
$text$);
-- +goose StatementEnd

-- +goose Down
DROP TABLE email_templates;
