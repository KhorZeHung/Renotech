package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Company struct {
	ID              *primitive.ObjectID     `bson:"_id,omitempty" json:"_id,omitempty"`
	Name            string                  `bson:"name" json:"name"`
	Address         SystemAddress           `bson:"address" json:"address"`
	Website         string                  `bson:"website" json:"website"`
	Email           string                  `bson:"email" json:"email"`
	RegistrationNo  string                  `bson:"registrationNo" json:"registrationNo"`
	Owner           *primitive.ObjectID     `bson:"owner,omitempty" json:"owner,omitempty"`
	Logo            string                  `bson:"logo" json:"logo"`
	Contact         string                  `bson:"contact" json:"contact"`
	QuotationConfig *CompanyQuotationConfig `bson:"quotationConfig" json:"quotationConfig"`
	OrderConfig     *CompanyOrderConfig     `bson:"orderConfig" json:"orderConfig"`
	IsEnabled       bool                    `bson:"isEnabled" json:"isEnabled"`
	IsDeleted       bool                    `bson:"isDeleted" json:"isDeleted"`
	CreatedAt       time.Time               `bson:"createdAt" json:"createdAt"`
	CreatedBy       *primitive.ObjectID     `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	UpdatedAt       time.Time               `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy       *primitive.ObjectID     `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
}

type CompanyQuotationConfig struct {
	Description   string   `bson:"description" json:"description"`
	TermCondition []string `bson:"termCondition" json:"termCondition"`
}

type CompanyOrderConfig struct {
	Description   string   `bson:"description" json:"description"`
	TermCondition []string `bson:"termCondition" json:"termCondition"`
}
