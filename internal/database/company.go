package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Company struct {
	ID                  *primitive.ObjectID  `bson:"_id,omitempty" json:"_id,omitempty"`
	Name                string               `bson:"name" json:"name"`
	ClientDisplayName   string               `bson:"clientDisplayName" json:"clientDisplayName"`
	SupplierDisplayName string               `bson:"supplierDisplayName" json:"supplierDisplayName"`
	Address             string               `bson:"address" json:"address"`
	Website             string               `bson:"website" json:"website"`
	Owner               *primitive.ObjectID  `bson:"owner,omitempty" json:"owner,omitempty"`
	Logo                string               `bson:"logo" json:"logo"`
	Contact             string               `bson:"contact" json:"contact"`
	TermCondition       []string             `bson:"termCondition" json:"termCondition"`
	IsDeleted           bool                 `bson:"isDeleted" json:"isDeleted"`
	IsEnabled           bool                 `bson:"isEnabled" json:"isEnabled"`
	CreatedAt           time.Time            `bson:"createdAt" json:"createdAt"`
	CreatedBy           *primitive.ObjectID  `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	UpdatedAt           time.Time            `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy           *primitive.ObjectID  `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
}
