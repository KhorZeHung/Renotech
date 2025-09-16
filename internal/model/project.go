package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/database"
)

// Project CRUD request/response models
type ProjectCreateRequest struct {
	QuotationID         primitive.ObjectID   `json:"quotationId" binding:"required"`
	PIC                 []primitive.ObjectID `json:"pic" binding:"required,min=1"`
	EstimatedCompleteAt time.Time            `json:"estimatedCompleteAt" binding:"required"`
}

type ProjectUpdateRequest struct {
	ID                  primitive.ObjectID                `json:"_id" binding:"required"`
	Description         string                            `json:"description"`
	Remark              string                            `json:"remark"`
	AreaMaterials       []database.SystemAreaMaterial    `json:"areaMaterials"`
	Discount            database.SystemDiscount          `json:"discount"`
	Quotation           primitive.ObjectID               `json:"quotation"`
	PIC                 []primitive.ObjectID             `json:"pic" binding:"required,min=1"`
	EstimatedCompleteAt time.Time                        `json:"estimatedCompleteAt" binding:"required"`
}

type ProjectListRequest struct {
	Page        int                 `json:"page"`
	Limit       int                 `json:"limit"`
	Sort        bson.M              `json:"sort"`
	Search      string              `json:"search"`
	Description string              `json:"description"`
	Folder      *primitive.ObjectID `json:"folder"`
	IsStared    *bool               `json:"isStared"`
}

type ProjectListResponse struct {
	Data       []bson.M `json:"data"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"totalPages"`
}

type ProjectToggleStarRequest struct {
	IsStared bool `json:"isStared"`
}