package main

import (
	"log"

	"github.com/joho/godotenv"

	"code-runner/dashboard/internal/bootstrap/server"
	"code-runner/dashboard/internal/config"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No local .env loaded (using environment variables):", err)
	}
	cfg := config.LoadConfig()
	server.StartServer(cfg)
}
