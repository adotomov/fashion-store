package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"strings"
	"sync"
	texttemplate "text/template"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/domain"
)

// brandingTTL caches store branding briefly so a batch of emails costs one
// settings read rather than one per message, while still picking up an admin
// change within a few minutes.
const brandingTTL = 5 * time.Minute

// TemplateRenderer renders a stored template fragment inside the shared email
// layout. Subjects and the plain-text part use text/template (HTML-escaping a
// subject header would corrupt characters like &), while the HTML body uses
// html/template so customer-supplied values are escaped.
type TemplateRenderer struct {
	branding application.BrandingProvider

	mu       sync.Mutex
	cached   domain.Branding
	cachedAt time.Time
}

func NewTemplateRenderer(branding application.BrandingProvider) *TemplateRenderer {
	return &TemplateRenderer{branding: branding}
}

func (r *TemplateRenderer) Render(ctx context.Context, tmpl domain.Template, vars map[string]any) (domain.Rendered, error) {
	branding, err := r.currentBranding(ctx)
	if err != nil {
		return domain.Rendered{}, fmt.Errorf("load branding: %w", err)
	}

	data := mergeData(vars, branding)

	subject, err := renderText("subject:"+tmpl.Key, tmpl.Subject, data)
	if err != nil {
		return domain.Rendered{}, fmt.Errorf("subject: %w", err)
	}

	body, err := renderHTML("html:"+tmpl.Key, tmpl.HTMLBody, data)
	if err != nil {
		return domain.Rendered{}, fmt.Errorf("html body: %w", err)
	}

	html, err := renderLayout(body, data)
	if err != nil {
		return domain.Rendered{}, fmt.Errorf("layout: %w", err)
	}

	var text string
	if strings.TrimSpace(tmpl.TextBody) != "" {
		text, err = renderText("text:"+tmpl.Key, tmpl.TextBody, data)
		if err != nil {
			return domain.Rendered{}, fmt.Errorf("text body: %w", err)
		}
	}

	return domain.Rendered{
		Subject: strings.TrimSpace(subject),
		HTML:    html,
		Text:    strings.TrimSpace(text),
	}, nil
}

func (r *TemplateRenderer) currentBranding(ctx context.Context) (domain.Branding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.cachedAt.IsZero() && time.Since(r.cachedAt) < brandingTTL {
		return r.cached, nil
	}
	branding, err := r.branding.Branding(ctx)
	if err != nil {
		// Serve a stale value rather than failing the send — branding is
		// cosmetic, and dead-lettering an order confirmation over it is worse.
		if !r.cachedAt.IsZero() {
			return r.cached, nil
		}
		return domain.Branding{}, err
	}
	r.cached = branding
	r.cachedAt = time.Now()
	return branding, nil
}

// mergeData layers branding over the producer's variables so every template can
// rely on StoreName/LogoURL/StorefrontURL/Year being present and consistent.
func mergeData(vars map[string]any, b domain.Branding) map[string]any {
	data := make(map[string]any, len(vars)+6)
	for k, v := range vars {
		data[k] = v
	}
	data["StoreName"] = b.StoreName
	data["LogoURL"] = b.LogoURL
	data["StorefrontURL"] = strings.TrimSuffix(b.StorefrontURL, "/")
	data["SupportEmail"] = b.SupportEmail
	data["PostalAddress"] = b.PostalAddress
	data["Year"] = time.Now().Year()
	// Preheader is the hidden inbox-preview line. Templates may set it in their
	// payload; default it so the layout never renders "<no value>" for the key.
	if _, ok := data["Preheader"]; !ok {
		data["Preheader"] = ""
	}
	return data
}

func renderText(name, body string, data map[string]any) (string, error) {
	t, err := texttemplate.New(name).Option("missingkey=zero").Parse(body)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderHTML(name, body string, data map[string]any) (htmltemplate.HTML, error) {
	t, err := htmltemplate.New(name).Option("missingkey=zero").Parse(body)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return htmltemplate.HTML(buf.String()), nil
}

func renderLayout(content htmltemplate.HTML, data map[string]any) (string, error) {
	layoutData := make(map[string]any, len(data)+1)
	for k, v := range data {
		layoutData[k] = v
	}
	// Already rendered and escaped by renderHTML; marked safe so the layout does
	// not escape it a second time.
	layoutData["Content"] = content

	var buf bytes.Buffer
	if err := emailLayout.Execute(&buf, layoutData); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// emailLayout is the shared chrome around every email. The look is an editorial
// boutique style: a serif wordmark and headings on a warm ivory canvas, a single
// white card, and refined tables/panels. Typography and colour live in a <style>
// block for the modern clients that keep it; anything that must not break when a
// client strips <style> (the container, the CTA buttons, the panels, the totals
// rule) is also inlined in the fragments, so the email degrades gracefully.
var emailLayout = htmltemplate.Must(htmltemplate.New("layout").Parse(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="x-apple-disable-message-reformatting">
<title>{{.StoreName}}</title>
<style>
  body,table,td,a { -webkit-text-size-adjust:100%; -ms-text-size-adjust:100%; }
  table,td { mso-table-lspace:0pt; mso-table-rspace:0pt; }
  img { border:0; line-height:100%; outline:none; text-decoration:none; -ms-interpolation-mode:bicubic; }
  body { margin:0; padding:0; width:100%; background:#f4f1ec; }

  .wordmark { font-family:Georgia,'Times New Roman',serif; font-size:22px; letter-spacing:.28em;
              text-transform:uppercase; color:#1c1917; }
  h1 { font-family:Georgia,'Times New Roman',serif; font-weight:400; font-size:24px; line-height:1.28;
       color:#1c1917; margin:0 0 18px; }
  p  { font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;
       font-size:15px; line-height:1.65; color:#44403c; margin:0 0 16px; }
  .eyebrow { font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;
             font-size:11px; letter-spacing:.18em; text-transform:uppercase; color:#a8a29e; margin:0 0 14px; }
  .muted { color:#78716c; font-size:13px; line-height:1.6; }

  table.items { width:100%; border-collapse:collapse; margin:6px 0 4px;
                font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif; }
  table.items th { text-align:left; font-size:11px; letter-spacing:.1em; text-transform:uppercase;
                   color:#a8a29e; font-weight:600; padding:0 0 10px; border-bottom:1px solid #ece7e0; }
  table.items td { font-size:15px; color:#292524; padding:12px 0; border-bottom:1px solid #f4f1ec; vertical-align:top; }
  table.items td.num { text-align:right; white-space:nowrap; }
  table.items .variant { color:#a8a29e; font-size:13px; }
  table.items tr.sub td { border-bottom:none; color:#78716c; padding:5px 0; }
  table.items tr.total td { border-top:2px solid #1c1917; border-bottom:none; padding-top:14px;
                            font-size:17px; font-weight:700; color:#1c1917; }

  @media only screen and (max-width:600px){
    .px { padding-left:24px !important; padding-right:24px !important; }
    h1 { font-size:22px !important; }
  }
</style>
</head>
<body style="margin:0;padding:0;background:#f4f1ec;">
<div style="display:none;max-height:0;overflow:hidden;mso-hide:all;font-size:1px;line-height:1px;color:#f4f1ec;opacity:0;">{{.Preheader}}</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4f1ec;">
<tr><td align="center" style="padding:30px 12px;">
  <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:600px;max-width:600px;">
    <tr><td align="center" style="padding:6px 0 24px;">
      {{if .LogoURL}}<img src="{{.LogoURL}}" alt="{{.StoreName}}" height="34" style="height:34px;width:auto;display:block;">
      {{else}}<span class="wordmark" style="font-family:Georgia,'Times New Roman',serif;font-size:22px;letter-spacing:.28em;text-transform:uppercase;color:#1c1917;">{{.StoreName}}</span>{{end}}
    </td></tr>
    <tr><td>
      <table role="presentation" width="100%" cellpadding="0" cellspacing="0"
             style="background:#ffffff;border:1px solid #ece7e0;border-radius:8px;overflow:hidden;">
        <tr><td class="px" style="padding:40px;">{{.Content}}</td></tr>
      </table>
    </td></tr>
    <tr><td class="px" style="padding:22px 40px 6px;">
      {{if .SupportEmail}}<p class="muted" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;color:#78716c;font-size:13px;line-height:1.6;margin:0 0 6px;">Questions? Reach us at <a href="mailto:{{.SupportEmail}}" style="color:#78716c;">{{.SupportEmail}}</a></p>{{end}}
      {{if .PostalAddress}}<p class="muted" style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;color:#78716c;font-size:13px;line-height:1.6;margin:0 0 6px;">{{.PostalAddress}}</p>{{end}}
      <p style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;color:#c4bcb0;font-size:12px;margin:0;">&copy; {{.Year}} {{.StoreName}}</p>
    </td></tr>
  </table>
</td></tr>
</table>
</body>
</html>`))
