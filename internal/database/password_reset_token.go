package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PasswordResetToken struct {
	ID         *primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Email      string              `bson:"email" json:"email"`
	ResetToken string              `bson:"resetToken" json:"resetToken"`
	ExpiresAt  time.Time           `bson:"expiresAt" json:"expiresAt"`
	CreatedAt  time.Time           `bson:"createdAt" json:"createdAt"`
	IsUsed     bool                `bson:"isUsed" json:"isUsed"`
}