package domain

import (
	"time"

	"github.com/google/uuid"
)

type ProductMedia struct {
	ID          uuid.UUID
	ProductID   uuid.UUID
	Bucket      string
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	Position    int
	AltText     string
	CreatedAt   time.Time
}
