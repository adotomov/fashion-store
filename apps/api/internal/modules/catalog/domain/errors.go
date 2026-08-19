package domain

import "errors"

var (
	ErrCatalogNotFound        = errors.New("catalog not found")
	ErrInvalidStatus          = errors.New("invalid catalog status")
	ErrCategoryNotFound       = errors.New("category not found")
	ErrProductTypeNotFound    = errors.New("product type not found")
	ErrAttributeNotFound      = errors.New("attribute not found")
	ErrAttributeValueNotFound = errors.New("attribute value not found")
	// ErrSystemAttributeReadOnly guards the built-in "Default" attributes
	// (e.g. Color) from being deleted by admins.
	ErrSystemAttributeReadOnly = errors.New("system attributes cannot be deleted")
	// ErrSystemAttributeValueReadOnly guards permanent built-in attribute
	// values (e.g. the Multicolor swatch) from being deleted by admins.
	ErrSystemAttributeValueReadOnly = errors.New("this attribute value cannot be deleted")
	ErrProductNotFound              = errors.New("product not found")
	ErrVariantNotFound              = errors.New("product variant not found")
	ErrMediaNotFound                = errors.New("product media not found")
	ErrThumbnailNotFound            = errors.New("category thumbnail not found")
	// ErrCategoryIdentifierConflict signals the internal_identifier is already
	// used by another category (partial unique index violation).
	ErrCategoryIdentifierConflict = errors.New("internal identifier is already in use by another category")
	// ErrCategoryNotAssignable signals an attempt to assign a product to a
	// placeholder (grouping-only) category.
	ErrCategoryNotAssignable = errors.New("category is a placeholder and cannot have products assigned to it")
	// ErrCategoryHasProducts signals an attempt to mark a category as a
	// placeholder while products are still assigned to it.
	ErrCategoryHasProducts = errors.New("category has products assigned; reassign them before marking it a placeholder")
)

type ValidationError string

func (e ValidationError) Error() string { return string(e) }
