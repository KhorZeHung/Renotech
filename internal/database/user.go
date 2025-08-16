package database

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Username       string             `bson:"username" json:"username"`
	Email          string             `bson:"email" json:"email"`
	Password       string             `bson:"password" json:"password"`
	Contact        string             `bson:"contact" json:"contact"`
	Company        string             `bson:"company" json:"company"`
	ProfilePicture string             `bson:"profilePicture" json:"profilePicture"`
	Permissions    []string           `bson:"permissions" json:"permissions"`
	Comment        string             `bson:"comment" json:"comment"`
	IsDeleted      bool               `bson:"isDeleted" json:"isDeleted"`
	IsEnabled      bool               `bson:"isEnabled" json:"isEnabled"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt" json:"updatedAt"`
	LastActiveTime time.Time          `bson:"lastActiveTime" json:"lastActiveTime"`
}
