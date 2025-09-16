package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"renotech.com.my/internal/database"
	"renotech.com.my/internal/enum"
)

// Order CRUD request/response models
type OrderInitRequest struct {
	ProjectID primitive.ObjectID `json:"projectId" binding:"required"`
	// Optional: Allow user to select specific materials/suppliers
	SelectedSuppliers []primitive.ObjectID `json:"selectedSuppliers,omitempty"` // If empty, include all suppliers
}

type OrderCreateRequest struct {
	Project         *primitive.ObjectID        `json:"project,omitempty"`
	Supplier        database.OrderSupplier     `json:"supplier" binding:"required"`
	OrderDate       time.Time                  `json:"orderDate" binding:"required"`
	ExpectedDelivery time.Time                 `json:"expectedDelivery" binding:"required"`
	DeliveryAddress database.SystemAddress     `json:"deliveryAddress"`
	DeliveryContact string                     `json:"deliveryContact"`
	DeliveryPhone   string                     `json:"deliveryPhone"`
	DeliveryRemark  string                     `json:"deliveryRemark"`
	TermConditions  []string                   `json:"termConditions"`
	Items           []database.OrderItem       `json:"items" binding:"required,min=1"`
	TaxRate         float64                    `json:"taxRate"`
	Priority        enum.OrderPriority         `json:"priority"`
	Remark          string                     `json:"remark"`
	InternalNotes   string                     `json:"internalNotes"`
}

type OrderUpdateRequest struct {
	ID              primitive.ObjectID         `json:"_id" binding:"required"`
	Supplier        database.OrderSupplier     `json:"supplier" binding:"required"`
	OrderDate       time.Time                  `json:"orderDate" binding:"required"`
	ExpectedDelivery time.Time                 `json:"expectedDelivery" binding:"required"`
	DeliveryAddress database.SystemAddress     `json:"deliveryAddress"`
	DeliveryContact string                     `json:"deliveryContact"`
	DeliveryPhone   string                     `json:"deliveryPhone"`
	DeliveryRemark  string                     `json:"deliveryRemark"`
	TermConditions  []string                   `json:"termConditions"`
	Items           []database.OrderItem       `json:"items" binding:"required,min=1"`
	TaxRate         float64                    `json:"taxRate"`
	Priority        enum.OrderPriority         `json:"priority"`
	Remark          string                     `json:"remark"`
	InternalNotes   string                     `json:"internalNotes"`
}

type OrderListRequest struct {
	Page            int                  `json:"page"`
	Limit           int                  `json:"limit"`
	Sort            bson.M               `json:"sort"`
	Search          string               `json:"search"`
	Project         *primitive.ObjectID  `json:"project"`
	SupplierID      *primitive.ObjectID  `json:"supplierId"`      // System supplier ID
	SupplierName    string               `json:"supplierName"`    // Supplier name search
	Status          *enum.OrderStatus    `json:"status"`
	Priority        *enum.OrderPriority  `json:"priority"`
	PONumber        string               `json:"poNumber"`
	DateFrom        *time.Time           `json:"dateFrom"`        // Order date range
	DateTo          *time.Time           `json:"dateTo"`
}

type OrderListResponse struct {
	Data       []bson.M `json:"data"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	Total      int64    `json:"total"`
	TotalPages int      `json:"totalPages"`
}

type OrderStatusUpdateRequest struct {
	Status enum.OrderStatus `json:"status" binding:"required"`
}

type OrderInitResponse struct {
	Orders []database.Order `json:"orders"`
	Summary struct {
		TotalOrders    int                            `json:"totalOrders"`
		SupplierCount  int                            `json:"supplierCount"`
		TotalValue     float64                        `json:"totalValue"`
		BySupplier     map[string]OrderSupplierSummary `json:"bySupplier"`
	} `json:"summary"`
}

type OrderSupplierSummary struct {
	SupplierName string  `json:"supplierName"`
	ItemCount    int     `json:"itemCount"`
	TotalValue   float64 `json:"totalValue"`
}