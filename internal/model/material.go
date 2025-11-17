package model

import "go.mongodb.org/mongo-driver/bson"

type MaterialListRequest struct {
	Page         int    `json:"page"`
	Limit        int    `json:"limit"`
	Sort         bson.M `json:"sort"`
	Search       string `json:"search"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Brand        string `json:"brand"`
	Unit         string `json:"unit"`
	Status       string `json:"status"`
	CostPerUnit  bson.M `json:"costPerUnit"`
	PricePerUnit bson.M `json:"pricePerUnit"`
}

type MaterialListResponse struct {
	Data       []bson.M `json:"data"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"totalPages"`
}
