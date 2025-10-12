package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DocumentTemplateResponse struct {
	ID              *primitive.ObjectID `json:"_id,omitempty"`
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	Html            string              `json:"html"`
	VariableHtml    map[string]string   `json:"variableHtml"`
	EmbeddedHtml    map[string]string   `json:"embeddedHtml"`
	DefaultFileName string              `json:"defaultFileName"`
	IsDefault       bool                `json:"isDefault"`
	Type            string              `json:"type"`
	Company         *primitive.ObjectID `json:"company,omitempty"`
	CreatedAt       time.Time           `json:"createdAt"`
	CreatedBy       primitive.ObjectID  `json:"createdBy"`
	UpdatedAt       time.Time           `json:"updatedAt"`
	UpdatedBy       *primitive.ObjectID `json:"updatedBy,omitempty"`
	IsDeleted       bool                `json:"isDeleted"`
}

type DocumentTemplateCreateRequest struct {
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	Html            string              `json:"html"`
	VariableHtml    map[string]string   `json:"variableHtml"`
	EmbeddedHtml    map[string]string   `json:"embeddedHtml"`
	DefaultFileName string              `json:"defaultFileName"`
	Type            string              `json:"type"`
	Company         *primitive.ObjectID `json:"company,omitempty"`
}

type DocumentTemplateUpdateRequest struct {
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	Html            string              `json:"html"`
	VariableHtml    map[string]string   `json:"variableHtml"`
	EmbeddedHtml    map[string]string   `json:"embeddedHtml"`
	DefaultFileName string              `json:"defaultFileName"`
	Type            string              `json:"type"`
	Company         *primitive.ObjectID `json:"company,omitempty"`
}
