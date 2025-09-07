package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type QuotationTemplateResponse struct {
	ID              primitive.ObjectID  `json:"_id"`
	Name            string              `json:"name"`
	CreatedAt       time.Time           `json:"createdAt"`
	CreatedBy       primitive.ObjectID  `json:"createdBy"`
	UpdatedAt       time.Time           `json:"updatedAt"`
	UpdatedBy       primitive.ObjectID  `json:"updatedBy"`
	IsEnabled       bool                `json:"isEnabled"`
	MainHTMLContent string              `json:"mainHTMLContent"`
	CSSContent      string              `json:"cssContent"`
	AreaHTMLContent string              `json:"areaHTMLContent"`
	VariableList    []string            `json:"variableList"`
	DefaultFileName string              `json:"defaultFileName"`
	Company         *primitive.ObjectID `json:"company"`
}

type QuotationTemplateCreateRequest struct {
	Name            string              `json:"name"`
	MainHTMLContent string              `json:"mainHTMLContent"`
	CSSContent      string              `json:"cssContent"`
	AreaHTMLContent string              `json:"areaHTMLContent"`
	DefaultFileName string              `json:"defaultFileName"`
	Company         *primitive.ObjectID `json:"company"`
}

type QuotationTemplateUpdateRequest struct {
	Name            string              `json:"name"`
	MainHTMLContent string              `json:"mainHTMLContent"`
	CSSContent      string              `json:"cssContent"`
	AreaHTMLContent string              `json:"areaHTMLContent"`
	DefaultFileName string              `json:"defaultFileName"`
	IsEnabled       bool                `json:"isEnabled"`
	Company         *primitive.ObjectID `json:"company"`
}

type QuotationTemplatePreviewRequest struct {
	TemplateID     primitive.ObjectID             `json:"templateId"`
	Variables      map[string]interface{}         `json:"variables"`
	Areas          []QuotationTemplatePreviewArea `json:"areas"`
	TermConditions []string                       `json:"termConditions"`
}

type QuotationTemplatePreviewArea struct {
	AreaNameTitle     string                             `json:"areaNameTitle"`
	AreaName          string                             `json:"areaName"`
	AreaDetail        string                             `json:"areaDetail"`
	AreaItems         []QuotationTemplatePreviewAreaItem `json:"areaItems"`
	AreaSubTotalTitle string                             `json:"areaSubTotalTitle"`
	AreaSubTotal      string                             `json:"areaSubTotal"`
}

type QuotationTemplatePreviewAreaItem struct {
	ItemNo          string `json:"itemNo"`
	ItemName        string `json:"itemName"`
	ItemDescription string `json:"itemDescription"`
	ItemQuantity    string `json:"itemQuantity"`
	ItemUnit        string `json:"itemUnit"`
	ItemUnitPrice   string `json:"itemUnitPrice"`
	ItemTotalPrince string `json:"itemTotalPrince"`
}