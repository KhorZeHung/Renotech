package model

import "go.mongodb.org/mongo-driver/bson"

type CompanyListRequest struct {
	Page                int    `json:"page"`
	Limit               int    `json:"limit"`
	Sort                bson.M `json:"sort"`
	Search              string `json:"search"`
	Name                string `json:"name"`
	ClientDisplayName   string `json:"clientDisplayName"`
	SupplierDisplayName string `json:"supplierDisplayName"`
	Address             string `json:"address"`
	Website             string `json:"website"`
	Owner               string `json:"owner"`
	Contact             string `json:"contact"`
	IsEnabled           *bool  `json:"isEnabled"`
}

type CompanyListResponse struct {
	Data       []bson.M `json:"data"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"totalPages"`
}
