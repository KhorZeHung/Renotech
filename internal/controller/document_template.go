package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/model"
	"renotech.com.my/internal/service"
	"renotech.com.my/internal/utils"
)

func documentTemplateCreateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Document template creation started", zap.String("endpoint", "/api/v1/document-template"))
	defer systemContext.Logger.Info("Document template creation completed")

	var input model.DocumentTemplateCreateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.DocumentTemplateCreate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Document template creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Document template creation successful",
		zap.String("templateID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func documentTemplateGetHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	templateID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.DocumentTemplateGetByID(templateID, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func documentTemplateListHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	page, limit, err := utils.ValidateListParameters(c.Query("page"), c.Query("limit"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	results, total, err := service.DocumentTemplateList(page, limit, nil, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, results, total)
}

func documentTemplateUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Document template update started", zap.String("endpoint", "/api/v1/document-template/:id"))
	defer systemContext.Logger.Info("Document template update completed")

	templateID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	var input model.DocumentTemplateUpdateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.DocumentTemplateUpdate(templateID, &input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Document template update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Document template update successful",
		zap.String("templateID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func documentTemplateDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Document template deletion started", zap.String("endpoint", "/api/v1/document-template/:id"))
	defer systemContext.Logger.Info("Document template deletion completed")

	templateID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	err = service.DocumentTemplateDelete(templateID, systemContext)
	if err != nil {
		systemContext.Logger.Error("Document template deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Document template deletion successful",
		zap.String("templateID", templateID.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "Document template deleted successfully")
}

func documentTemplatePreviewHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	templateType := c.Param("type")
	if templateType == "" {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Template type is required",
			nil,
		))
		return
	}

	var data bson.M
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// TODO: Get companyID from authenticated user context
	var companyID *primitive.ObjectID = nil

	html, err := service.DocumentTemplatePreview(templateType, data, companyID, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func documentTemplateGenerateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Document template PDF generation started", zap.String("endpoint", "/api/v1/document-template/:type/generate"))
	defer systemContext.Logger.Info("Document template PDF generation completed")

	templateType := c.Param("type")
	if templateType == "" {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Template type is required",
			nil,
		))
		return
	}

	var data bson.M
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// TODO: Get companyID from authenticated user context
	var companyID *primitive.ObjectID = nil

	pdfBuffer, filename, err := service.DocumentTemplateGenerate(templateType, data, companyID, systemContext)
	if err != nil {
		systemContext.Logger.Error("Document template PDF generation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Document template PDF generation successful",
		zap.String("templateType", templateType),
		zap.String("filename", filename),
		zap.Int("pdfSize", len(pdfBuffer)),
	)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Length", strconv.Itoa(len(pdfBuffer)))

	c.Data(http.StatusOK, "application/pdf", pdfBuffer)
}

func DocumentTemplateAPIInit(r *gin.Engine) {
	r.POST("/api/v1/document-template", documentTemplateCreateHandler)
	r.GET("/api/v1/document-template/:id", documentTemplateGetHandler)
	r.GET("/api/v1/document-template", documentTemplateListHandler)
	r.PUT("/api/v1/document-template/:id", documentTemplateUpdateHandler)
	r.DELETE("/api/v1/document-template/:id", documentTemplateDeleteHandler)
	r.POST("/api/v1/document-template/:type/preview", documentTemplatePreviewHandler)
	r.POST("/api/v1/document-template/:type/generate", documentTemplateGenerateHandler)
}
