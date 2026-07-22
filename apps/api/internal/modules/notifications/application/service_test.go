package application_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/notifications/domain"
)

// --- fakes -----------------------------------------------------------------

type fakeRepo struct {
	enqueued []domain.Message
	// dedupe mimics the unique index on dedupe_key.
	seen map[string]bool
	due  []domain.Message

	sent       map[uuid.UUID]string
	retried    map[uuid.UUID]time.Time
	failed     map[uuid.UUID]string
	suppressed map[uuid.UUID]string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		seen:       map[string]bool{},
		sent:       map[uuid.UUID]string{},
		retried:    map[uuid.UUID]time.Time{},
		failed:     map[uuid.UUID]string{},
		suppressed: map[uuid.UUID]string{},
	}
}

func (r *fakeRepo) Enqueue(_ context.Context, msg domain.Message) (bool, error) {
	if r.seen[msg.DedupeKey] {
		return false, nil
	}
	r.seen[msg.DedupeKey] = true
	r.enqueued = append(r.enqueued, msg)
	return true, nil
}

func (r *fakeRepo) ClaimDue(_ context.Context, _ int, _ time.Duration) ([]domain.Message, error) {
	due := r.due
	r.due = nil
	return due, nil
}

func (r *fakeRepo) MarkSent(_ context.Context, id uuid.UUID, providerMessageID string) error {
	r.sent[id] = providerMessageID
	return nil
}

func (r *fakeRepo) MarkRetry(_ context.Context, id uuid.UUID, next time.Time, _ string) error {
	r.retried[id] = next
	return nil
}

func (r *fakeRepo) MarkFailed(_ context.Context, id uuid.UUID, reason string) error {
	r.failed[id] = reason
	return nil
}

func (r *fakeRepo) MarkSuppressed(_ context.Context, id uuid.UUID, reason string) error {
	r.suppressed[id] = reason
	return nil
}

func (r *fakeRepo) MarkDeliveryFailure(_ context.Context, _, _ string) error { return nil }

type fakeTemplates struct {
	byKeyLocale map[string]domain.Template
}

func (t *fakeTemplates) Get(_ context.Context, key, locale string) (domain.Template, error) {
	tmpl, ok := t.byKeyLocale[key+":"+locale]
	if !ok {
		return domain.Template{}, domain.ErrTemplateNotFound
	}
	return tmpl, nil
}

type fakeRenderer struct{ err error }

func (r *fakeRenderer) Render(_ context.Context, tmpl domain.Template, _ map[string]any) (domain.Rendered, error) {
	if r.err != nil {
		return domain.Rendered{}, r.err
	}
	return domain.Rendered{Subject: tmpl.Subject, HTML: tmpl.HTMLBody, Text: tmpl.TextBody}, nil
}

type fakeSender struct {
	err   error
	calls int
	last  application.SendRequest
}

func (s *fakeSender) Send(_ context.Context, req application.SendRequest) (string, error) {
	s.calls++
	s.last = req
	if s.err != nil {
		return "", s.err
	}
	return "provider-123", nil
}

type fakeSuppressions struct {
	blocked map[string]bool
}

func (s *fakeSuppressions) IsSuppressed(_ context.Context, email string) (bool, error) {
	return s.blocked[email], nil
}

func (s *fakeSuppressions) Suppress(_ context.Context, email, _, _ string) error {
	if s.blocked == nil {
		s.blocked = map[string]bool{}
	}
	s.blocked[email] = true
	return nil
}

type fakeBranding struct{ locale string }

func (b *fakeBranding) Branding(context.Context) (domain.Branding, error) {
	return domain.Branding{StoreName: "Test Store", Locale: b.locale}, nil
}

// --- helpers ---------------------------------------------------------------

func newService(repo *fakeRepo, templates *fakeTemplates, renderer *fakeRenderer, sender *fakeSender, suppressions application.SuppressionStore, locale string) *application.Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return application.NewService(repo, templates, renderer, sender, suppressions, &fakeBranding{locale: locale}, "en", logger)
}

func seededTemplates() *fakeTemplates {
	return &fakeTemplates{byKeyLocale: map[string]domain.Template{
		"welcome:en": {Key: "welcome", Locale: "en", Subject: "Welcome", HTMLBody: "<p>Hi</p>"},
		"welcome:bg": {Key: "welcome", Locale: "bg", Subject: "Здравейте", HTMLBody: "<p>Здравей</p>"},
	}}
}

func dueMessage(attempts int) domain.Message {
	return domain.Message{
		ID:          uuid.New(),
		TemplateKey: "welcome",
		Locale:      "en",
		ToEmail:     "customer@example.com",
		DedupeKey:   "welcome:1",
		Status:      domain.StatusSending,
		Attempts:    attempts,
		Payload:     map[string]any{},
	}
}

// --- tests -----------------------------------------------------------------

func TestEnqueue_IsIdempotentOnDedupeKey(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo, seededTemplates(), &fakeRenderer{}, &fakeSender{}, nil, "en")

	input := application.EnqueueInput{
		TemplateKey: domain.TemplateWelcome,
		ToEmail:     "customer@example.com",
		DedupeKey:   "welcome:user-1",
	}
	for i := 0; i < 3; i++ {
		if err := svc.Enqueue(context.Background(), input); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if len(repo.enqueued) != 1 {
		t.Fatalf("expected the duplicate enqueues to be no-ops, got %d messages", len(repo.enqueued))
	}
}

func TestEnqueue_UsesStoreLocaleWhenUnspecified(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo, seededTemplates(), &fakeRenderer{}, &fakeSender{}, nil, "bg")

	err := svc.Enqueue(context.Background(), application.EnqueueInput{
		TemplateKey: domain.TemplateWelcome,
		ToEmail:     "customer@example.com",
		DedupeKey:   "welcome:user-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := repo.enqueued[0].Locale; got != "bg" {
		t.Fatalf("expected the store locale bg, got %q", got)
	}
}

func TestEnqueue_RejectsMessageWithoutRecipient(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo, seededTemplates(), &fakeRenderer{}, &fakeSender{}, nil, "en")

	err := svc.Enqueue(context.Background(), application.EnqueueInput{
		TemplateKey: domain.TemplateWelcome,
		DedupeKey:   "welcome:user-1",
	})
	if err == nil {
		t.Fatal("expected a validation error for a missing recipient")
	}
}

func TestDispatch_SendsAndMarksSent(t *testing.T) {
	repo := newFakeRepo()
	msg := dueMessage(1)
	repo.due = []domain.Message{msg}
	sender := &fakeSender{}

	svc := newService(repo, seededTemplates(), &fakeRenderer{}, sender, nil, "en")
	if _, err := svc.DispatchDue(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sender.calls != 1 {
		t.Fatalf("expected exactly one send, got %d", sender.calls)
	}
	if repo.sent[msg.ID] != "provider-123" {
		t.Fatalf("expected the provider message id to be recorded, got %q", repo.sent[msg.ID])
	}
}

func TestDispatch_RetriesTransientFailureWithBackoff(t *testing.T) {
	repo := newFakeRepo()
	msg := dueMessage(1)
	repo.due = []domain.Message{msg}
	sender := &fakeSender{err: errors.New("provider unavailable")}

	svc := newService(repo, seededTemplates(), &fakeRenderer{}, sender, nil, "en")
	if _, err := svc.DispatchDue(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	next, ok := repo.retried[msg.ID]
	if !ok {
		t.Fatal("expected the message to be scheduled for retry")
	}
	if !next.After(time.Now()) {
		t.Fatal("expected the retry to be scheduled in the future")
	}
	if _, dead := repo.failed[msg.ID]; dead {
		t.Fatal("expected the message not to be dead-lettered on its first failure")
	}
}

func TestDispatch_DeadLettersAfterMaxAttempts(t *testing.T) {
	repo := newFakeRepo()
	msg := dueMessage(application.MaxAttempts)
	repo.due = []domain.Message{msg}
	sender := &fakeSender{err: errors.New("provider unavailable")}

	svc := newService(repo, seededTemplates(), &fakeRenderer{}, sender, nil, "en")
	if _, err := svc.DispatchDue(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, dead := repo.failed[msg.ID]; !dead {
		t.Fatal("expected the message to be dead-lettered once retries are exhausted")
	}
	if _, retried := repo.retried[msg.ID]; retried {
		t.Fatal("expected no further retry to be scheduled")
	}
}

func TestDispatch_SkipsSuppressedRecipient(t *testing.T) {
	repo := newFakeRepo()
	msg := dueMessage(1)
	repo.due = []domain.Message{msg}
	sender := &fakeSender{}
	suppressions := &fakeSuppressions{blocked: map[string]bool{"customer@example.com": true}}

	svc := newService(repo, seededTemplates(), &fakeRenderer{}, sender, suppressions, "en")
	if _, err := svc.DispatchDue(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sender.calls != 0 {
		t.Fatal("expected no send for a suppressed recipient")
	}
	if _, ok := repo.suppressed[msg.ID]; !ok {
		t.Fatal("expected the message to be marked suppressed")
	}
}

func TestDispatch_FallsBackToDefaultLocaleTemplate(t *testing.T) {
	repo := newFakeRepo()
	msg := dueMessage(1)
	msg.Locale = "de" // never seeded
	repo.due = []domain.Message{msg}
	sender := &fakeSender{}

	svc := newService(repo, seededTemplates(), &fakeRenderer{}, sender, nil, "en")
	if _, err := svc.DispatchDue(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sender.calls != 1 {
		t.Fatalf("expected the send to fall back to the default locale, got %d sends", sender.calls)
	}
	if sender.last.Subject != "Welcome" {
		t.Fatalf("expected the English template, got subject %q", sender.last.Subject)
	}
}

func TestDispatch_DeadLettersUnrenderableTemplateImmediately(t *testing.T) {
	repo := newFakeRepo()
	msg := dueMessage(1)
	repo.due = []domain.Message{msg}
	sender := &fakeSender{}

	svc := newService(repo, seededTemplates(), &fakeRenderer{err: errors.New("bad template")}, sender, nil, "en")
	if _, err := svc.DispatchDue(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sender.calls != 0 {
		t.Fatal("expected no send attempt when rendering fails")
	}
	if _, dead := repo.failed[msg.ID]; !dead {
		t.Fatal("expected an unrenderable template to dead-letter immediately rather than retry")
	}
}

func TestHandleProviderEvent_SuppressesHardBounceOnly(t *testing.T) {
	repo := newFakeRepo()
	suppressions := &fakeSuppressions{blocked: map[string]bool{}}
	svc := newService(repo, seededTemplates(), &fakeRenderer{}, &fakeSender{}, suppressions, "en")

	soft := application.ProviderEvent{
		Type:       application.EventBounce,
		Email:      "soft@example.com",
		BounceType: application.BounceTypeBlocked,
	}
	if err := svc.HandleProviderEvent(context.Background(), soft); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suppressions.blocked["soft@example.com"] {
		t.Fatal("expected a soft bounce not to suppress the address")
	}

	hard := application.ProviderEvent{
		Type:       application.EventBounce,
		Email:      "hard@example.com",
		BounceType: application.BounceTypePermanent,
	}
	if err := svc.HandleProviderEvent(context.Background(), hard); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !suppressions.blocked["hard@example.com"] {
		t.Fatal("expected a hard bounce to suppress the address")
	}
}

func TestHandleProviderEvent_SuppressesSpamComplaint(t *testing.T) {
	repo := newFakeRepo()
	suppressions := &fakeSuppressions{blocked: map[string]bool{}}
	svc := newService(repo, seededTemplates(), &fakeRenderer{}, &fakeSender{}, suppressions, "en")

	ev := application.ProviderEvent{Type: application.EventSpamReport, Email: "angry@example.com"}
	if err := svc.HandleProviderEvent(context.Background(), ev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !suppressions.blocked["angry@example.com"] {
		t.Fatal("expected a spam complaint to suppress the address")
	}
}

func TestHandleProviderEvent_IgnoresUnknownType(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo, seededTemplates(), &fakeRenderer{}, &fakeSender{}, &fakeSuppressions{}, "en")

	// An event type we don't handle must not error, or the provider would retry
	// the whole batch forever.
	err := svc.HandleProviderEvent(context.Background(), application.ProviderEvent{Type: "open", Email: "a@b.com"})
	if err != nil {
		t.Fatalf("expected unknown events to be ignored, got %v", err)
	}
}
