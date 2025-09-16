package enum

type MaterialType string
type MediaType string
type MaterialStatus string
type DiscountType string
type UserType string
type ErrorCode string
type OrderStatus string
type OrderPriority string

const (
	ErrorCodeValidation   ErrorCode = "VALIDATION_ERROR"
	ErrorCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrorCodeInternal     ErrorCode = "INTERNAL_ERROR"
	ErrorCodeBadRequest   ErrorCode = "BAD_REQUEST"
	ErrorCodeTooLarge     ErrorCode = "FILE_TOO_LARGE"
)
const (
	MaterialTypeProduct  MaterialType = "product"
	MaterialTypeService  MaterialType = "service"
	MaterialTypeTemplate MaterialType = "template"
)

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeDocument MediaType = "document"
)

const (
	MaterialStatusActive      MaterialStatus = "active"
	MaterialStatusInactive    MaterialStatus = "inactive"
	MaterialStatusDiscontinue MaterialStatus = "discontinue"
	MaterialStatusRecommend   MaterialStatus = "recommend"
)

const (
	DiscountTypeRate   DiscountType = "rate"
	DiscountTypeAmount DiscountType = "amount"
)

const (
	UserTypeTenant      UserType = "tenant"
	UserTypeSystemAdmin UserType = "system_admin"
)

const (
	OrderStatusDraft     OrderStatus = "draft"
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusSent      OrderStatus = "sent"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusRejected  OrderStatus = "rejected"
)

const (
	OrderPriorityLow    OrderPriority = "low"
	OrderPriorityMedium OrderPriority = "medium"
	OrderPriorityHigh   OrderPriority = "high"
	OrderPriorityUrgent OrderPriority = "urgent"
)
