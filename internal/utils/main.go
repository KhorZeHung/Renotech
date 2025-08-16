package utils

import (
	"github.com/gin-gonic/gin"
	"path/filepath"
	"renotech.com.my/internal/model"
	"renotech.com.my/logs"
	"strings"
)

func SendErrorResponse(c *gin.Context, errMsg interface{}) {
	type ResponseError struct {
		Status string      `json:"status"`
		ErrObj interface{} `json:"errObj"`
	}

	errResponse := ResponseError{
		Status: "failed",
		ErrObj: errMsg,
	}

	c.JSON(400, errResponse)
}

func SendSuccessResponse(c *gin.Context, data interface{}, count ...interface{}) {

	if len(count) == 1 {
		if _, oK := count[0].(int64); oK {
			type ResponseSuccess struct {
				Status string      `json:"status"`
				Total  interface{} `json:"total"`
				Data   interface{} `json:"data"`
			}

			successResponse := ResponseSuccess{
				Status: "Success",
				Total:  count[0],
				Data:   data,
			}
			c.JSON(200, successResponse)
			return
		} else if val, valid := count[0].(*string); valid && val != nil {
			type ResponseSuccess struct {
				Status  string      `json:"status"`
				Message interface{} `json:"message"`
				Data    interface{} `json:"data"`
			}

			successResponse := ResponseSuccess{
				Status:  "Success",
				Message: count[0],
				Data:    data,
			}

			c.JSON(200, successResponse)
			return
		}
	}

	type ResponseSuccess struct {
		Status string      `json:"status"`
		Data   interface{} `json:"data"`
	}

	successResponse := ResponseSuccess{
		Status: "Success",
		Data:   data,
	}

	c.JSON(200, successResponse)
}

func RemoveExtension(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

func SystemContextBaseInit() *model.SystemContext {
	logger := logs.LoggerGet()
	mongoDB := MongoGet()

	return &model.SystemContext{
		Logger:  logger,
		MongoDB: mongoDB,
	}
}

func ConvertToFullPath(input string) string {
	projectRoot, _ := filepath.Abs(filepath.Join())
	fullPath := filepath.Join(projectRoot, input)
	return fullPath
}
