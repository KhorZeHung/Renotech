package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Quotation struct {
	ID                    *primitive.ObjectID      `bson:"_id,omitempty" json:"_id,omitempty"`
	Folder                *primitive.ObjectID      `bson:"folder" json:"folder"`
	Name                  string                   `bson:"name" json:"name"`
	Client                SystemClient             `bson:"client" json:"client"`
	Budget                float64                  `bson:"budget" json:"budget"`
	Address               SystemAddress            `bson:"address" json:"address"`
	ExpiredAt             time.Time                `bson:"expiredAt" json:"expiredAt"`
	Description           string                   `bson:"description" json:"description"`
	Remark                string                   `bson:"remark" json:"remark"`
	AreaMaterials         []SystemAreaMaterial     `bson:"areaMaterials" json:"areaMaterials"`
	Discounts             []SystemDiscount         `bson:"discounts" json:"discounts"`
	AdditionalCharges     []SystemAdditionalCharge `bson:"additionalCharges" json:"additionalCharges"`
	IsStared              bool                     `bson:"isStared" json:"isStared"`
	TotalCharge           float64                  `bson:"totalCharge" json:"totalCharge"`
	TotalDiscount         float64                  `bson:"totalDiscount" json:"totalDiscount"`
	TotalAdditionalCharge float64                  `bson:"totalAdditionalCharge" json:"totalAdditionalCharge"`
	TotalNettCharge       float64                  `bson:"totalNettCharge" json:"totalNettCharge"`
	Media                 []SystemMedia            `bson:"media" json:"media"`
	Company               *primitive.ObjectID      `bson:"company" json:"company"`
	CreatedAt             time.Time                `bson:"createdAt" json:"createdAt"`
	CreatedBy             primitive.ObjectID       `bson:"createdBy" json:"createdBy"`
	UpdatedAt             time.Time                `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy             *primitive.ObjectID      `bson:"updatedBy" json:"updatedBy"`
	IsDeleted             bool                     `bson:"isDeleted" json:"isDeleted"`
}
