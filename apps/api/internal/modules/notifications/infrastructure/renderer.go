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

// emailLayout is the shared chrome around every email. Styling is deliberately
// simple and mostly inline: many mail clients strip <style> blocks, so anything
// that must survive (colours, spacing on the container) is set inline.
var emailLayout = htmltemplate.Must(htmltemplate.New("layout").Parse(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.StoreName}}</title>
<style>
  body { margin:0; padding:0; background:#f5f5f4; }
  h1 { font-size:20px; font-weight:600; color:#1c1917; margin:0 0 16px; }
  p { margin:0 0 14px; }
  a.btn { display:inline-block; background:#1c1917; color:#ffffff !important; text-decoration:none;
          padding:11px 20px; border-radius:4px; font-weight:500; }
  table.items { width:100%; border-collapse:collapse; margin:18px 0; }
  table.items th { border-bottom:1px solid #e7e5e4; padding:8px 0; font-size:12px;
                   text-transform:uppercase; letter-spacing:.04em; color:#78716c; }
  table.items td { border-bottom:1px solid #f5f5f4; padding:8px 0; }
  table.items tr.total td { border-bottom:none; padding-top:12px; }
</style>
</head>
<body style="margin:0;padding:0;background:#f5f5f4;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f5f5f4;padding:24px 12px;">
<tr><td align="center">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0"
         style="max-width:600px;background:#ffffff;border-radius:6px;overflow:hidden;
                font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;
                font-size:15px;line-height:1.55;color:#292524;">
    <tr><td align="center" style="padding:24px 32px 8px;border-bottom:1px solid #f5f5f4;">
      {{if .LogoURL}}<img src="{{.LogoURL}}" alt="{{.StoreName}}" height="32" style="height:32px;width:auto;display:block;">
      {{else}}<span style="font-size:20px;font-weight:600;letter-spacing:.04em;color:#1c1917;">{{.StoreName}}</span>{{end}}
    </td></tr>
    <tr><td style="padding:28px 32px 8px;">{{.Content}}</td></tr>
    <tr><td style="padding:20px 32px 28px;border-top:1px solid #f5f5f4;color:#a8a29e;font-size:12px;line-height:1.6;">
      {{if .SupportEmail}}<div>Questions? <a href="mailto:{{.SupportEmail}}" style="color:#78716c;">{{.SupportEmail}}</a></div>{{end}}
      {{if .PostalAddress}}<div>{{.PostalAddress}}</div>{{end}}
      <div>&copy; {{.Year}} {{.StoreName}}</div>
    </td></tr>
  </table>
</td></tr>
</table>
</body>
</html>`))
