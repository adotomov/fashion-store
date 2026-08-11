-- +goose Up
-- Restyle the transactional email bodies to the editorial boutique look shipped
-- in renderer.go's emailLayout (serif wordmark/headings, warm ivory canvas,
-- refined items/totals, soft panels, squared uppercase CTAs). Copy-and-markup
-- only: template keys, locales and the producer variable contract are unchanged,
-- so no producer or renderer change is required.
--
-- Structural styling that must survive clients which strip <style> (headings,
-- paragraphs, buttons, panels, table alignment) is inlined here; the layout's
-- <style> block is a progressive enhancement on top.
--
-- Done as a single INSERT ... ON CONFLICT upsert (one statement) so goose treats
-- the dollar-quoted HTML as one unit, mirroring the original seed migration.

-- +goose StatementBegin
INSERT INTO email_templates (template_key, locale, subject, html_body, text_body) VALUES

('welcome', 'en', $sub$Welcome to {{.StoreName}}$sub$, $html$
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.18em;text-transform:uppercase;color:#a8a29e;margin:0 0 14px;">A warm welcome</p>
<h1 style="font-family:Georgia,'Times New Roman',serif;font-weight:400;font-size:24px;line-height:1.28;color:#1c1917;margin:0 0 18px;">Welcome, {{.CustomerName}}</h1>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 16px;">Thank you for creating an account at {{.StoreName}}. Everything is ready — from here you can track your orders, save the pieces you love, and check out faster next time.</p>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 22px;">We're glad to have you with us.</p>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:2px 0;"><tr><td align="center" bgcolor="#1c1917" style="background:#1c1917;border-radius:2px;"><a href="{{.StorefrontURL}}/shop" style="display:inline-block;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:13px;font-weight:600;letter-spacing:.12em;text-transform:uppercase;color:#ffffff;text-decoration:none;padding:14px 30px;">Start shopping</a></td></tr></table>
$html$, $text$Welcome, {{.CustomerName}}

Thank you for creating an account at {{.StoreName}}. Everything is ready — from here you can track your orders, save the pieces you love, and check out faster next time.

We're glad to have you with us.

Start shopping: {{.StorefrontURL}}/shop
$text$),

('welcome', 'bg', $sub$Добре дошли в {{.StoreName}}$sub$, $html$
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.18em;text-transform:uppercase;color:#a8a29e;margin:0 0 14px;">Добре дошли</p>
<h1 style="font-family:Georgia,'Times New Roman',serif;font-weight:400;font-size:24px;line-height:1.28;color:#1c1917;margin:0 0 18px;">Добре дошли, {{.CustomerName}}</h1>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 16px;">Благодарим ви, че създадохте профил в {{.StoreName}}. Всичко е готово — оттук можете да проследявате поръчките си, да запазвате любимите си артикули и да поръчвате по-бързо.</p>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 22px;">Радваме се, че сте с нас.</p>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:2px 0;"><tr><td align="center" bgcolor="#1c1917" style="background:#1c1917;border-radius:2px;"><a href="{{.StorefrontURL}}/shop" style="display:inline-block;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:13px;font-weight:600;letter-spacing:.12em;text-transform:uppercase;color:#ffffff;text-decoration:none;padding:14px 30px;">Към магазина</a></td></tr></table>
$html$, $text$Добре дошли, {{.CustomerName}}

Благодарим ви, че създадохте профил в {{.StoreName}}. Всичко е готово — оттук можете да проследявате поръчките си, да запазвате любимите си артикули и да поръчвате по-бързо.

Радваме се, че сте с нас.

Към магазина: {{.StorefrontURL}}/shop
$text$),

('order_confirmation', 'en', $sub${{.StoreName}} — order {{.OrderNumber}} confirmed$sub$, $html$
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.18em;text-transform:uppercase;color:#a8a29e;margin:0 0 14px;">Order confirmed</p>
<h1 style="font-family:Georgia,'Times New Roman',serif;font-weight:400;font-size:24px;line-height:1.28;color:#1c1917;margin:0 0 18px;">Thank you for your order</h1>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 20px;">Hi {{.CustomerName}}, we've received order <strong style="color:#1c1917;">{{.OrderNumber}}</strong> and we're preparing it now.</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="border-collapse:collapse;margin:0 0 6px;">
  <tr>
    <th align="left" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;text-align:left;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;font-weight:600;padding:0 0 10px;border-bottom:1px solid #ece7e0;">Item</th>
    <th align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;text-align:right;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;font-weight:600;padding:0 0 10px;border-bottom:1px solid #ece7e0;">Qty</th>
    <th align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;text-align:right;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;font-weight:600;padding:0 0 10px;border-bottom:1px solid #ece7e0;">Total</th>
  </tr>
  {{range .Items}}
  <tr>
    <td style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;padding:12px 0;border-bottom:1px solid #f4f1ec;vertical-align:top;">{{.Name}}{{if .Variant}}<span style="color:#a8a29e;font-size:13px;"> — {{.Variant}}</span>{{end}}</td>
    <td align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;padding:12px 0;border-bottom:1px solid #f4f1ec;text-align:right;white-space:nowrap;vertical-align:top;">{{.Quantity}}</td>
    <td align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;padding:12px 0;border-bottom:1px solid #f4f1ec;text-align:right;white-space:nowrap;vertical-align:top;">{{.LineTotal}}</td>
  </tr>
  {{end}}
  <tr>
    <td style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:14px;color:#78716c;padding:6px 0;">Delivery</td>
    <td></td>
    <td align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:14px;color:#78716c;padding:6px 0;text-align:right;white-space:nowrap;">{{.DeliveryFee}}</td>
  </tr>
  <tr>
    <td style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:17px;font-weight:700;color:#1c1917;padding:14px 0 0;border-top:2px solid #1c1917;">Total</td>
    <td style="border-top:2px solid #1c1917;"></td>
    <td align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:17px;font-weight:700;color:#1c1917;padding:14px 0 0;border-top:2px solid #1c1917;text-align:right;white-space:nowrap;">{{.OrderTotal}}</td>
  </tr>
</table>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:22px 0 2px;background:#faf8f5;border-radius:6px;"><tr><td style="padding:20px 22px;">
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;margin:0 0 3px;">Delivery method</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;line-height:1.5;margin:0 0 14px;">{{.DeliveryMethod}}</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;margin:0 0 3px;">Payment</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;line-height:1.5;margin:0 0 14px;">{{.PaymentMethod}}</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;margin:0 0 3px;">Shipping to</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;line-height:1.5;margin:0;">{{.ShippingAddress}}</p>
</td></tr></table>
$html$, $text$Thank you for your order

Hi {{.CustomerName}}, we've received order {{.OrderNumber}} and we're preparing it now.

{{range .Items}}- {{.Name}}{{if .Variant}} ({{.Variant}}){{end}} x{{.Quantity}}  {{.LineTotal}}
{{end}}Delivery: {{.DeliveryFee}}
Total: {{.OrderTotal}}

Delivery method: {{.DeliveryMethod}}
Payment: {{.PaymentMethod}}
Shipping to: {{.ShippingAddress}}
$text$),

('order_confirmation', 'bg', $sub${{.StoreName}} — поръчка {{.OrderNumber}} е потвърдена$sub$, $html$
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.18em;text-transform:uppercase;color:#a8a29e;margin:0 0 14px;">Поръчката е потвърдена</p>
<h1 style="font-family:Georgia,'Times New Roman',serif;font-weight:400;font-size:24px;line-height:1.28;color:#1c1917;margin:0 0 18px;">Благодарим за поръчката</h1>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 20px;">Здравейте, {{.CustomerName}}. Получихме поръчка <strong style="color:#1c1917;">{{.OrderNumber}}</strong> и вече я подготвяме.</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="border-collapse:collapse;margin:0 0 6px;">
  <tr>
    <th align="left" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;text-align:left;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;font-weight:600;padding:0 0 10px;border-bottom:1px solid #ece7e0;">Продукт</th>
    <th align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;text-align:right;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;font-weight:600;padding:0 0 10px;border-bottom:1px solid #ece7e0;">Бр.</th>
    <th align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;text-align:right;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;font-weight:600;padding:0 0 10px;border-bottom:1px solid #ece7e0;">Сума</th>
  </tr>
  {{range .Items}}
  <tr>
    <td style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;padding:12px 0;border-bottom:1px solid #f4f1ec;vertical-align:top;">{{.Name}}{{if .Variant}}<span style="color:#a8a29e;font-size:13px;"> — {{.Variant}}</span>{{end}}</td>
    <td align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;padding:12px 0;border-bottom:1px solid #f4f1ec;text-align:right;white-space:nowrap;vertical-align:top;">{{.Quantity}}</td>
    <td align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;padding:12px 0;border-bottom:1px solid #f4f1ec;text-align:right;white-space:nowrap;vertical-align:top;">{{.LineTotal}}</td>
  </tr>
  {{end}}
  <tr>
    <td style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:14px;color:#78716c;padding:6px 0;">Доставка</td>
    <td></td>
    <td align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:14px;color:#78716c;padding:6px 0;text-align:right;white-space:nowrap;">{{.DeliveryFee}}</td>
  </tr>
  <tr>
    <td style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:17px;font-weight:700;color:#1c1917;padding:14px 0 0;border-top:2px solid #1c1917;">Общо</td>
    <td style="border-top:2px solid #1c1917;"></td>
    <td align="right" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:17px;font-weight:700;color:#1c1917;padding:14px 0 0;border-top:2px solid #1c1917;text-align:right;white-space:nowrap;">{{.OrderTotal}}</td>
  </tr>
</table>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:22px 0 2px;background:#faf8f5;border-radius:6px;"><tr><td style="padding:20px 22px;">
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;margin:0 0 3px;">Начин на доставка</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;line-height:1.5;margin:0 0 14px;">{{.DeliveryMethod}}</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;margin:0 0 3px;">Плащане</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;line-height:1.5;margin:0 0 14px;">{{.PaymentMethod}}</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;margin:0 0 3px;">Доставка до</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;color:#292524;line-height:1.5;margin:0;">{{.ShippingAddress}}</p>
</td></tr></table>
$html$, $text$Благодарим за поръчката

Здравейте, {{.CustomerName}}. Получихме поръчка {{.OrderNumber}} и вече я подготвяме.

{{range .Items}}- {{.Name}}{{if .Variant}} ({{.Variant}}){{end}} x{{.Quantity}}  {{.LineTotal}}
{{end}}Доставка: {{.DeliveryFee}}
Общо: {{.OrderTotal}}

Начин на доставка: {{.DeliveryMethod}}
Плащане: {{.PaymentMethod}}
Доставка до: {{.ShippingAddress}}
$text$),

('shipping_update', 'en', $sub$Your order {{.OrderNumber}} is on its way$sub$, $html$
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.18em;text-transform:uppercase;color:#a8a29e;margin:0 0 14px;">On its way</p>
<h1 style="font-family:Georgia,'Times New Roman',serif;font-weight:400;font-size:24px;line-height:1.28;color:#1c1917;margin:0 0 18px;">Your order has shipped</h1>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 20px;">Hi {{.CustomerName}}, order <strong style="color:#1c1917;">{{.OrderNumber}}</strong> has been handed to {{.Carrier}} and is on its way to you.</p>
{{if .TrackingNumber}}<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 22px;background:#faf8f5;border-radius:6px;"><tr><td style="padding:18px 22px;">
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;margin:0 0 3px;">Tracking number</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:16px;color:#1c1917;letter-spacing:.02em;margin:0;">{{.TrackingNumber}}</p>
</td></tr></table>{{end}}
{{if .TrackingURL}}<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:2px 0;"><tr><td align="center" bgcolor="#1c1917" style="background:#1c1917;border-radius:2px;"><a href="{{.TrackingURL}}" style="display:inline-block;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:13px;font-weight:600;letter-spacing:.12em;text-transform:uppercase;color:#ffffff;text-decoration:none;padding:14px 30px;">Track your parcel</a></td></tr></table>{{end}}
$html$, $text$Your order has shipped

Hi {{.CustomerName}}, order {{.OrderNumber}} has been handed to {{.Carrier}} and is on its way to you.
{{if .TrackingNumber}}Tracking number: {{.TrackingNumber}}
{{end}}{{if .TrackingURL}}Track your parcel: {{.TrackingURL}}
{{end}}$text$),

('shipping_update', 'bg', $sub$Поръчка {{.OrderNumber}} е изпратена$sub$, $html$
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.18em;text-transform:uppercase;color:#a8a29e;margin:0 0 14px;">На път към вас</p>
<h1 style="font-family:Georgia,'Times New Roman',serif;font-weight:400;font-size:24px;line-height:1.28;color:#1c1917;margin:0 0 18px;">Поръчката ви е изпратена</h1>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 20px;">Здравейте, {{.CustomerName}}. Поръчка <strong style="color:#1c1917;">{{.OrderNumber}}</strong> беше предадена на {{.Carrier}} и вече пътува към вас.</p>
{{if .TrackingNumber}}<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 22px;background:#faf8f5;border-radius:6px;"><tr><td style="padding:18px 22px;">
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:#a8a29e;margin:0 0 3px;">Товарителница</p>
  <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:16px;color:#1c1917;letter-spacing:.02em;margin:0;">{{.TrackingNumber}}</p>
</td></tr></table>{{end}}
{{if .TrackingURL}}<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:2px 0;"><tr><td align="center" bgcolor="#1c1917" style="background:#1c1917;border-radius:2px;"><a href="{{.TrackingURL}}" style="display:inline-block;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:13px;font-weight:600;letter-spacing:.12em;text-transform:uppercase;color:#ffffff;text-decoration:none;padding:14px 30px;">Проследи пратката</a></td></tr></table>{{end}}
$html$, $text$Поръчката ви е изпратена

Здравейте, {{.CustomerName}}. Поръчка {{.OrderNumber}} беше предадена на {{.Carrier}} и вече пътува към вас.
{{if .TrackingNumber}}Товарителница: {{.TrackingNumber}}
{{end}}{{if .TrackingURL}}Проследи пратката: {{.TrackingURL}}
{{end}}$text$),

('payment_failed', 'en', $sub$Payment could not be completed for order {{.OrderNumber}}$sub$, $html$
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.18em;text-transform:uppercase;color:#a8a29e;margin:0 0 14px;">Payment unsuccessful</p>
<h1 style="font-family:Georgia,'Times New Roman',serif;font-weight:400;font-size:24px;line-height:1.28;color:#1c1917;margin:0 0 18px;">We couldn't complete your payment</h1>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 16px;">Hi {{.CustomerName}}, the card payment for order <strong style="color:#1c1917;">{{.OrderNumber}}</strong> ({{.OrderTotal}}) didn't go through, so the order hasn't been placed.</p>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 22px;">No money has been taken and your basket is still saved — you can try again with another card, or choose to pay on delivery.</p>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:2px 0;"><tr><td align="center" bgcolor="#1c1917" style="background:#1c1917;border-radius:2px;"><a href="{{.StorefrontURL}}/cart" style="display:inline-block;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:13px;font-weight:600;letter-spacing:.12em;text-transform:uppercase;color:#ffffff;text-decoration:none;padding:14px 30px;">Return to your basket</a></td></tr></table>
$html$, $text$We couldn't complete your payment

Hi {{.CustomerName}}, the card payment for order {{.OrderNumber}} ({{.OrderTotal}}) didn't go through, so the order hasn't been placed.
No money has been taken and your basket is still saved — you can try again with another card, or choose to pay on delivery.

Return to your basket: {{.StorefrontURL}}/cart
$text$),

('payment_failed', 'bg', $sub$Неуспешно плащане за поръчка {{.OrderNumber}}$sub$, $html$
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:11px;letter-spacing:.18em;text-transform:uppercase;color:#a8a29e;margin:0 0 14px;">Неуспешно плащане</p>
<h1 style="font-family:Georgia,'Times New Roman',serif;font-weight:400;font-size:24px;line-height:1.28;color:#1c1917;margin:0 0 18px;">Плащането не беше успешно</h1>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 16px;">Здравейте, {{.CustomerName}}. Плащането с карта за поръчка <strong style="color:#1c1917;">{{.OrderNumber}}</strong> ({{.OrderTotal}}) не беше успешно и поръчката не е направена.</p>
<p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#44403c;margin:0 0 22px;">Не са удържани средства, а количката ви е запазена — можете да опитате с друга карта или да изберете плащане при доставка.</p>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:2px 0;"><tr><td align="center" bgcolor="#1c1917" style="background:#1c1917;border-radius:2px;"><a href="{{.StorefrontURL}}/cart" style="display:inline-block;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;font-size:13px;font-weight:600;letter-spacing:.12em;text-transform:uppercase;color:#ffffff;text-decoration:none;padding:14px 30px;">Към количката</a></td></tr></table>
$html$, $text$Плащането не беше успешно

Здравейте, {{.CustomerName}}. Плащането с карта за поръчка {{.OrderNumber}} ({{.OrderTotal}}) не беше успешно и поръчката не е направена.
Не са удържани средства, а количката ви е запазена — можете да опитате с друга карта или да изберете плащане при доставка.

Към количката: {{.StorefrontURL}}/cart
$text$)

ON CONFLICT (template_key, locale) DO UPDATE SET
	subject   = EXCLUDED.subject,
	html_body = EXCLUDED.html_body,
	text_body = EXCLUDED.text_body,
	updated_at = NOW();
-- +goose StatementEnd

-- +goose Down
-- Copy-only restyle; there is no meaningful automatic rollback of email wording.
-- Re-run the original seed migration's bodies by hand if a revert is ever needed.
SELECT 1;
