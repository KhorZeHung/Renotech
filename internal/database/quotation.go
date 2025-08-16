package database

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Quotation struct {
	ID            *primitive.ObjectID  `bson:"_id,omitempty" json:"_id,omitempty"`
	Folder        primitive.ObjectID   `bson:"folder" json:"folder"`
	Description   string               `bson:"description" json:"description"`
	Remark        string               `bson:"remark" json:"remark"`
	AreaMaterials []SystemAreaMaterial `bson:"areaMaterials" json:"areaMaterials"`
	Discount      SystemDiscount       `bson:"discount" json:"discount"`
	TotalDiscount float64              `bson:"totalDiscount" json:"totalDiscount"`
	TotalCharge   float64              `bson:"totalCharge" json:"totalCharge"`
	TotalCost     float64              `bson:"totalCost" json:"totalCost"`
	IsStared      bool                 `bson:"isStared" json:"isStared"`
	CreatedAt     time.Time            `bson:"createdAt" json:"createdAt"`
	CreatedBy     primitive.ObjectID   `bson:"createdBy" json:"createdBy"`
	UpdatedAt     time.Time            `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy     *primitive.ObjectID  `bson:"updatedBy" json:"updatedBy"`
	IsDeleted     bool                 `bson:"isDeleted" json:"isDeleted"`
}
