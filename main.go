package main

import (
	"hercules/src/arg_parser"
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
	arg_parser.ArgParser()
}
