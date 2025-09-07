package model

import (
	"go.mongodb.org/mongo-driver/bson"
)

// Folder CRUD request/response models
type FolderListRequest struct {
	Page           int    `json:"page"`
	Limit          int    `json:"limit"`
	Sort           bson.M `json:"sort"`
	Search         string `json:"search"`
	Name           string `json:"name"`
	ClientName     string `json:"clientName"`
	ClientContact  string `json:"clientContact"`
	ClientEmail    string `json:"clientEmail"`
	ProjectAddress string `json:"projectAddress"`
	Status         string `json:"status"`
}

type FolderListResponse struct {
	Data       []bson.M `json:"data"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"totalPages"`
}