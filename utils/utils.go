package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// ReadSchemaFile reads the Prisma schema file and returns its contents as a string.
func ReadSchemaFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file: %v", err)
	}
	return string(data), nil
}

// WriteStructToFile writes the generated Go struct to a file inside the 'models' folder.
func WriteStructToFile(model Model, goStruct string) error {
	// Ensure the "models" directory exists
	if err := os.MkdirAll("models", os.ModePerm); err != nil {
		return fmt.Errorf("failed to create models directory: %v", err)
	}

	// Set the file name and create the file
	fileName := fmt.Sprintf("models/%s.go", strings.ToLower(model.Name))
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", fileName, err)
	}
	defer file.Close()

	// Write Go struct to the file
	if _, err := file.WriteString(goStruct); err != nil {
		return fmt.Errorf("failed to write to file %s: %v", fileName, err)
	}

	return nil
}

// ParseSchemaFile parses the Prisma schema and extracts models and enums.
func ParseSchemaFile(schema string) ([]Model, []Enum, error) {
	modelPattern := regexp.MustCompile(`model (\w+) \{([\s\S]+?)\}`)
	enumPattern := regexp.MustCompile(`enum (\w+) \{([\s\S]+?)\}`)

	// Parse models
	models, err := parseModels(schema, modelPattern)
	if err != nil {
		return nil, nil, err
	}

	// Parse enums
	enums, err := parseEnums(schema, enumPattern)
	if err != nil {
		return nil, nil, err
	}

	return models, enums, nil
}

// parseModels parses model definitions using the provided regex pattern.
func parseModels(schema string, modelPattern *regexp.Regexp) ([]Model, error) {
	var models []Model
	for _, match := range modelPattern.FindAllStringSubmatch(schema, -1) {
		modelName, modelFields := match[1], match[2]
		fields := parseFields(modelFields)

		models = append(models, Model{Name: modelName, Fields: fields})
	}
	return models, nil
}

// parseEnums parses enum definitions from the Prisma schema.
func parseEnums(schema string, enumPattern *regexp.Regexp) ([]Enum, error) {
	var enums []Enum
	for _, match := range enumPattern.FindAllStringSubmatch(schema, -1) {
		enumName, enumValues := match[1], match[2]
		values := parseEnumValues(enumValues)

		enums = append(enums, Enum{Name: enumName, Values: values})
	}
	return enums, nil
}

// parseFields parses individual field definitions for a model.
func parseFields(modelFields string) []Field {
	var fields []Field
	fieldPattern := regexp.MustCompile(`(\w+)\s+([\w\[\]\?]+)(\s+@[\w]+(\([^\)]*\))?)?`)

	for _, fieldMatch := range fieldPattern.FindAllStringSubmatch(modelFields, -1) {
		fieldName, fieldType := fieldMatch[1], fieldMatch[2]
		fieldAnnotation := fieldMatch[3]
		goType, isDefaultType := mapPrismaTypeToGo(fieldType)

		fields = append(fields, Field{
			Name:          fieldName,
			Type:          goType,
			IsDefaultType: isDefaultType,
			annotation:    fieldAnnotation,
		})
	}

	return fields
}

// parseEnumValues extracts the values from enum definitions.
func parseEnumValues(enumValues string) []string {
	var values []string
	enumValuePattern := regexp.MustCompile(`\s*(\w+)`)
	for _, valueMatch := range enumValuePattern.FindAllStringSubmatch(enumValues, -1) {
		values = append(values, valueMatch[1])
	}
	return values
}

// mapPrismaTypeToGo maps Prisma types to Go types.
func mapPrismaTypeToGo(prismaType string) (string, bool) {
	prismaType, containsMark := transformQuestionMarks(prismaType)
	var fieldType string
	var isDefaultType bool = true

	switch prismaType {
	case "Int":
		fieldType = "int"
	case "String":
		fieldType = "string"
	case "Boolean":
		fieldType = "bool"
	case "Float":
		fieldType = "float64"
	case "DateTime":
		fieldType = "time.Time"
	case "Json":
		fieldType = "interface{}"
	default:
		fieldType = prismaType
		isDefaultType = false
	}

	fieldType = transformArrayNotation(fieldType)

	if containsMark && isDefaultType {
		return "*" + fieldType, isDefaultType
	}

	return fieldType, isDefaultType
}

// GenerateGoStruct generates the Go struct for a given model.
func GenerateGoStruct(model Model, enums []Enum) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("package models\n\n"))
	sb.WriteString(fmt.Sprintf("type %s struct {\n", model.Name))

	// Use a map for fast enum name lookup
	enumMap := make(map[string]Enum)
	for _, enum := range enums {
		enumMap[enum.Name] = enum
	}

	// Generate struct fields
	var enumNames []string
	for _, field := range model.Fields {
		fieldType := field.Type
		if !field.IsDefaultType {
			if _, exists := enumMap[fieldType]; exists {
				enumNames = append(enumNames, fieldType)
				fieldType = field.Type
			} else {
				fieldType = "*" + field.Type
			}
		}

		attributes := transformAttributes(field)
		sb.WriteString(fmt.Sprintf("\t%s %s %s\n", strings.Title(field.Name), fieldType, attributes))
	}
	sb.WriteString("}\n\n")

	// Generate enum types and constants if applicable
	if len(enumNames) > 0 {
		for _, enumName := range enumNames {
			matchingEnum := enumMap[enumName]
			sb.WriteString(fmt.Sprintf("type %s string\n\n", matchingEnum.Name))
			sb.WriteString("const (\n")

			// Build acronym for enum values
			acronym := buildAcronym(matchingEnum.Name)
			for _, value := range matchingEnum.Values {
				sb.WriteString(fmt.Sprintf("\t%s %s = \"%s\"\n", acronym+"_"+value, matchingEnum.Name, value))
			}
			sb.WriteString(")\n")
		}
	}

	return sb.String()
}

// buildAcronym generates the acronym based on capital letters in the name.
func buildAcronym(name string) string {
	var acronym string
	for _, ch := range name {
		if unicode.IsUpper(ch) {
			acronym += string(ch)
		}
	}
	return acronym
}

// transformArrayNotation handles array notation transformation.
func transformArrayNotation(s string) string {
	if strings.Contains(s, "[]") {
		s = strings.Replace(s, "[]", "", 1)
		return "[]" + s
	}
	return s
}

// transformQuestionMarks handles nullable field types by removing '?'.
func transformQuestionMarks(s string) (string, bool) {
	if strings.Contains(s, "?") {
		return strings.Replace(s, "?", "", 1), true
	}
	return s, false
}

// transformAttributes generates the struct field attributes (e.g., JSON, GORM tags).
func transformAttributes(field Field) string {
	field.annotation = strings.TrimSpace(field.annotation)
	gormValue := transformGormTag(field)
	return fmt.Sprintf("`gorm:\"%s\" json:\"%s\"`", gormValue, field.Name)
}

// transformGormTag generates GORM-specific field tags based on annotations.
func transformGormTag(field Field) string {
	var gormValue string

	if strings.Contains(field.annotation, "@unique") {
		gormValue = "unique;"
	}

	if strings.Contains(field.annotation, "@id") {
		gormValue = "primaryKey;type:uuid;default:uuid_generate_v4()"
	}

	if strings.Contains(field.annotation, "@updatedAt") {
		gormValue = "default:now();autoUpdateTime"
	}

	if strings.Contains(field.annotation, "@default") {
		// Extract default value using regex
		defaultRegex := regexp.MustCompile(`@default\(([^)]*)\)`)
		fieldsMatch := defaultRegex.FindStringSubmatch(field.annotation)
		if len(fieldsMatch) > 1 {
			var fieldValue string
			// TODO: refactor
			if strings.Contains(fieldsMatch[1], "now") {
				fieldValue = "now()"
			} else {
				fieldValue = fieldsMatch[1]
			}
			gormValue = fmt.Sprintf("default:%s;", fieldValue)
		}
	}

	if strings.Contains(field.annotation, "@relation") && strings.Contains(field.annotation, "fields") && strings.Contains(field.annotation, "references") {
		gormValue = transformRelationTag(field)
	}

	return fmt.Sprintf("%scolumn:%s", gormValue, field.Name)
}

// transformRelationTag generates GORM relation tags.
func transformRelationTag(field Field) string {
	fieldsRegex := regexp.MustCompile(`fields:\s*\[(.*?)\]`)
	referencesRegex := regexp.MustCompile(`references:\s*\[(.*?)\]`)

	fieldsMatch := fieldsRegex.FindStringSubmatch(field.annotation)
	referencesMatch := referencesRegex.FindStringSubmatch(field.annotation)

	var fieldsValue, referencesValue string
	if len(fieldsMatch) > 1 {
		fieldsValue = fieldsMatch[1]
	}
	if len(referencesMatch) > 1 {
		referencesValue = referencesMatch[1]
	}

	return fmt.Sprintf("foreignKey:%s;references:%s;", strings.Title(fieldsValue), strings.Title(referencesValue))
}
