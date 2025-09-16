package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/enum"
)

type Order struct {
	ID      *primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Project *primitive.ObjectID `bson:"project" json:"project"` // Optional: Original project reference
	Company *primitive.ObjectID `bson:"company" json:"company"` // Tenant isolation

	// Embedded Supplier Information (flexible - can be from system or external)
	Supplier OrderSupplier `bson:"supplier" json:"supplier"`

	// PO Basic Information
	PONumber         string    `bson:"poNumber" json:"poNumber"` // Auto-generated unique PO number
	OrderDate        time.Time `bson:"orderDate" json:"orderDate"`
	ExpectedDelivery time.Time `bson:"expectedDelivery" json:"expectedDelivery"`

	// Delivery Information
	DeliveryAddress SystemAddress `bson:"deliveryAddress" json:"deliveryAddress"`
	DeliveryContact string        `bson:"deliveryContact" json:"deliveryContact"`
	DeliveryPhone   string        `bson:"deliveryPhone" json:"deliveryPhone"`
	DeliveryRemark  string        `bson:"deliveryRemark" json:"deliveryRemark"`

	// Terms & Conditions
	TermConditions []string `bson:"termConditions" json:"termConditions"` // Array of terms

	// Order Items & Pricing
	Items       []OrderItem `bson:"items" json:"items"`
	SubTotal    float64     `bson:"subTotal" json:"subTotal"`
	TaxRate     float64     `bson:"taxRate" json:"taxRate"`         // Tax percentage
	TaxAmount   float64     `bson:"taxAmount" json:"taxAmount"`     // Calculated tax
	TotalCharge float64     `bson:"totalCharge" json:"totalCharge"` // Final total

	// Status & Tracking
	Status   enum.OrderStatus   `bson:"status" json:"status"`     // Draft, Sent, Confirmed, Delivered, etc.
	Priority enum.OrderPriority `bson:"priority" json:"priority"` // Low, Medium, High, Urgent

	// Additional Information
	Remark        string `bson:"remark" json:"remark"`
	InternalNotes string `bson:"internalNotes" json:"internalNotes"` // Not visible in PO

	// Audit Fields
	ActionLogs []SystemActionLog   `bson:"actionLogs" json:"actionLogs"`
	CreatedAt  time.Time           `bson:"createdAt" json:"createdAt"`
	CreatedBy  primitive.ObjectID  `bson:"createdBy" json:"createdBy"`
	UpdatedAt  time.Time           `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy  *primitive.ObjectID `bson:"updatedBy" json:"updatedBy"`
	IsDeleted  bool                `bson:"isDeleted" json:"isDeleted"`
}

type OrderSupplier struct {
	ID      *primitive.ObjectID `bson:"_id" json:"_id"`   // System supplier ID (nullable)
	Name    string              `bson:"name" json:"name"` // Required: supplier name
	Contact string              `bson:"contact" json:"contact"`
	Email   string              `bson:"email" json:"email"`
	Logo    string              `bson:"logo" json:"logo"`
	Address SystemAddress       `bson:"address" json:"address"`
}

type OrderItem struct {
	Material    *primitive.ObjectID `bson:"material" json:"material"` // Reference to material (optional)
	Name        string              `bson:"name" json:"name"`
	Description string              `bson:"description" json:"description"`
	Brand       string              `bson:"brand" json:"brand"`
	Unit        string              `bson:"unit" json:"unit"` // pcs, kg, m², etc.
	Quantity    float64             `bson:"quantity" json:"quantity"`
	UnitPrice   float64             `bson:"unitPrice" json:"unitPrice"`   // Cost per unit
	TotalPrice  float64             `bson:"totalPrice" json:"totalPrice"` // quantity × unitPrice
	Remark      string              `bson:"remark" json:"remark"`
}
