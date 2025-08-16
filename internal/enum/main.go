package enum

type MaterialType string
type MediaType string
type MaterialStatus string
type DiscountType string

const (
	MaterialTypeProduct MaterialType = "product"
	MaterialTypeService MaterialType = "service"
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
