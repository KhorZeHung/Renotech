package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Project struct {
	ID                    *primitive.ObjectID        `bson:"_id,omitempty" json:"_id,omitempty"`
	Name                  string                     `bson:"name" json:"name"`
	Client                SystemClient               `bson:"client" json:"client"`
	Budget                float64                    `bson:"budget" json:"budget"`
	Address               SystemAddress              `bson:"address" json:"address"`
	Description           string                     `bson:"description" json:"description"`
	Remark                string                     `bson:"remark" json:"remark"`
	AreaMaterials         []SystemAreaMaterial       `bson:"areaMaterials" json:"areaMaterials"`
	TermCondition         []string                   `bson:"termCondition" json:"termCondition"`
	Discounts             []SystemDiscount           `bson:"discounts" json:"discounts"`
	AdditionalCharges     []SystemAdditionalCharge   `bson:"additionalCharges" json:"additionalCharges"`
	TotalCharge           float64                    `bson:"totalCharge" json:"totalCharge"`
	TotalDiscount         float64                    `bson:"totalDiscount" json:"totalDiscount"`
	TotalAdditionalCharge float64                    `bson:"totalAdditionalCharge" json:"totalAdditionalCharge"`
	TotalNettCharge       float64                    `bson:"totalNettCharge" json:"totalNettCharge"`
	Media                 []QuotationMedia           `bson:"media" json:"media"`
	MaterialSummary       []ProjectMaterialSummary   `bson:"materialSummary" json:"materialSummary"` // Procurement tracking
	EstimatedCompleteAt   time.Time                  `bson:"estimatedCompleteAt" json:"estimatedCompleteAt"`
	ActionLogs            []SystemActionLog          `bson:"actionLogs" json:"actionLogs"`
	Pic                   []primitive.ObjectID       `bson:"pic" json:"pic"`
	Company               *primitive.ObjectID        `bson:"company" json:"company"`
	IsDeleted             bool                       `bson:"isDeleted" json:"isDeleted"`
	CreatedAt             time.Time                  `bson:"createdAt" json:"createdAt"`
	CreatedBy             primitive.ObjectID         `bson:"createdBy" json:"createdBy"`
	UpdatedAt             time.Time                  `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy             *primitive.ObjectID        `bson:"updatedBy" json:"updatedBy"`
}
