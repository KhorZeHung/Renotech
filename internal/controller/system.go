package controller

import (
	"github.com/gin-gonic/gin"
	"renotech.com.my/internal/utils"
)

func pingHandler(c *gin.Context) {
	utils.SendSuccessResponse(c, "hello world")
}

func SystemAPIInit(r *gin.Engine) {
	r.GET("", pingHandler)
}
