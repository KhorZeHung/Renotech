package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"renotech.com.my/internal/database"
	"renotech.com.my/internal/enum"
	"renotech.com.my/internal/model"
	"renotech.com.my/internal/utils"
)

func QuotationTemplateCreate(input *model.QuotationTemplateCreateRequest, systemContext *model.SystemContext) (*model.QuotationTemplateResponse, error) {
	collection := systemContext.MongoDB.Collection("quotation_templates")

	if strings.TrimSpace(input.Name) == "" {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Name is required", nil)
	}
	if strings.TrimSpace(input.MainHTMLContent) == "" {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Main HTML content is required", nil)
	}
	if strings.TrimSpace(input.DefaultFileName) == "" {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Default filename is required", nil)
	}

	// Auto-extract variables from HTML content
	variableList := quotationTemplateExtractVariablesFromHTML(input.MainHTMLContent)

	defaultUserID := primitive.NewObjectID() // Default system user ID

	template := &database.QuotationTemplate{
		Name:            input.Name,
		CreatedAt:       time.Now(),
		CreatedBy:       defaultUserID,
		UpdatedAt:       time.Now(),
		UpdatedBy:       defaultUserID,
		IsDeleted:       false,
		IsEnabled:       true,
		MainHTMLContent: input.MainHTMLContent,
		CSSContent:      input.CSSContent,
		AreaHTMLContent: input.AreaHTMLContent,
		VariableList:    variableList,
		DefaultFileName: input.DefaultFileName,
		Company:         input.Company,
	}

	result, err := collection.InsertOne(context.Background(), template)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create quotation template", nil)
	}

	templateID := result.InsertedID.(primitive.ObjectID)

	var createdTemplate database.QuotationTemplate
	err = collection.FindOne(context.Background(), bson.M{"_id": templateID}).Decode(&createdTemplate)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve created template", nil)
	}

	response := quotationTemplateMapToResponse(&createdTemplate)
	return &response, nil
}

func QuotationTemplateGetByID(templateID primitive.ObjectID, systemContext *model.SystemContext) (*model.QuotationTemplateResponse, error) {
	collection := systemContext.MongoDB.Collection("quotation_templates")

	var template database.QuotationTemplate
	err := collection.FindOne(context.Background(), bson.M{
		"_id":       templateID,
		"isDeleted": false,
	}).Decode(&template)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Quotation template not found", nil)
	}

	response := quotationTemplateMapToResponse(&template)
	return &response, nil
}

func QuotationTemplateList(page, limit int, companyID *primitive.ObjectID, systemContext *model.SystemContext) ([]model.QuotationTemplateResponse, int64, error) {
	collection := systemContext.MongoDB.Collection("quotation_templates")

	filter := bson.M{"isDeleted": false}
	if companyID != nil {
		filter["company"] = companyID
	}

	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return nil, 0, utils.SystemError(enum.ErrorCodeInternal, "Failed to count templates", nil)
	}

	skip := (page - 1) * limit
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"createdAt": -1})

	cursor, err := collection.Find(context.Background(), filter, findOptions)
	if err != nil {
		return nil, 0, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve templates", nil)
	}
	defer cursor.Close(context.Background())

	var templates []database.QuotationTemplate
	if err = cursor.All(context.Background(), &templates); err != nil {
		return nil, 0, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode templates", nil)
	}

	var responses []model.QuotationTemplateResponse
	for _, template := range templates {
		responses = append(responses, quotationTemplateMapToResponse(&template))
	}

	return responses, total, nil
}

func QuotationTemplateUpdate(templateID primitive.ObjectID, input *model.QuotationTemplateUpdateRequest, systemContext *model.SystemContext) (*model.QuotationTemplateResponse, error) {
	collection := systemContext.MongoDB.Collection("quotation_templates")

	if strings.TrimSpace(input.Name) == "" {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Name is required", nil)
	}
	if strings.TrimSpace(input.MainHTMLContent) == "" {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Main HTML content is required", nil)
	}
	if strings.TrimSpace(input.DefaultFileName) == "" {
		return nil, utils.SystemError(enum.ErrorCodeValidation, "Default filename is required", nil)
	}

	// Auto-extract variables from HTML content
	variableList := quotationTemplateExtractVariablesFromHTML(input.MainHTMLContent)

	defaultUserID := primitive.NewObjectID() // Default system user ID

	update := bson.M{
		"$set": bson.M{
			"name":            input.Name,
			"mainHTMLContent": input.MainHTMLContent,
			"cssContent":      input.CSSContent,
			"areaHTMLContent": input.AreaHTMLContent,
			"variableList":    variableList,
			"defaultFileName": input.DefaultFileName,
			"isEnabled":       input.IsEnabled,
			"company":         input.Company,
			"updatedAt":       time.Now(),
			"updatedBy":       defaultUserID,
		},
	}

	_, err := collection.UpdateOne(context.Background(), bson.M{
		"_id":       templateID,
		"isDeleted": false,
	}, update)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update quotation template", nil)
	}

	return QuotationTemplateGetByID(templateID, systemContext)
}

func QuotationTemplateDelete(templateID primitive.ObjectID, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("quotation_templates")

	defaultUserID := primitive.NewObjectID() // Default system user ID

	update := bson.M{
		"$set": bson.M{
			"isDeleted": true,
			"updatedAt": time.Now(),
			"updatedBy": defaultUserID,
		},
	}

	result, err := collection.UpdateOne(context.Background(), bson.M{
		"_id":       templateID,
		"isDeleted": false,
	}, update)
	if err != nil {
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to delete quotation template", nil)
	}

	if result.MatchedCount == 0 {
		return utils.SystemError(enum.ErrorCodeNotFound, "Quotation template not found", nil)
	}

	return nil
}

func QuotationTemplatePreview(input *model.QuotationTemplatePreviewRequest, systemContext *model.SystemContext) (string, error) {
	template, err := QuotationTemplateGetByID(input.TemplateID, systemContext)
	if err != nil {
		return "", err
	}

	// Check if all required variables are provided
	for _, variable := range template.VariableList {
		if _, exists := input.Variables[variable]; !exists {
			return "", utils.SystemError(enum.ErrorCodeValidation,
				fmt.Sprintf("Variable '%s' is required", variable), nil)
		}
	}

	// Generate dynamic HTML from structured areas
	areaSectionHTML := quotationTemplateGenerateAreaSectionHTML(input.Areas, template.AreaHTMLContent)

	// Generate term conditions HTML
	termConditionHTML := quotationTemplateGenerateTermConditionHTML(input.TermConditions)

	html := template.MainHTMLContent

	// Replace system-generated placeholders with double square brackets
	html = strings.ReplaceAll(html, "[[areaSection]]", areaSectionHTML)
	html = strings.ReplaceAll(html, "[[termConditionSection]]", termConditionHTML)

	// Replace user variables with formatted values
	for variable, value := range input.Variables {
		placeholder := fmt.Sprintf("{{%s}}", variable)
		var processedValue string

		// Check if this is a price-related variable
		if quotationTemplateIsPriceVariable(variable) {
			processedValue = utils.FormatPriceString(value, 2)
		} else {
			processedValue = fmt.Sprintf("%v", value)
		}

		html = strings.ReplaceAll(html, placeholder, processedValue)
	}

	if template.CSSContent != "" {
		html = fmt.Sprintf("<style>%s</style>%s", template.CSSContent, html)
	}

	return html, nil
}

func QuotationTemplateGenerate(input *model.QuotationTemplatePreviewRequest, systemContext *model.SystemContext) ([]byte, string, error) {
	template, err := QuotationTemplateGetByID(input.TemplateID, systemContext)
	if err != nil {
		return nil, "", err
	}

	// Check if all required variables are provided
	for _, variable := range template.VariableList {
		if _, exists := input.Variables[variable]; !exists {
			return nil, "", utils.SystemError(enum.ErrorCodeValidation,
				fmt.Sprintf("Variable '%s' is required", variable), nil)
		}
	}

	// Generate dynamic HTML from structured areas
	areaSectionHTML := quotationTemplateGenerateAreaSectionHTML(input.Areas, template.AreaHTMLContent)

	// Generate term conditions HTML
	termConditionHTML := quotationTemplateGenerateTermConditionHTML(input.TermConditions)

	html := template.MainHTMLContent

	// Replace system-generated placeholders with double square brackets
	html = strings.ReplaceAll(html, "[[areaSection]]", areaSectionHTML)
	html = strings.ReplaceAll(html, "[[termConditionSection]]", termConditionHTML)

	// Replace user variables with formatted values
	for variable, value := range input.Variables {
		placeholder := fmt.Sprintf("{{%s}}", variable)
		var processedValue string

		// Check if this is a price-related variable
		if quotationTemplateIsPriceVariable(variable) {
			processedValue = utils.FormatPriceString(value, 2)
		} else {
			processedValue = fmt.Sprintf("%v", value)
		}

		html = strings.ReplaceAll(html, placeholder, processedValue)
	}

	if template.CSSContent != "" {
		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<style>%s</style>
</head>
<body>%s</body>
</html>`, template.CSSContent, html)
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var pdfBuffer []byte
	err = chromedp.Run(ctx, quotationTemplateHTMLToPDF(html, &pdfBuffer))
	if err != nil {
		return nil, "", utils.SystemError(enum.ErrorCodeInternal, "Failed to generate PDF", nil)
	}

	filename := template.DefaultFileName
	if !strings.HasSuffix(filename, ".pdf") {
		filename += ".pdf"
	}

	return pdfBuffer, filename, nil
}

// non-service

func quotationTemplateGenerateAreaSectionHTML(areas []model.QuotationTemplatePreviewArea, areaHTMLContent string) string {
	var areaHTML strings.Builder

	for _, area := range areas {
		// Replace area-specific variables in the template
		areaSection := areaHTMLContent
		areaSection = strings.ReplaceAll(areaSection, "{{areaNameTitle}}", area.AreaNameTitle)
		areaSection = strings.ReplaceAll(areaSection, "{{areaName}}", area.AreaName)
		areaSection = strings.ReplaceAll(areaSection, "{{areaDetail}}", area.AreaDetail)
		areaSection = strings.ReplaceAll(areaSection, "{{areaSubTotalTitle}}", area.AreaSubTotalTitle)
		areaSection = strings.ReplaceAll(areaSection, "{{areaSubTotal}}", area.AreaSubTotal)

		// Generate items HTML
		var itemsHTML strings.Builder
		for _, item := range area.AreaItems {
			itemRow := fmt.Sprintf(`<tr><td class="item-number">%s</td><td class="description">%s<br />%s</td><td class="qty">%s</td><td class="unit">%s</td><td class="unit-price amount">%s</td><td class="total-price amount">%s</td></tr>`,
				item.ItemNo, item.ItemName, item.ItemDescription, item.ItemQuantity, item.ItemUnit, item.ItemUnitPrice, item.ItemTotalPrince)
			itemsHTML.WriteString(itemRow)
		}

		// Replace the items placeholder
		areaSection = strings.ReplaceAll(areaSection, "{{areaItemRowFunc}}", itemsHTML.String())

		areaHTML.WriteString(areaSection)
	}

	return areaHTML.String()
}

func quotationTemplateGenerateTermConditionHTML(termConditions []string) string {
	if len(termConditions) == 0 {
		return ""
	}

	var termHTML strings.Builder
	termHTML.WriteString(`<ol class="term-conditions">`)

	for _, condition := range termConditions {
		termHTML.WriteString(fmt.Sprintf(`<li class="term-condition-item">%s</li>`, condition))
	}

	termHTML.WriteString(`</ol>`)
	return termHTML.String()
}

func quotationTemplateHTMLToPDF(html string, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).
				WithPaperHeight(11.7).
				Do(ctx)
			if err != nil {
				return err
			}
			*res = buf
			return nil
		}),
	}
}

func quotationTemplateMapToResponse(template *database.QuotationTemplate) model.QuotationTemplateResponse {
	return model.QuotationTemplateResponse{
		ID:              template.ID,
		Name:            template.Name,
		CreatedAt:       template.CreatedAt,
		CreatedBy:       template.CreatedBy,
		UpdatedAt:       template.UpdatedAt,
		UpdatedBy:       template.UpdatedBy,
		IsEnabled:       template.IsEnabled,
		MainHTMLContent: template.MainHTMLContent,
		CSSContent:      template.CSSContent,
		AreaHTMLContent: template.AreaHTMLContent,
		VariableList:    template.VariableList,
		DefaultFileName: template.DefaultFileName,
		Company:         template.Company,
	}
}

func quotationTemplateExtractVariablesFromHTML(htmlContent string) []string {
	pattern := `\{\{([^}]+)\}\}`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(htmlContent, -1)

	variableSet := make(map[string]bool)
	var variables []string

	for _, match := range matches {
		if len(match) > 1 {
			variable := strings.TrimSpace(match[1])
			if variable != "" && !variableSet[variable] {
				variableSet[variable] = true
				variables = append(variables, variable)
			}
		}
	}

	return variables
}

// isPriceVariable checks if a variable name indicates it should be formatted as price
func quotationTemplateIsPriceVariable(variableName string) bool {
	lowerName := strings.ToLower(variableName)
	priceKeywords := []string{"price", "total", "subtotal", "amount", "cost", "discount", "tax", "sst", "grand"}

	for _, keyword := range priceKeywords {
		if strings.Contains(lowerName, keyword) {
			return true
		}
	}
	return false
}
