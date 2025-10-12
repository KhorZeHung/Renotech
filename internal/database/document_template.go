package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DocumentTemplate struct {
	ID              *primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name            string              `bson:"name" json:"name"`
	Description     string              `bson:"description" json:"description"`
	Html            string              `bson:"html" json:"html"`
	VariableHtml    map[string]string   `bson:"variableHtml" json:"variableHtml"`
	EmbeddedHtml    map[string]string   `bson:"embeddedHtml" json:"embeddedHtml"`
	DefaultFileName string              `bson:"defaultFileName" json:"defaultFileName"`
	IsDefault       bool                `bson:"isDefault" json:"isDefault"`
	Type            string              `bson:"type" json:"type"`
	Company         *primitive.ObjectID `bson:"company" json:"company"`
	CreatedAt       time.Time           `bson:"createdAt" json:"createdAt"`
	CreatedBy       primitive.ObjectID  `bson:"createdBy" json:"createdBy"`
	UpdatedAt       time.Time           `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy       *primitive.ObjectID `bson:"updatedBy" json:"updatedBy"`
	IsDeleted       bool                `bson:"isDeleted" json:"isDeleted"`
}
