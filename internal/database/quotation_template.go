package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type QuotationTemplate struct {
	ID               primitive.ObjectID  `bson:"_id,omitempty" json:"_id,omitempty"`
	Name             string              `bson:"name" json:"name"`
	CreatedAt        time.Time           `bson:"createdAt" json:"createdAt"`
	CreatedBy        primitive.ObjectID  `bson:"createdBy" json:"createdBy"`
	UpdatedAt        time.Time           `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy        primitive.ObjectID  `bson:"updatedBy" json:"updatedBy"`
	IsDeleted        bool                `bson:"isDeleted" json:"isDeleted"`
	IsEnabled        bool                `bson:"isEnabled" json:"isEnabled"`
	MainHTMLContent  string              `bson:"mainHTMLContent" json:"mainHTMLContent"`
	CSSContent       string              `bson:"cssContent" json:"cssContent"`
	AreaHTMLContent  string              `bson:"areaHTMLContent" json:"areaHTMLContent"`
	VariableList     []string            `bson:"variableList" json:"variableList"`
	DefaultFileName  string              `bson:"defaultFileName" json:"defaultFileName"`
	Company          *primitive.ObjectID `bson:"company,omitempty" json:"company,omitempty"`
}