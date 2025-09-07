package model

import "go.mongodb.org/mongo-driver/bson"

type MaterialListRequest struct {
	Page                int    `json:"page"`
	Limit               int    `json:"limit"`
	Sort                bson.M `json:"sort"`
	Search              string `json:"search"`
	Name                string `json:"name"`
	ClientDisplayName   string `json:"clientDisplayName"`
	SupplierDisplayName string `json:"supplierDisplayName"`
	Type                string `json:"type"`
	Supplier            string `json:"supplier"`
	Brand               string `json:"brand"`
	Unit                string `json:"unit"`
	Status              string `json:"status"`
	CostPerUnit         bson.M `json:"costPerUnit"`
	PricePerUnit        bson.M `json:"pricePerUnit"`
	Categories          string `json:"categories"`
	Tags                string `json:"tags"`
}

type MaterialListResponse struct {
	Data       []bson.M `json:"data"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"totalPages"`
}
