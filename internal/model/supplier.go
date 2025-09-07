package model

import "go.mongodb.org/mongo-driver/bson"

type SupplierListRequest struct {
	Page             int    `json:"page"`
	Limit            int    `json:"limit"`
	Sort             bson.M `json:"sort"`
	Search           string `json:"search"`
	Label            string `json:"label"`
	Name             string `json:"name"`
	Contact          string `json:"contact"`
	Email            string `json:"email"`
	Tags             string `json:"tags"`
	Description      string `json:"description"`
}

type SupplierListResponse struct {
	Data       []bson.M `json:"data"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"totalPages"`
}