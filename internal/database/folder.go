package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Folder struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty" json:"_id,omitempty"`
	Name        string              `bson:"name" json:"name"`
	Client      SystemClient        `bson:"client" json:"client"`
	Budget      float64             `bson:"budget" json:"budget"`
	Address     SystemAddress       `bson:"address" json:"address"`
	Description string              `bson:"description" json:"description"`
	Remark      string              `bson:"remark" json:"remark"`
	Status      string              `bson:"status" json:"status"`
	Media       []SystemMedia       `bson:"media" json:"media"`
	Areas       []SystemArea        `bson:"areas" json:"areas"`
	Company     *primitive.ObjectID `bson:"company" json:"company"`
	CreatedAt   time.Time           `bson:"createdAt" json:"createdAt"`
	CreatedBy   primitive.ObjectID  `bson:"createdBy" json:"createdBy"`
	UpdatedAt   time.Time           `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy   *primitive.ObjectID `bson:"updatedBy" json:"updatedBy"`
	IsDeleted   bool                `bson:"isDeleted" json:"isDeleted"`
}
