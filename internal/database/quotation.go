package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Quotation struct {
	ID              *primitive.ObjectID  `bson:"_id,omitempty" json:"_id,omitempty"`
	Folder          primitive.ObjectID   `bson:"folder" json:"folder"`
	Name            string               `bson:"name" json:"name"`
	Description     string               `bson:"description" json:"description"`
	Remark          string               `bson:"remark" json:"remark"`
	AreaMaterials   []SystemAreaMaterial `bson:"areaMaterials" json:"areaMaterials"`
	Discount        SystemDiscount       `bson:"discount" json:"discount"`
	IsStared        bool                 `bson:"isStared" json:"isStared"`
	TotalDiscount   float64              `bson:"totalDiscount" json:"totalDiscount"`
	TotalCharge     float64              `bson:"totalCharge" json:"totalCharge"`
	TotalNettCharge float64              `bson:"totalNettCharge" json:"totalNettCharge"`
	TotalCost       float64              `bson:"totalCost" json:"totalCost"`
	Company         *primitive.ObjectID  `bson:"company" json:"company"`
	CreatedAt       time.Time            `bson:"createdAt" json:"createdAt"`
	CreatedBy       primitive.ObjectID   `bson:"createdBy" json:"createdBy"`
	UpdatedAt       time.Time            `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy       *primitive.ObjectID  `bson:"updatedBy" json:"updatedBy"`
	IsDeleted       bool                 `bson:"isDeleted" json:"isDeleted"`
}
