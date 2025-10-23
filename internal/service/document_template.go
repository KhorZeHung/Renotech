package service

import (
	"context"
	"fmt"
	"regexp"
	"sort"
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

// DocumentTemplateCreate creates a new document template
func DocumentTemplateCreate(input *model.DocumentTemplateCreateRequest, systemContext *model.SystemContext) (*model.DocumentTemplateResponse, error) {
	if err := documentTemplateValidateCreate(input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("document_templates")
	defaultUserID := primitive.NewObjectID()

	template := &database.DocumentTemplate{
		Name:            input.Name,
		Description:     input.Description,
		Html:            input.Html,
		VariableHtml:    input.VariableHtml,
		EmbeddedHtml:    input.EmbeddedHtml,
		DefaultFileName: input.DefaultFileName,
		IsDefault:       false,
		Type:            input.Type,
		Company:         input.Company,
		CreatedAt:       time.Now(),
		CreatedBy:       defaultUserID,
		UpdatedAt:       time.Now(),
		UpdatedBy:       &defaultUserID,
		IsDeleted:       false,
	}

	result, err := collection.InsertOne(context.Background(), template)
	if err != nil {
		systemContext.Logger.Error("Failed to create document template: " + err.Error())
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to create document template", nil)
	}

	templateID := result.InsertedID.(primitive.ObjectID)
	return DocumentTemplateGetByID(templateID, systemContext)
}

// DocumentTemplateUpdate updates an existing document template
func DocumentTemplateUpdate(templateID primitive.ObjectID, input *model.DocumentTemplateUpdateRequest, systemContext *model.SystemContext) (*model.DocumentTemplateResponse, error) {
	if err := documentTemplateValidateUpdate(templateID, input, systemContext); err != nil {
		return nil, err
	}

	collection := systemContext.MongoDB.Collection("document_templates")
	defaultUserID := primitive.NewObjectID()

	update := bson.M{
		"$set": bson.M{
			"name":            input.Name,
			"description":     input.Description,
			"html":            input.Html,
			"variableHtml":    input.VariableHtml,
			"embeddedHtml":    input.EmbeddedHtml,
			"defaultFileName": input.DefaultFileName,
			"type":            input.Type,
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
		systemContext.Logger.Error("Failed to update document template: " + err.Error())
		return nil, utils.SystemError(enum.ErrorCodeInternal, "Failed to update document template", nil)
	}

	return DocumentTemplateGetByID(templateID, systemContext)
}

// DocumentTemplateDelete soft deletes a document template
func DocumentTemplateDelete(templateID primitive.ObjectID, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("document_templates")
	defaultUserID := primitive.NewObjectID()

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
		systemContext.Logger.Error("Failed to delete document template: " + err.Error())
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to delete document template", nil)
	}

	if result.MatchedCount == 0 {
		return utils.SystemError(enum.ErrorCodeNotFound, "Document template not found", nil)
	}

	return nil
}

// DocumentTemplateGetByID retrieves a document template by ID
func DocumentTemplateGetByID(templateID primitive.ObjectID, systemContext *model.SystemContext) (*model.DocumentTemplateResponse, error) {
	collection := systemContext.MongoDB.Collection("document_templates")

	var template database.DocumentTemplate
	err := collection.FindOne(context.Background(), bson.M{
		"_id":       templateID,
		"isDeleted": false,
	}).Decode(&template)
	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Document template not found", nil)
	}

	response := documentTemplateMapToResponse(&template)
	return &response, nil
}

// DocumentTemplateList retrieves a paginated list of document templates
func DocumentTemplateList(page, limit int, companyID *primitive.ObjectID, systemContext *model.SystemContext) ([]model.DocumentTemplateResponse, int64, error) {
	collection := systemContext.MongoDB.Collection("document_templates")

	filter := bson.M{"isDeleted": false}
	if companyID != nil {
		filter["company"] = companyID
	}

	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("Failed to count document templates: " + err.Error())
		return nil, 0, utils.SystemError(enum.ErrorCodeInternal, "Failed to count templates", nil)
	}

	skip := (page - 1) * limit
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"createdAt": -1})

	cursor, err := collection.Find(context.Background(), filter, findOptions)
	if err != nil {
		systemContext.Logger.Error("Failed to retrieve document templates: " + err.Error())
		return nil, 0, utils.SystemError(enum.ErrorCodeInternal, "Failed to retrieve templates", nil)
	}
	defer cursor.Close(context.Background())

	var templates []database.DocumentTemplate
	if err = cursor.All(context.Background(), &templates); err != nil {
		systemContext.Logger.Error("Failed to decode document templates: " + err.Error())
		return nil, 0, utils.SystemError(enum.ErrorCodeInternal, "Failed to decode templates", nil)
	}

	var responses []model.DocumentTemplateResponse
	for _, template := range templates {
		responses = append(responses, documentTemplateMapToResponse(&template))
	}

	return responses, total, nil
}

// DocumentTemplatePreview generates HTML preview from template and data
func DocumentTemplatePreview(templateType string, data bson.M, companyID *primitive.ObjectID, systemContext *model.SystemContext) (string, error) {
	template, err := documentTemplateFindByType(templateType, companyID, systemContext)
	if err != nil {
		return "", err
	}

	html, err := documentTemplateProcessHTML(template, data, true, systemContext)
	if err != nil {
		return "", err
	}

	return html, nil
}

// DocumentTemplateGenerate generates PDF from template and data
func DocumentTemplateGenerate(templateType string, data bson.M, companyID *primitive.ObjectID, systemContext *model.SystemContext) ([]byte, string, error) {
	template, err := documentTemplateFindByType(templateType, companyID, systemContext)
	if err != nil {
		return nil, "", err
	}

	html, err := documentTemplateProcessHTML(template, data, false, systemContext)
	if err != nil {
		return nil, "", err
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var pdfBuffer []byte
	err = chromedp.Run(ctx, documentTemplateHTMLToPDF(html, &pdfBuffer))
	if err != nil {
		systemContext.Logger.Error("Failed to generate PDF: " + err.Error())
		return nil, "", utils.SystemError(enum.ErrorCodeInternal, "Failed to generate PDF", nil)
	}

	// Process filename to replace variables
	filename := documentTemplateProcessFilename(template.DefaultFileName, data)
	if !strings.HasSuffix(filename, ".pdf") {
		filename += ".pdf"
	}

	return pdfBuffer, filename, nil
}

// Validation functions

func documentTemplateValidateCreate(input *model.DocumentTemplateCreateRequest, systemContext *model.SystemContext) error {
	if strings.TrimSpace(input.Name) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Name is required", nil)
	}
	if strings.TrimSpace(input.Html) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "HTML is required", nil)
	}
	if strings.TrimSpace(input.Type) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Type is required", nil)
	}
	if strings.TrimSpace(input.DefaultFileName) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Default filename is required", nil)
	}

	// Check name uniqueness
	if err := documentTemplateValidateNameUniqueness(primitive.NilObjectID, input.Name, input.Company, systemContext); err != nil {
		return err
	}

	// Check type uniqueness per company
	if err := documentTemplateValidateTypeUniqueness(primitive.NilObjectID, input.Type, input.Company, systemContext); err != nil {
		return err
	}

	// Validate variableHtml
	if err := documentTemplateValidateVariableHtml(input.VariableHtml); err != nil {
		return err
	}

	// Validate embeddedHtml
	if err := documentTemplateValidateEmbeddedHtml(input.EmbeddedHtml); err != nil {
		return err
	}

	return nil
}

func documentTemplateValidateUpdate(templateID primitive.ObjectID, input *model.DocumentTemplateUpdateRequest, systemContext *model.SystemContext) error {
	if strings.TrimSpace(input.Name) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Name is required", nil)
	}
	if strings.TrimSpace(input.Html) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "HTML is required", nil)
	}
	if strings.TrimSpace(input.Type) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Type is required", nil)
	}
	if strings.TrimSpace(input.DefaultFileName) == "" {
		return utils.SystemError(enum.ErrorCodeValidation, "Default filename is required", nil)
	}

	// Check if template exists
	_, err := DocumentTemplateGetByID(templateID, systemContext)
	if err != nil {
		return err
	}

	// Check name uniqueness
	if err := documentTemplateValidateNameUniqueness(templateID, input.Name, input.Company, systemContext); err != nil {
		return err
	}

	// Check type uniqueness per company
	if err := documentTemplateValidateTypeUniqueness(templateID, input.Type, input.Company, systemContext); err != nil {
		return err
	}

	// Validate variableHtml
	if err := documentTemplateValidateVariableHtml(input.VariableHtml); err != nil {
		return err
	}

	// Validate embeddedHtml
	if err := documentTemplateValidateEmbeddedHtml(input.EmbeddedHtml); err != nil {
		return err
	}

	return nil
}

func documentTemplateValidateNameUniqueness(excludeID primitive.ObjectID, name string, companyID *primitive.ObjectID, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("document_templates")

	filter := bson.M{
		"name":      name,
		"isDeleted": false,
	}

	if companyID != nil {
		filter["company"] = companyID
	} else {
		filter["company"] = nil
	}

	if excludeID != primitive.NilObjectID {
		filter["_id"] = bson.M{"$ne": excludeID}
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("Failed to check name uniqueness: " + err.Error())
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to validate name", nil)
	}

	if count > 0 {
		return utils.SystemError(enum.ErrorCodeValidation, "Template name already exists", nil)
	}

	return nil
}

func documentTemplateValidateTypeUniqueness(excludeID primitive.ObjectID, templateType string, companyID *primitive.ObjectID, systemContext *model.SystemContext) error {
	collection := systemContext.MongoDB.Collection("document_templates")

	filter := bson.M{
		"type":      templateType,
		"isDeleted": false,
	}

	if companyID != nil {
		filter["company"] = companyID
	} else {
		filter["company"] = nil
	}

	if excludeID != primitive.NilObjectID {
		filter["_id"] = bson.M{"$ne": excludeID}
	}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		systemContext.Logger.Error("Failed to check type uniqueness: " + err.Error())
		return utils.SystemError(enum.ErrorCodeInternal, "Failed to validate type", nil)
	}

	if count > 0 {
		return utils.SystemError(enum.ErrorCodeValidation, "Template type already exists for this company", nil)
	}

	return nil
}

// documentTemplateValidateVariableHtml validates that variable keys match variables in their HTML
func documentTemplateValidateVariableHtml(variableHtml map[string]string) error {
	for key, html := range variableHtml {
		extractedVars := documentTemplateExtractVariables(html)

		// Check if the key appears in the HTML as a variable
		found := false
		for _, extractedVar := range extractedVars {
			if extractedVar == key {
				found = true
				break
			}
		}

		if !found {
			return utils.SystemError(enum.ErrorCodeValidation,
				fmt.Sprintf("VariableHtml key '%s' must appear as {{%s}} in its HTML value", key, key), nil)
		}
	}

	return nil
}

// documentTemplateValidateEmbeddedHtml validates embedded HTML structure and nesting
func documentTemplateValidateEmbeddedHtml(embeddedHtml map[string]string) error {
	if len(embeddedHtml) == 0 {
		return nil
	}

	// Sort keys by nesting depth (deepest first)
	sortedKeys := documentTemplateSortKeysByDepth(embeddedHtml)

	// Track which keys exist
	keySet := make(map[string]bool)
	for key := range embeddedHtml {
		keySet[key] = true
	}

	// Validate each embedded HTML
	for _, key := range sortedKeys {
		html := embeddedHtml[key]

		// Extract variables from this HTML
		extractedVars := documentTemplateExtractVariables(html)

		// Validate all variables start with the key prefix
		for _, variable := range extractedVars {
			if !strings.HasPrefix(variable, key+".") && variable != key {
				return utils.SystemError(enum.ErrorCodeValidation,
					fmt.Sprintf("Variable '%s' in embeddedHtml['%s'] must start with '%s.' prefix", variable, key, key), nil)
			}
		}

		// If this is a nested key (contains dot), validate parent exists
		if strings.Contains(key, ".") {
			parts := strings.Split(key, ".")
			parent := strings.Join(parts[:len(parts)-1], ".")

			if !keySet[parent] {
				return utils.SystemError(enum.ErrorCodeValidation,
					fmt.Sprintf("Nested key '%s' requires parent '%s' to be defined in embeddedHtml", key, parent), nil)
			}

			// Check if parent's HTML contains this child as a placeholder
			parentHtml := embeddedHtml[parent]
			childPlaceholder := fmt.Sprintf("{{%s}}", key)
			if !strings.Contains(parentHtml, childPlaceholder) {
				return utils.SystemError(enum.ErrorCodeValidation,
					fmt.Sprintf("Parent embeddedHtml['%s'] must contain placeholder {{%s}} for nested child", parent, key), nil)
			}
		}
	}

	return nil
}

// Processing functions

func documentTemplateFindByType(templateType string, companyID *primitive.ObjectID, systemContext *model.SystemContext) (*database.DocumentTemplate, error) {
	collection := systemContext.MongoDB.Collection("document_templates")

	// Try to find company template first
	if companyID != nil {
		var template database.DocumentTemplate
		err := collection.FindOne(context.Background(), bson.M{
			"type":      templateType,
			"company":   companyID,
			"isDeleted": false,
		}).Decode(&template)
		if err == nil {
			return &template, nil
		}
	}

	// Fallback to default template
	var template database.DocumentTemplate
	err := collection.FindOne(context.Background(), bson.M{
		"type":      templateType,
		"isDefault": true,
		"isDeleted": false,
	}).Decode(&template)

	if err != nil {
		return nil, utils.SystemError(enum.ErrorCodeNotFound, "Document template not found for type: "+templateType, nil)
	}

	return &template, nil
}

func documentTemplateProcessHTML(template *database.DocumentTemplate, data bson.M, forPreview bool, systemContext *model.SystemContext) (string, error) {
	// Clone data to avoid modifying original
	processedData := make(bson.M)
	for k, v := range data {
		processedData[k] = v
	}

	embeddedKeys := make(map[string]bool)

	// Phase 1: Process embeddedHtml (deepest nesting first) and update payload
	if len(template.EmbeddedHtml) > 0 {
		sortedKeys := documentTemplateSortKeysByDepth(template.EmbeddedHtml)

		for _, key := range sortedKeys {
			fmt.Println(key)
			embeddedTemplate := template.EmbeddedHtml[key]
			fmt.Println(embeddedTemplate)
			processedEmbedded := documentTemplateProcessEmbeddedHtml(key, embeddedTemplate, processedData, template.EmbeddedHtml)

			// Update the payload with rendered HTML
			// For top-level keys (e.g., "areas"), update directly
			// For nested keys (e.g., "areas.areaMaterials"), this is already handled in recursion
			if !strings.Contains(key, ".") {
				processedData[key] = processedEmbedded
				embeddedKeys[key] = true
			}
		}
	}

	// Phase 2: Process variableHtml and update payload
	if len(template.VariableHtml) > 0 {
		// Separate variables into two groups: those in embeddedHtml and those not
		varsInEmbedded := []string{}
		varsNotInEmbedded := []string{}

		// Categorize variables
		for varName := range template.VariableHtml {
			if embeddedKeys[varName] {
				varsInEmbedded = append(varsInEmbedded, varName)
			} else {
				varsNotInEmbedded = append(varsNotInEmbedded, varName)
			}
		}

		// Process variables in embeddedHtml first
		for _, varName := range varsInEmbedded {
			htmlTemplate := template.VariableHtml[varName]

			// Check if variable exists in data
			if value, exists := processedData[varName]; exists && value != nil {
				valueStr := documentTemplateConvertToString(value)
				placeholder := fmt.Sprintf("{{%s}}", varName)

				// Only apply variableHtml if the template actually contains the placeholder
				// This prevents double-wrapping when value is already the final HTML
				if strings.Contains(htmlTemplate, placeholder) {
					// Replace the placeholder in variableHtml template with the value
					rendered := strings.ReplaceAll(htmlTemplate, placeholder, valueStr)

					// Recursively replace any other placeholders in the rendered HTML
					// This handles cases where variableHtml contains other variable references
					rendered = documentTemplateReplaceVariables(rendered, processedData)

					processedData[varName] = rendered
				}
				// If template doesn't contain the placeholder, keep the value as-is
			}
		}

		// Then process remaining variables
		for _, varName := range varsNotInEmbedded {
			htmlTemplate := template.VariableHtml[varName]
			// Check if variable exists in data
			if value, exists := processedData[varName]; exists && value != nil {
				valueStr := documentTemplateConvertToString(value)
				placeholder := fmt.Sprintf("{{%s}}", varName)

				// Only apply variableHtml if the template actually contains the placeholder
				// This prevents double-wrapping when value is already the final HTML
				if strings.Contains(htmlTemplate, placeholder) {
					// Replace the placeholder in variableHtml template with the value
					rendered := strings.ReplaceAll(htmlTemplate, placeholder, valueStr)

					// Recursively replace any other placeholders in the rendered HTML
					// This handles cases where variableHtml contains other variable references
					rendered = documentTemplateReplaceVariables(rendered, processedData)

					processedData[varName] = rendered
				}
				// If template doesn't contain the placeholder, keep the value as-is
			}
		}
	}

	// Phase 3: Replace all payload values into main HTML
	html := template.Html
	html = documentTemplateReplaceVariables(html, processedData)

	// Phase 4: Remove all unused placeholders (replace with empty string)
	html = documentTemplateRemoveUnusedPlaceholders(html)

	// Wrap for preview if needed
	if forPreview {
		html = documentTemplateWrapForPreview(html)
	}

	return html, nil
}

// documentTemplateRemoveUnusedPlaceholders removes all remaining {{variable}} placeholders
func documentTemplateRemoveUnusedPlaceholders(html string) string {
	// Regex to match {{variable}} or {{nested.variable}} patterns
	re := regexp.MustCompile(`\{\{[a-zA-Z0-9_.]+\}\}`)
	return re.ReplaceAllString(html, "")
}

// documentTemplateProcessFilename processes filename by replacing {{variable}} placeholders with data values
func documentTemplateProcessFilename(filename string, data bson.M) string {
	// Find all {{variable}} patterns
	re := regexp.MustCompile(`\{\{([a-zA-Z0-9_.]+)\}\}`)

	result := re.ReplaceAllStringFunc(filename, func(match string) string {
		// Extract variable name (remove {{ and }})
		varName := match[2 : len(match)-2]

		// Look up value in data
		value := documentTemplateGetValueByPath(data, varName)
		if value == nil {
			return "" // Replace with empty string if not found
		}

		// Convert to string
		return documentTemplateConvertToString(value)
	})

	return result
}

func documentTemplateProcessEmbeddedHtml(key string, embeddedHtml string, data bson.M, embeddedHtmlMap map[string]string) string {
	// Navigate to the data using the key path
	value := documentTemplateGetValueByPath(data, key)
	fmt.Println(value)
	if value == nil {
		return ""
	}

	// Check if value is an array
	switch v := value.(type) {
	case []interface{}:
		// Check if it's an array of objects or primitives
		if len(v) == 0 {
			return ""
		}

		// Check first element to determine type
		switch v[0].(type) {
		case map[string]interface{}, bson.M:
			// Array of objects - process each with nested recursion
			var results []string
			for _, item := range v {
				var itemMap map[string]interface{}

				switch it := item.(type) {
				case map[string]interface{}:
					itemMap = it
				case bson.M:
					itemMap = map[string]interface{}(it)
				default:
					continue
				}

				// Process this item's HTML
				itemHtml := embeddedHtml
				fmt.Println(key)
				fmt.Println(embeddedHtmlMap)

				// First, recursively process any nested embeddedHtml placeholders
				itemHtml = documentTemplateProcessNestedEmbedded(itemHtml, key, itemMap, embeddedHtmlMap)
				// Then replace simple variables for this item
				itemHtml = documentTemplateReplaceNestedVariables(itemHtml, key, itemMap)
				results = append(results, itemHtml)
			}
			return strings.Join(results, "")

		default:
			// Array of primitives (strings, numbers, etc.)
			var results []string
			placeholder := fmt.Sprintf("{{%s}}", key)
			for _, item := range v {
				itemHtml := strings.ReplaceAll(embeddedHtml, placeholder, fmt.Sprintf("%v", item))
				results = append(results, itemHtml)
			}
			return strings.Join(results, "")
		}

	case map[string]interface{}:
		// Single object, not an array
		itemHtml := embeddedHtml
		itemHtml = documentTemplateProcessNestedEmbedded(itemHtml, key, v, embeddedHtmlMap)
		itemHtml = documentTemplateReplaceNestedVariables(itemHtml, key, v)
		return itemHtml

	case bson.M:
		// Single bson.M object
		itemMap := map[string]interface{}(v)
		itemHtml := embeddedHtml
		itemHtml = documentTemplateProcessNestedEmbedded(itemHtml, key, itemMap, embeddedHtmlMap)
		itemHtml = documentTemplateReplaceNestedVariables(itemHtml, key, itemMap)
		return itemHtml

	default:
		return fmt.Sprintf("%v", value)
	}
}

// documentTemplateProcessNestedEmbedded handles recursive processing of nested embeddedHtml placeholders
func documentTemplateProcessNestedEmbedded(html string, parentKey string, itemData map[string]interface{}, embeddedHtmlMap map[string]string) string {
	// Find all nested embedded placeholders in this HTML

	for nestedKey, nestedTemplate := range embeddedHtmlMap {
		// Check if this nested key belongs to the current parent
		if strings.HasPrefix(nestedKey, parentKey+".") {
			placeholder := fmt.Sprintf("{{%s}}", nestedKey)

			fmt.Println(placeholder)

			// Check if this placeholder exists in the current HTML
			if strings.Contains(html, placeholder) {
				// Extract the field name from the nested key (e.g., "areas.areaMaterials" -> "areaMaterials")
				parts := strings.Split(nestedKey, ".")
				fieldName := parts[len(parts)-1]

				// Get the nested data from itemData
				if nestedValue, exists := itemData[fieldName]; exists {
					// Create a temporary data structure for recursive processing
					tempData := bson.M{
						fieldName: nestedValue,
					}

					// Recursively process this nested embedded HTML
					processedNested := documentTemplateProcessEmbeddedHtml(nestedKey, nestedTemplate, tempData, embeddedHtmlMap)

					// Replace the placeholder with the processed HTML
					html = strings.ReplaceAll(html, placeholder, processedNested)
				} else {
					// Field doesn't exist, replace with empty string
					html = strings.ReplaceAll(html, placeholder, "")
				}
			} else {
				fmt.Println(placeholder, "not contain")
			}
		}
	}

	return html
}

func documentTemplateReplaceNestedVariables(html string, parentKey string, data map[string]interface{}) string {
	for key, value := range data {
		fullKey := fmt.Sprintf("%s.%s", parentKey, key)
		placeholder := fmt.Sprintf("{{%s}}", fullKey)

		// Handle nested structures
		switch v := value.(type) {
		case map[string]interface{}:
			// Recursively process nested objects
			html = documentTemplateReplaceNestedVariables(html, fullKey, v)
		case []interface{}:
			// Skip arrays - they should be handled by embeddedHtml
			// Don't replace array placeholders here to avoid conflicts
			continue
		case bson.M:
			// Convert bson.M to map[string]interface{} and recurse
			convertedMap := make(map[string]interface{})
			for k, val := range v {
				convertedMap[k] = val
			}
			html = documentTemplateReplaceNestedVariables(html, fullKey, convertedMap)
		default:
			// Replace simple values only
			html = strings.ReplaceAll(html, placeholder, documentTemplateConvertToString(value))
		}
	}
	return html
}

// documentTemplateConvertToString converts any value to string with proper formatting
// For float types, ensures minimum 2 decimal places while preserving more decimals if present
func documentTemplateConvertToString(value interface{}) string {
	if value == nil {
		return ""
	}

	// If already a string, return as-is
	if str, ok := value.(string); ok {
		return str
	}

	// Handle float types with decimal preservation
	switch v := value.(type) {
	case float32:
		return documentTemplateFormatFloat(float64(v))
	case float64:
		return documentTemplateFormatFloat(v)
	default:
		// For other types, use default formatting
		return fmt.Sprintf("%v", value)
	}
}

// documentTemplateFormatFloat formats a float with minimum 2 decimal places
// but preserves more decimal places if they exist and are non-zero
func documentTemplateFormatFloat(f float64) string {
	// Convert to string with high precision first
	str := fmt.Sprintf("%.10f", f)

	// Remove trailing zeros but keep at least 2 decimal places
	parts := strings.Split(str, ".")
	if len(parts) != 2 {
		return fmt.Sprintf("%.2f", f)
	}

	decimals := parts[1]

	// Find the last non-zero digit
	lastNonZero := -1
	for i := len(decimals) - 1; i >= 0; i-- {
		if decimals[i] != '0' {
			lastNonZero = i
			break
		}
	}

	// Keep at least 2 decimal places
	decimalCount := lastNonZero + 1
	if decimalCount < 2 {
		decimalCount = 2
	}

	return fmt.Sprintf("%.*f", decimalCount, f)
}

func documentTemplateGetValueByPath(data bson.M, path string) interface{} {
	parts := strings.Split(path, ".")
	current := interface{}(data)

	part := parts[len(parts)-1]

	// for _, part := range parts {
	switch v := current.(type) {
	case map[string]interface{}:
		if val, exists := v[part]; exists {
			current = val
		} else {
			return nil
		}
	case bson.M:
		if val, exists := v[part]; exists {
			current = val
		} else {
			return nil
		}
	default:
		return nil
	}
	// }

	return current
}

func documentTemplateReplaceVariables(html string, data bson.M) string {
	for key, value := range data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		html = strings.ReplaceAll(html, placeholder, fmt.Sprintf("%v", value))
	}
	return html
}

// Helper functions

func documentTemplateGetNestingDepth(key string) int {
	return strings.Count(key, ".")
}

func documentTemplateSortKeysByDepth(embeddedHtml map[string]string) []string {
	keys := make([]string, 0, len(embeddedHtml))
	for key := range embeddedHtml {
		keys = append(keys, key)
	}

	// Sort by depth (deepest first), then alphabetically for same depth
	sort.Slice(keys, func(i, j int) bool {
		depthI := documentTemplateGetNestingDepth(keys[i])
		depthJ := documentTemplateGetNestingDepth(keys[j])

		if depthI == depthJ {
			return keys[i] < keys[j]
		}
		return depthI < depthJ // Deeper nesting last
	})

	return keys
}

func documentTemplateExtractVariables(html string) []string {
	pattern := `\{\{([^}]+)\}\}`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(html, -1)

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

func documentTemplateHTMLToPDF(html string, res *[]byte) chromedp.Tasks {
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

func documentTemplateWrapForPreview(content string) string {
	previewCSS := `
	<style>
		* {
			box-sizing: border-box;
		}
		body {
			margin: 0;
			padding: 20px;
			background: #f5f5f5;
			font-family: Arial, sans-serif;
		}
		.a4-preview-container {
			max-width: 890px;
			margin: 0 auto;
		}
		.a4-page {
			width: 794px;
			min-height: 1123px;
			padding: 48px;
			margin: 0 auto 20px;
			background: white;
			box-shadow: 0 0 10px rgba(0,0,0,0.1);
			position: relative;
		}
		.a4-page:last-child {
			margin-bottom: 0;
		}
		@media print {
			body {
				background: white;
				padding: 0;
			}
			.a4-preview-container {
				max-width: none;
			}
			.a4-page {
				box-shadow: none;
				margin: 0;
				padding: 0;
				page-break-after: always;
			}
			.a4-page:last-child {
				page-break-after: auto;
			}
		}
	</style>
	`

	wrappedHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Document Preview</title>
	%s
</head>
<body>
	<div class="a4-preview-container">
		<div class="a4-page">
			%s
		</div>
	</div>
</body>
</html>`, previewCSS, content)

	return wrappedHTML
}

func documentTemplateMapToResponse(template *database.DocumentTemplate) model.DocumentTemplateResponse {
	return model.DocumentTemplateResponse{
		ID:              template.ID,
		Name:            template.Name,
		Description:     template.Description,
		Html:            template.Html,
		VariableHtml:    template.VariableHtml,
		EmbeddedHtml:    template.EmbeddedHtml,
		DefaultFileName: template.DefaultFileName,
		IsDefault:       template.IsDefault,
		Type:            template.Type,
		Company:         template.Company,
		CreatedAt:       template.CreatedAt,
		CreatedBy:       template.CreatedBy,
		UpdatedAt:       template.UpdatedAt,
		UpdatedBy:       template.UpdatedBy,
		IsDeleted:       template.IsDeleted,
	}
}
