package service

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"os"
	"renotech.com.my/internal/database"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/model"
	"strings"
	"time"
)

func MediaCreate(input *database.Media, systemContext *model.SystemContext) (*database.Media, error) {
	collection := systemContext.MongoDB.Collection("media")

	// set default field
	input.CreatedAt = time.Now()
	input.Path = strings.Replace(input.Path, "//", "/", -1)

	result, err := collection.InsertOne(context.Background(), input)

	if err != nil {
		return nil, err
	}

	oid := result.InsertedID.(primitive.ObjectID)

	var doc database.Media

	_ = collection.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&doc)

	return &doc, nil
}

func MediaDelete(input primitive.ObjectID, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("media")

	filter := bson.M{"_id": input}

	var doc database.Media

	_ = collection.FindOne(context.Background(), filter).Decode(&doc)

	if doc.ID == nil {
		return fmt.Errorf("media not found")
	}

	_, err := collection.DeleteOne(context.Background(), filter)

	if err != nil {
		return err
	}

	err = os.Remove(doc.Path)

	if err != nil {
		return err
	}

	return nil
}

func mediaAssignType(ext string) enum.MediaType {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp", ".svg", ".ico":
		return enum.MediaTypeImage
	case ".mp4", ".mov", ".avi", ".mkv", ".wmv", ".flv", ".webm", ".mpeg", ".mpg", ".m4v":
		return enum.MediaTypeVideo
	case ".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a", ".wma", ".aiff", ".opus":
		return enum.MediaTypeAudio
	case ".pdf", ".doc", ".docx", ".txt", ".rtf", ".odt", ".xls", ".xlsx", ".ppt", ".pptx", ".csv", ".md", ".html":
		return enum.MediaTypeDocument
	default:
		return enum.MediaTypeDocument
	}
}
