package database

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/enum"
	"time"
)

type Material struct {
	ID                  *primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name                string              `bson:"name" json:"name"`
	ClientDisplayName   string              `bson:"clientDisplayName" json:"clientDisplayName"`
	SupplierDisplayName string              `bson:"supplierDisplayName" json:"supplierDisplayName"`
	Type                enum.MaterialType   `bson:"type" json:"type"`
	Supplier            *primitive.ObjectID `bson:"supplier" json:"supplier"`
	Brand               string              `bson:"brand" json:"brand"`
	Unit                string              `bson:"unit" json:"unit"`
	CostPerUnit         float64             `bson:"costPerUnit" json:"costPerUnit"`
	PricePerUnit        float64             `bson:"pricePerUnit" json:"pricePerUnit"`
	MarkupRate          float64             `bson:"markupRate" json:"markupRate"`
	Categories          []string            `bson:"categories" json:"categories"`
	Tags                []string            `bson:"tags" json:"tags"`
	Media               []SystemMedia       `bson:"media" json:"media"`
	Company             primitive.ObjectID  `bson:"company" json:"company"`
	Status              enum.MaterialStatus `bson:"status" json:"status"`
	Remark              string              `bson:"remark" json:"remark"`
	CreatedAt           time.Time           `bson:"createdAt" json:"createdAt"`
	CreatedBy           primitive.ObjectID  `bson:"createdBy" json:"createdBy"`
	UpdatedAt           time.Time           `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy           primitive.ObjectID  `bson:"updatedBy" json:"updatedBy"`
	IsDeleted           bool                `bson:"isDeleted" json:"isDeleted"`
}
