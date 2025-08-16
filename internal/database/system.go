package database

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/enum"
	"time"
)

type SystemMedia struct {
	Path        string         `bson:"path" json:"path"`
	Description string         `bson:"description" json:"description"`
	Type        enum.MediaType `bson:"type" json:"type"`
}

type SystemAddress struct {
	Street   string `bson:"street" json:"street"`
	State    string `bson:"state" json:"state"`
	Postcode string `bson:"postcode" json:"postcode"`
}

type SystemArea struct {
	Name        string  `bson:"name" json:"name"`
	Description string  `bson:"description" json:"description"`
	Type        string  `bson:"type" json:"type"`
	Length      float64 `bson:"length" json:"length"`
	Width       float64 `bson:"width" json:"width"`
	Height      float64 `bson:"height" json:"height"`
}

type SystemAreaMaterial struct {
	Area      SystemArea                 `bson:"area" json:"area"`
	Materials []SystemAreaMaterialDetail `bson:"materials" json:"materials"`
	Status    string                     `bson:"status" json:"status"`
	Index     int                        `bson:"index" json:"index"`
}

type SystemAreaMaterialDetail struct {
	Material     *primitive.ObjectID `bson:"material" json:"material"`
	Name         string              `bson:"name" json:"name"`
	Type         enum.MaterialType   `bson:"type" json:"type"`
	Supplier     *primitive.ObjectID `bson:"supplier" json:"supplier"`
	Brand        string              `bson:"brand" json:"brand"`
	Unit         string              `bson:"unit" json:"unit"`
	CostPerUnit  float64             `bson:"costPerUnit" json:"costPerUnit"`
	PricePerUnit float64             `bson:"pricePerUnit" json:"pricePerUnit"`
	Quantity     float64             `bson:"quantity" json:"quantity"`
	TotalCost    float64             `bson:"totalCost" json:"totalCost"`
	TotalPrice   float64             `bson:"totalPrice" json:"totalPrice"`
	Remark       string              `bson:"remark" json:"remark"`
	Status       enum.MaterialStatus `bson:"status" json:"status"`
}

type SystemDiscount struct {
	Amount      float64           `bson:"amount" json:"amount"`
	Rate        float64           `bson:"rate" json:"rate"`
	Type        enum.DiscountType `bson:"type" json:"type"`
	Description string            `bson:"description" json:"description"`
}

type SystemActionLog struct {
	Name        string              `bson:"name" json:"name"`
	Description string              `bson:"description" json:"description"`
	Time        time.Time           `bson:"time" json:"time"`
	ByName      string              `bson:"byName" json:"byName"`
	ById        *primitive.ObjectID `bson:"byId" json:"byId"`
}
