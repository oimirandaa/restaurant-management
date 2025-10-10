package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Menu struct {
	ID        primitive.ObjectID `bson:"_id"`
	Name      string             `json:"name" validate:"required,min=2,max=100"`
	Category  string             `json:"category" validate:"required"`
	StartDate *time.Time         `json:"start_date" validate:"required"`
	EndDate   *time.Time         `json:"end_date" validate:"required"`
	CreatedAt time.Time          `json:"created_at" validate:"required"`
	UpdatedAt time.Time          `json:"updated_at" validate:"required"`
	MenuId    string             `json:"menu_id"`
}
