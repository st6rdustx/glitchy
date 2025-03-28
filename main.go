package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"diogocastro.me/glitchy/github"
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

func main() {
	// Configure the logger
	log.SetLevel(log.InfoLevel)
	log.SetReportTimestamp(true)
	log.SetReportCaller(false)

	if err := godotenv.Load(); err != nil {
		log.Warn("Warning: .env file not found, using environment variables")
	}

	// Check for list-installations command
	if len(os.Args) > 1 && os.Args[1] == "--list-installations" {
		// Only check for App ID and private key
		requiredEnvVars := []string{
			"GITHUB_APP_ID",
		}
		for _, envVar := range requiredEnvVars {
			if os.Getenv(envVar) == "" {
				log.Fatal("Error: environment variable required", "var", envVar)
			}
		}
		
		// Initialize just the app auth
		appAuth, err := github.NewAppAuth()
		if err != nil {
			log.Fatal("Error initializing GitHub App auth", "error", err)
		}
		
		// Get and print installations
		installations, err := appAuth.GetInstallations(context.Background())
		if err != nil {
			log.Fatal("Error getting installations", "error", err)
		}
		
		fmt.Println("GitHub App Installations:")
		for _, inst := range installations {
			fmt.Printf("- ID: %d, Account: %s\n", inst.GetID(), inst.GetAccount().GetLogin())
		}
		return
	}

	// Regular startup with all required env vars
	requiredEnvVars := []string{
		"GITHUB_APP_ID", 
		"GITHUB_APP_INSTALLATION_ID",
		"CLAUDE_API_KEY", 
		"WEBHOOK_SECRET",
	}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Fatal("Error: environment variable required", "var", envVar)
		}
	}

	// Verify the private key file exists
	privateKeyPath := os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH")
	if privateKeyPath != "" {
		if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
			log.Fatal("GitHub App private key file not found", "path", privateKeyPath)
		} else if err != nil {
			log.Fatal("Error accessing GitHub App private key file", "path", privateKeyPath, "error", err)
		}
		log.Debug("GitHub App private key file found", "path", privateKeyPath)
	}

	bot := github.NewGlitchy()

	// Main webhook handler
	http.HandleFunc("/webhook", bot.HandleWebhook)
	
	// Add a debug endpoint
	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Debug endpoint accessed", "method", r.Method, "path", r.URL.Path)
		
		info := map[string]string{
			"status":    "running",
			"version":   runtime.Version(),
			"app_id":    os.Getenv("GITHUB_APP_ID"),
			"webhook_configured": "true",
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Info("starting server", "port", port)
	log.Info("debug url", "url", "http://localhost:"+port+"/debug")
	log.Info("webhook url", "url", "http://localhost:"+port+"/webhook")
	
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("error starting server", "error", err)
	}
}