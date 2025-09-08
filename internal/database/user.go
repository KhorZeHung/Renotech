package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/enum"
)

type User struct {
	ID               *primitive.ObjectID  `bson:"_id,omitempty" json:"_id,omitempty"`
	Username         string              `bson:"username" json:"username"`
	Email            string              `bson:"email" json:"email"`
	Password         string              `bson:"password" json:"password"`
	Contact          string              `bson:"contact" json:"contact"`
	Company          *primitive.ObjectID `bson:"company,omitempty" json:"company,omitempty"`
	ProfilePicture   string              `bson:"profilePicture" json:"profilePicture"`
	Permissions      []string            `bson:"permissions" json:"permissions"`
	Type             enum.UserType       `bson:"type" json:"type"`
	Comment          string              `bson:"comment" json:"comment"`
	IsDeleted        bool                `bson:"isDeleted" json:"isDeleted"`
	IsEnabled        bool                `bson:"isEnabled" json:"isEnabled"`
	ResetToken       string              `bson:"resetToken,omitempty" json:"resetToken,omitempty"`
	ResetTokenExpiry time.Time           `bson:"resetTokenExpiry,omitempty" json:"resetTokenExpiry,omitempty"`
	CreatedAt        time.Time           `bson:"createdAt" json:"createdAt"`
	CreatedBy        *primitive.ObjectID `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	UpdatedAt        time.Time           `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy        *primitive.ObjectID `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
	LastActiveTime   time.Time           `bson:"lastActiveTime" json:"lastActiveTime"`
}
