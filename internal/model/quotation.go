package model

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Quotation CRUD request/response models
type QuotationListRequest struct {
	Page        int                 `json:"page"`
	Limit       int                 `json:"limit"`
	Sort        bson.M              `json:"sort"`
	Search      string              `json:"search"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Folder      *primitive.ObjectID `json:"folder"`
	IsStared    *bool               `json:"isStared"`
}

type QuotationListResponse struct {
	Data       []bson.M `json:"data"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"totalPages"`
}

type QuotationToggleStarRequest struct {
	IsStared bool `json:"isStared"`
}

type QuotationCreateFolderRequest struct {
	ID   primitive.ObjectID `json:"_id"`
	Name string             `json:"name"`
}

type QuotationMoveRequest struct {
	ID     primitive.ObjectID  `json:"_id"`
	Folder *primitive.ObjectID `json:"folder"`
}
