package model

import "go.mongodb.org/mongo-driver/bson"

type MediaListRequest struct {
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	Sort       bson.M `json:"sort"`
	Search     string `json:"search"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Module     string `json:"module"`
	Extension  string `json:"extension"`
	FileName   string `json:"fileName"`
}

type MediaListResponse struct {
	Data       []bson.M `json:"data"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"totalPages"`
}