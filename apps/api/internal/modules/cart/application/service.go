package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/cart/domain"
)

// CartOwner identifies whose cart is being acted on: an authenticated user,
// an anonymous guest (identified by an opaque token the client persists),
// or neither (a first-time anonymous visitor who hasn't added anything yet).
type CartOwner struct {
	UserID     *uuid.UUID
	GuestToken *uuid.UUID
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func emptyCart() *domain.Cart {
	return &domain.Cart{Items: []domain.CartItem{}}
}

// GetCart returns the owner's cart, or an empty (unpersisted) cart if they
// don't have one yet — viewing the cart should never create one.
func (s *Service) GetCart(ctx context.Context, owner CartOwner) (*domain.Cart, error) {
	cart, err := s.findCart(ctx, owner)
	if errors.Is(err, domain.ErrCartNotFound) {
		return emptyCart(), nil
	}
	if err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *Service) findCart(ctx context.Context, owner CartOwner) (*domain.Cart, error) {
	if owner.UserID != nil {
		return s.repo.FindByUser(ctx, *owner.UserID)
	}
	if owner.GuestToken != nil {
		return s.repo.FindByGuestToken(ctx, *owner.GuestToken)
	}
	return nil, domain.ErrCartNotFound
}

// resolveOrCreateCart finds the owner's cart, creating one if needed. A
// guest with no token yet is assigned a brand new one — the caller (HTTP
// handler) surfaces it back to the client to persist for subsequent calls.
func (s *Service) resolveOrCreateCart(ctx context.Context, owner CartOwner) (*domain.Cart, error) {
	if owner.UserID != nil {
		cart, err := s.repo.FindByUser(ctx, *owner.UserID)
		if errors.Is(err, domain.ErrCartNotFound) {
			return s.repo.CreateForUser(ctx, *owner.UserID)
		}
		return cart, err
	}

	token := owner.GuestToken
	if token != nil {
		cart, err := s.repo.FindByGuestToken(ctx, *token)
		if errors.Is(err, domain.ErrCartNotFound) {
			return s.repo.CreateForGuest(ctx, *token)
		}
		return cart, err
	}

	newToken := uuid.New()
	return s.repo.CreateForGuest(ctx, newToken)
}

func (s *Service) AddItem(ctx context.Context, owner CartOwner, variantID uuid.UUID, quantity int) (*domain.Cart, error) {
	if quantity < 1 {
		return nil, domain.ValidationError("quantity must be at least 1")
	}
	cart, err := s.resolveOrCreateCart(ctx, owner)
	if err != nil {
		return nil, err
	}
	return s.repo.AddOrIncrementItem(ctx, cart.ID, variantID, quantity)
}

func (s *Service) UpdateItemQuantity(ctx context.Context, owner CartOwner, itemID uuid.UUID, quantity int) (*domain.Cart, error) {
	if quantity < 1 {
		return nil, domain.ValidationError("quantity must be at least 1")
	}
	cart, err := s.findCart(ctx, owner)
	if err != nil {
		return nil, err
	}
	return s.repo.SetItemQuantity(ctx, cart.ID, itemID, quantity)
}

func (s *Service) RemoveItem(ctx context.Context, owner CartOwner, itemID uuid.UUID) (*domain.Cart, error) {
	cart, err := s.findCart(ctx, owner)
	if err != nil {
		return nil, err
	}
	return s.repo.RemoveItem(ctx, cart.ID, itemID)
}

// ClearCart empties the owner's cart once an order has been placed from
// it. A missing cart is a no-op — nothing to clear.
func (s *Service) ClearCart(ctx context.Context, owner CartOwner) error {
	cart, err := s.findCart(ctx, owner)
	if errors.Is(err, domain.ErrCartNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.repo.ClearItems(ctx, cart.ID)
}

// MergeGuestCartIntoUser folds an anonymous cart into the now-authenticated
// user's cart — called right after login so items added before sign-in
// aren't lost. If the guest token has no cart (or is empty/unknown), the
// user simply ends up with their own existing-or-new cart.
func (s *Service) MergeGuestCartIntoUser(ctx context.Context, userID, guestToken uuid.UUID) (*domain.Cart, error) {
	guestCart, err := s.repo.FindByGuestToken(ctx, guestToken)
	if errors.Is(err, domain.ErrCartNotFound) {
		return s.resolveOrCreateCart(ctx, CartOwner{UserID: &userID})
	}
	if err != nil {
		return nil, err
	}

	userCart, err := s.resolveOrCreateCart(ctx, CartOwner{UserID: &userID})
	if err != nil {
		return nil, err
	}
	if guestCart.ID == userCart.ID {
		return userCart, nil
	}
	return s.repo.MergeCarts(ctx, guestCart.ID, userCart.ID)
}
