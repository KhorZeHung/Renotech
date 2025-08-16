package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"renotech.com.my/internal/controller"
	"renotech.com.my/internal/utils"
	"renotech.com.my/logs"
)

func init() {
	// init logger
	logs.LoggingSetup()

	// init mongo db
	err := utils.MongoInit()

	if err != nil {
		fmt.Println("Init MongoDB error: ", err)
		panic(err)
	}
}

func main() {
	router := gin.Default()

	// clean up mongo connection
	defer utils.MongoCleanUp()

	controller.MediaAPIInit(router)

	err := router.Run("127.0.0.1:8000")

	if err != nil {
		panic(err)
	}
}
