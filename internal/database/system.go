package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/enum"
)

type SystemMedia struct {
	Path        string `bson:"path" json:"path"`
	Description string `bson:"description" json:"description"`
}

type SystemAddress struct {
	Line1    string `bson:"line1" json:"line1"`
	Line2    string `bson:"line2" json:"line2"`
	Line3    string `bson:"line3" json:"line3"`
	City     string `bson:"city" json:"city"`
	State    string `bson:"state" json:"state"`
	Postcode string `bson:"postcode" json:"postcode"`
}

type SystemArea struct {
	Name        string  `bson:"name" json:"name"`
	Description string  `bson:"description" json:"description"`
	Unit        string  `bson:"unit" json:"unit"`
	Length      float64 `bson:"length" json:"length"`
	Width       float64 `bson:"width" json:"width"`
	Height      float64 `bson:"height" json:"height"`
}

type SystemAreaMaterial struct {
	Area      SystemArea                 `bson:"area" json:"area"`
	Materials []SystemAreaMaterialDetail `bson:"materials" json:"materials"`
	Status    string                     `bson:"status" json:"status"`
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
}

type SystemDiscount struct {
	Value       float64           `bson:"value" json:"value"`
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
