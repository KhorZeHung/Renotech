package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Project struct {
	ID                  *primitive.ObjectID  `bson:"_id,omitempty" json:"_id,omitempty"`
	Folder              primitive.ObjectID   `bson:"folder" json:"folder"`
	Quotation           primitive.ObjectID   `bson:"quotation" json:"quotation"`
	Description         string               `bson:"description" json:"description"`
	Remark              string               `bson:"remark" json:"remark"`
	AreaMaterials       []SystemAreaMaterial `bson:"areaMaterials" json:"areaMaterials"`
	Discount            SystemDiscount       `bson:"discount" json:"discount"`
	TotalDiscount       float64              `bson:"totalDiscount" json:"totalDiscount"`
	TotalCharge         float64              `bson:"totalCharge" json:"totalCharge"`
	TotalNettCharge     float64              `bson:"totalNettCharge" json:"totalNettCharge"`
	TotalCost           float64              `bson:"totalCost" json:"totalCost"`
	CreatedAt           time.Time            `bson:"createdAt" json:"createdAt"`
	CreatedBy           primitive.ObjectID   `bson:"createdBy" json:"createdBy"`
	UpdatedAt           time.Time            `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy           *primitive.ObjectID  `bson:"updatedBy" json:"updatedBy"`
	EstimatedCompleteAt time.Time            `bson:"estimatedCompleteAt" json:"estimatedCompleteAt"`
	ActionLogs          []SystemActionLog    `bson:"actionLogs" json:"actionLogs"`
	PIC                 []primitive.ObjectID `bson:"pic" json:"pic"`
	Company             *primitive.ObjectID  `bson:"company" json:"company"`
	IsDeleted           bool                 `bson:"isDeleted" json:"isDeleted"`
}
