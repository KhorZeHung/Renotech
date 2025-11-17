package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/database"
)

// Project conversion request
type ProjectConvertRequest struct {
	QuotationID         primitive.ObjectID   `json:"quotationId" binding:"required"`
	PIC                 []primitive.ObjectID `json:"pic" binding:"required,min=1"`
	EstimatedCompleteAt time.Time            `json:"estimatedCompleteAt" binding:"required"`
}

// Project update request
type ProjectUpdateRequest struct {
	ID                    *primitive.ObjectID              `json:"_id" binding:"required"`
	Name                  string                           `json:"name"`
	Client                database.SystemClient            `json:"client"`
	Budget                float64                          `json:"budget"`
	Address               database.SystemAddress           `json:"address"`
	Description           string                           `json:"description"`
	Remark                string                           `json:"remark"`
	AreaMaterials         []database.SystemAreaMaterial    `json:"areaMaterials"`
	TermCondition         []string                         `json:"termCondition"`
	Discounts             []database.SystemDiscount        `json:"discounts"`
	AdditionalCharges     []database.SystemAdditionalCharge `json:"additionalCharges"`
	Media                 []database.QuotationMedia        `json:"media"`
	EstimatedCompleteAt   time.Time                        `json:"estimatedCompleteAt" binding:"required"`
	Pic                   []primitive.ObjectID             `json:"pic" binding:"required,min=1"`
}

// Project list request
type ProjectListRequest struct {
	Page        int    `json:"page"`
	Limit       int    `json:"limit"`
	Sort        bson.M `json:"sort"`
	Search      string `json:"search"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Project list response
type ProjectListResponse struct {
	Data       []bson.M `json:"data"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"totalPages"`
}