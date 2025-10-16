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
	Postcode string `bson:"postcode" json:"postcode"`
	City     string `bson:"city" json:"city"`
	State    string `bson:"state" json:"state"`
}

type SystemArea struct {
	Name        string `bson:"name" json:"name"`
	Description string `bson:"description" json:"description"`
}

type SystemAreaMaterial struct {
	Area      SystemArea                 `bson:"area" json:"area"`
	Materials []SystemAreaMaterialDetail `bson:"materials" json:"materials"`
	SubTotal  float64                    `bson:"subTotal" json:"subTotal"`
}

type SystemAreaMaterialDetail struct {
	Material     *primitive.ObjectID        `bson:"material" json:"material"`
	Template     []SystemAreaMaterialDetail `bson:"template" json:"template"`
	Name         string                     `bson:"name" json:"name"`
	Type         enum.MaterialType          `bson:"type" json:"type"`
	Brand        string                     `bson:"brand" json:"brand"`
	Unit         string                     `bson:"unit" json:"unit"`
	PricePerUnit float64                    `bson:"pricePerUnit" json:"pricePerUnit"`
	Quantity     float64                    `bson:"quantity" json:"quantity"`
	SubTotal     float64                    `bson:"subTotal" json:"subTotal"`
	Remark       string                     `bson:"remark" json:"remark"`
	Description  string                     `bson:"description" json:"description"`
}

type SystemDiscount struct {
	Name        string            `bson:"name" json:"name"`
	Value       float64           `bson:"value" json:"value"`
	Type        enum.DiscountType `bson:"type" json:"type"`
	Description string            `bson:"description" json:"description"`
}

type SystemAdditionalCharge struct {
	Name        string                    `bson:"name" json:"name"`
	Value       float64                   `bson:"value" json:"value"`
	Type        enum.AdditionalChargeType `bson:"type" json:"type"`
	Description string                    `bson:"description" json:"description"`
}

type SystemActionLog struct {
	Description string              `bson:"description" json:"description"`
	Time        time.Time           `bson:"time" json:"time"`
	ByName      string              `bson:"byName" json:"byName"`
	ById        *primitive.ObjectID `bson:"byId" json:"byId"`
}
