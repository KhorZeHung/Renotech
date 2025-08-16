package database

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Supplier struct {
	ID               *primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Label            string              `bson:"label" json:"label"`
	Name             string              `bson:"name" json:"name"`
	Contact          string              `bson:"contact" json:"contact"`
	Email            string              `bson:"email" json:"email"`
	Logo             string              `bson:"logo" json:"logo"`
	Tags             []string            `bson:"tags" json:"tags"`
	Description      string              `bson:"description" json:"description"`
	OfficeAddress    []SystemAddress     `bson:"officeAddress" json:"officeAddress"`
	WarehouseAddress []SystemAddress     `bson:"warehouseAddress" json:"warehouseAddress"`
	Company          primitive.ObjectID  `bson:"company" json:"company"`
	CreatedAt        time.Time           `bson:"createdAt" json:"createdAt"`
	CreatedBy        primitive.ObjectID  `bson:"createdBy" json:"createdBy"`
	UpdatedAt        time.Time           `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy        *primitive.ObjectID `bson:"updatedBy" json:"updatedBy"`
	IsDeleted        bool                `bson:"isDeleted" json:"isDeleted"`
}
