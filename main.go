package main

import (
	"fmt"
	"os"
	"prisma-gostruct-migration/utils"
)

func main() {
	// Check if the correct number of arguments is passed
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <prisma file path>")
		os.Exit(1)
	}

	// Get the arguments from command line
	prismaFile := os.Args[1]

	// Read prisma file
	schema, err := utils.ReadSchemaFile(prismaFile)
	if err != nil {
		fmt.Println("Error reading schema file:", err)
		os.Exit(1)
	}

	// Parse the schema to extract models and fields
	models, enums, err := utils.ParseSchemaFile(schema)
	if err != nil {
		fmt.Println("Error parsing schema:", err)
		os.Exit(1)
	}

	// Generate Go structs from the parsed models and write them to files
	for _, model := range models {
		goStruct := utils.GenerateGoStruct(model, enums)

		utils.WriteStructToFile(model, goStruct)

		fmt.Printf("Successfully wrote %s", model.Name)
	}
}
