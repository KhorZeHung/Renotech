package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/model"
	"renotech.com.my/internal/service"
	"renotech.com.my/internal/utils"
)

func quotationTemplateCreateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Quotation template creation started", zap.String("endpoint", "/api/v1/quotation-template"))
	defer systemContext.Logger.Info("Quotation template creation completed")

	var input model.QuotationTemplateCreateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.QuotationTemplateCreate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Quotation template creation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Quotation template creation successful",
		zap.String("templateID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func quotationTemplateGetHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	templateID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	result, err := service.QuotationTemplateGetByID(templateID, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, result)
}

func quotationTemplateListHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	page, limit, err := utils.ValidateListParameters(c.Query("page"), c.Query("limit"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	results, total, err := service.QuotationTemplateList(page, limit, nil, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	utils.SendSuccessResponse(c, results, total)
}

func quotationTemplateUpdateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Quotation template update started", zap.String("endpoint", "/api/v1/quotation-template/:id"))
	defer systemContext.Logger.Info("Quotation template update completed")

	templateID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	var input model.QuotationTemplateUpdateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	result, err := service.QuotationTemplateUpdate(templateID, &input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Quotation template update failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Quotation template update successful",
		zap.String("templateID", result.ID.Hex()),
		zap.String("name", result.Name),
	)

	utils.SendSuccessResponse(c, result)
}

func quotationTemplateDeleteHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Quotation template deletion started", zap.String("endpoint", "/api/v1/quotation-template/:id"))
	defer systemContext.Logger.Info("Quotation template deletion completed")

	templateID, err := utils.ValidateObjectID(c.Param("id"))
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	err = service.QuotationTemplateDelete(templateID, systemContext)
	if err != nil {
		systemContext.Logger.Error("Quotation template deletion failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Quotation template deletion successful",
		zap.String("templateID", templateID.Hex()),
	)

	utils.SendSuccessMessageResponse(c, "Quotation template deleted successfully")
}

func quotationTemplatePreviewHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)

	var input model.QuotationTemplatePreviewRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	html, err := service.QuotationTemplatePreview(&input, systemContext)
	if err != nil {
		utils.SendErrorResponse(c, err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func quotationTemplateGenerateHandler(c *gin.Context) {
	systemContext := utils.GetSystemContextFromGin(c)
	systemContext.Logger.Info("Quotation template PDF generation started", zap.String("endpoint", "/api/v1/quotation-template/generate"))
	defer systemContext.Logger.Info("Quotation template PDF generation completed")

	var input model.QuotationTemplatePreviewRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.SendErrorResponse(c, utils.SystemError(
			enum.ErrorCodeValidation,
			"Invalid request data",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	pdfBuffer, filename, err := service.QuotationTemplateGenerate(&input, systemContext)
	if err != nil {
		systemContext.Logger.Error("Quotation template PDF generation failed", zap.Error(err))
		utils.SendErrorResponse(c, err)
		return
	}

	systemContext.Logger.Info("Quotation template PDF generation successful",
		zap.String("templateID", input.TemplateID.Hex()),
		zap.String("filename", filename),
		zap.Int("pdfSize", len(pdfBuffer)),
	)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Length", strconv.Itoa(len(pdfBuffer)))

	c.Data(http.StatusOK, "application/pdf", pdfBuffer)
}

func QuotationTemplateAPIInit(r *gin.Engine) {
	r.POST("/api/v1/quotation-template", quotationTemplateCreateHandler)
	r.GET("/api/v1/quotation-template/:id", quotationTemplateGetHandler)
	r.GET("/api/v1/quotation-template", quotationTemplateListHandler)
	r.PUT("/api/v1/quotation-template/:id", quotationTemplateUpdateHandler)
	r.DELETE("/api/v1/quotation-template/:id", quotationTemplateDeleteHandler)
	r.POST("/api/v1/quotation-template/preview", quotationTemplatePreviewHandler)
	r.POST("/api/v1/quotation-template/generate", quotationTemplateGenerateHandler)
}
