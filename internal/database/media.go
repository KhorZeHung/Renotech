package database

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/enum"
	"time"
)

type Media struct {
	ID        *primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name      string              `bson:"name" json:"name"`
	Path      string              `bson:"path" json:"path"`
	Type      enum.MediaType      `bson:"type" json:"type"`
	Extension string              `bson:"extension" json:"extension"`
	FileName  string              `bson:"fileName" json:"fileName"`
	Company   primitive.ObjectID  `bson:"company" json:"company"`
	CreatedBy primitive.ObjectID  `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	CreatedAt time.Time           `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
}
