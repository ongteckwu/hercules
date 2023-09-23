package main

import (
	"hercules/src/workflow"
	"log"

	"github.com/joho/godotenv"
)

func initialize() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	initialize()
	workflow.RunGitCloneWorkflow("https://github.com/ashutoshji/online-store")
}
