package github

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/golang-jwt/jwt"
	"github.com/google/go-github/v61/github"
)

// AppAuth handles GitHub App authentication
type AppAuth struct {
	AppID          int64
	InstallationID int64
	PrivateKey     *rsa.PrivateKey
	HTTPClient     *http.Client
}

// NewAppAuth creates a new GitHub App authenticator
func NewAppAuth() (*AppAuth, error) {
	log.Debug("Initializing GitHub App authentication")
	
	// Parse App ID
	appIDStr := os.Getenv("GITHUB_APP_ID")
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub App ID: %v", err)
	}

	// Parse Installation ID
	installIDStr := os.Getenv("GITHUB_APP_INSTALLATION_ID")
	installID, err := strconv.ParseInt(installIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub Installation ID: %v", err)
	}

	// Load private key from environment-specified path
	keyPath := os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = "./glitchy.pem" // Default fallback
		log.Warn("GITHUB_APP_PRIVATE_KEY_PATH not set, using default path", "path", keyPath)
	}
	
	log.Debug("Loading private key", "path", keyPath)
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		log.Error("Private key file not found", "path", keyPath)
		return nil, fmt.Errorf("private key file not found at %s", keyPath)
	}
	
	privateKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		log.Error("Failed to read private key", "path", keyPath, "error", err)
		return nil, fmt.Errorf("error reading private key: %v", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing private key: %v", err)
	}

	log.Debug("GitHub App authentication initialized", "appID", appID, "installationID", installID)
	return &AppAuth{
		AppID:          appID,
		InstallationID: installID,
		PrivateKey:     privateKey,
		HTTPClient:     &http.Client{},
	}, nil
}

// CreateJWT creates a JWT for GitHub App authentication
func (a *AppAuth) CreateJWT() (string, error) {
	log.Debug("Creating JWT token for GitHub App")
	now := time.Now()
	claims := jwt.StandardClaims{
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(10 * time.Minute).Unix(),
		Issuer:    fmt.Sprintf("%d", a.AppID),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	
	// Set the algorithm header explicitly - GitHub requires this
	token.Header["alg"] = "RS256"
	
	signedToken, err := token.SignedString(a.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("error signing JWT: %v", err)
	}

	return signedToken, nil
}

// GetInstallationClient returns a GitHub client authenticated as an installation
func (a *AppAuth) GetInstallationClient(ctx context.Context) (*github.Client, error) {
	log.Debug("Getting GitHub installation client")
	
	// First, get a JWT-authenticated client
	jwtToken, err := a.CreateJWT()
	if err != nil {
		return nil, err
	}

	// Create a temporary client with JWT auth
	tempClient := github.NewClient(&http.Client{
		Transport: &transport{
			Base: http.DefaultTransport,
			Token: jwtToken,
		},
	})

	// Get an installation token
	log.Debug("Requesting installation token", "installationID", a.InstallationID)
	token, _, err := tempClient.Apps.CreateInstallationToken(
		ctx,
		a.InstallationID,
		&github.InstallationTokenOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("error getting installation token: %v", err)
	}

	// Create a client with the installation token
	tokenClient := github.NewClient(&http.Client{
		Transport: &github.BasicAuthTransport{
			Username: "x-access-token",
			Password: token.GetToken(),
		},
	})

	log.Debug("GitHub installation client created successfully")
	return tokenClient, nil
}

// GetInstallations lists all installations for this GitHub App
func (a *AppAuth) GetInstallations(ctx context.Context) ([]*github.Installation, error) {
	log.Debug("Listing GitHub App installations")
	
	// Create a JWT for GitHub App authentication
	jwtToken, err := a.CreateJWT()
	if err != nil {
		return nil, err
	}

	// Create a temporary client with JWT auth
	tempClient := github.NewClient(&http.Client{
		Transport: &transport{
			Base: http.DefaultTransport,
			Token: jwtToken,
		},
	})

	// List installations
	installations, _, err := tempClient.Apps.ListInstallations(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error listing installations: %v", err)
	}

	log.Debug("Found GitHub App installations", "count", len(installations))
	return installations, nil
}

// transport implements http.RoundTripper for JWT authentication
type transport struct {
	Base  http.RoundTripper
	Token string
}

// RoundTrip adds the Authorization header to each request
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req2 := req.Clone(req.Context())
	
	// Set the Authorization header with the Bearer token
	req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.Token))
	
	// Add required GitHub headers
	req2.Header.Set("Accept", "application/vnd.github.v3+json")
	
	// Make the actual request
	return t.Base.RoundTrip(req2)
}
