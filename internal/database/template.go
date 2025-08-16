package database

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Template struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty" json:"_id,omitempty"`
	Name        string              `bson:"name" json:"name"`
	Company     primitive.ObjectID  `bson:"company" json:"company"`
	Materials   []TemplateMaterials `bson:"materials" json:"materials"`
	Description string              `bson:"description" json:"description"`
	Media       []SystemMedia       `bson:"media" json:"media"`
	CreatedAt   time.Time           `bson:"createdAt" json:"createdAt"`
	CreatedBy   primitive.ObjectID  `bson:"createdBy" json:"createdBy"`
	UpdatedAt   time.Time           `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy   *primitive.ObjectID `bson:"updatedBy" json:"updatedBy"`
	IsDeleted   bool                `bson:"isDeleted" json:"isDeleted"`
}

type TemplateMaterials struct {
	Material        primitive.ObjectID `bson:"material" json:"material"`
	MaterialDoc     Material           `bson:"materialDoc" json:"materialDoc"`
	DefaultQuantity float64            `bson:"defaultQuantity" json:"defaultQuantity"`
}
