package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"diogocastro.me/glitchy/internal/claude"
	"github.com/charmbracelet/log"
	gh "github.com/google/go-github/v61/github"
)

// Glitchy handles GitHub PR webhook events and generates reviews
type Glitchy struct {
	claudeClient *claude.Client
	appAuth      *AppAuth
	webhookSecret string
}

// NewGlitchy creates a new PR review bot
func NewGlitchy() *Glitchy {
	// Initialize Claude client
	claudeClient := claude.NewClient()

	// Initialize GitHub App auth
	appAuth, err := NewAppAuth()
	if err != nil {
		log.Fatal("Error initializing GitHub App auth", "error", err)
	}

	return &Glitchy{
		claudeClient:  claudeClient,
		appAuth:      appAuth,
		webhookSecret: os.Getenv("WEBHOOK_SECRET"),
	}
}

// ValidateSignature validates the GitHub webhook signature
func (bot *Glitchy) ValidateSignature(payload []byte, signatureHeader string) bool {
	if signatureHeader == "" {
		return false
	}

	parts := strings.SplitN(signatureHeader, "=", 2)
	if len(parts) != 2 {
		return false
	}

	signature, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(bot.webhookSecret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)

	return hmac.Equal(signature, expectedMAC)
}

// HandleWebhook processes GitHub webhook events
func (bot *Glitchy) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	log.Info("Received webhook", "method", r.Method, "path", r.URL.Path)
	
	// Read and validate payload
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read request body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	
	log.Debug("Payload received", "size", len(payload))
	
	// Verify webhook signature
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		log.Warn("Missing X-Hub-Signature-256 header")
	}
	
	if !bot.ValidateSignature(payload, signature) {
		log.Error("Invalid signature", "signature", signature)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse the event
	eventType := r.Header.Get("X-GitHub-Event")
	log.Info("Processing webhook event", "type", eventType)
	
	if eventType == "ping" {
		log.Info("Received ping event")
		fmt.Fprintf(w, "Pong!")
		return
	}

	if eventType != "pull_request" {
		log.Info("Ignoring non-pull request event", "type", eventType)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse the pull request event
	event, err := gh.ParseWebHook(eventType, payload)
	if err != nil {
		log.Error("Failed to parse webhook", "error", err)
		http.Error(w, "Failed to parse webhook", http.StatusBadRequest)
		return
	}

	// Handle the pull request event
	prEvent, ok := event.(*gh.PullRequestEvent)
	if !ok {
		log.Error("Invalid event payload type", "type", fmt.Sprintf("%T", event))
		http.Error(w, "Invalid event payload", http.StatusBadRequest)
		return
	}

	// Process only opened or synchronize events (new PRs or updates)
	action := prEvent.GetAction()
	log.Info("Pull request action", "action", action)
	
	if action != "opened" && action != "synchronize" {
		log.Info("Ignoring pull request action", "action", action)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Info("Processing pull request",
		"pr", prEvent.GetPullRequest().GetNumber(),
		"repo", prEvent.GetRepo().GetFullName(),
		"async", true)
		
	// Process the pull request asynchronously
	go bot.processPullRequest(prEvent)

	w.WriteHeader(http.StatusOK)
}

// processPullRequest handles the pull request review process
func (bot *Glitchy) processPullRequest(event *gh.PullRequestEvent) {
	ctx := context.Background()
	pr := event.GetPullRequest()
	owner := event.GetRepo().GetOwner().GetLogin()
	repo := event.GetRepo().GetName()
	number := pr.GetNumber()

	log.Info("Processing pull request", "number", number, "repo", fmt.Sprintf("%s/%s", owner, repo))
	
	// Get GitHub client with installation token
	githubClient, err := bot.appAuth.GetInstallationClient(ctx)
	if err != nil {
		log.Error("Failed to get GitHub client", "error", err)
		return
	}

	// Get the PR diff
	diff, _, err := githubClient.PullRequests.GetRaw(
		ctx,
		owner,
		repo,
		number,
		gh.RawOptions{Type: gh.Diff},
	)
	if err != nil {
		log.Error("Failed to get PR diff", "error", err, "pr", number)
		return
	}

	// Get a review from Claude
	log.Info("Requesting review from Claude", "pr", number)
	review, err := bot.claudeClient.ReviewPullRequest(diff)
	if err != nil {
		log.Error("Failed to get review from Claude", "error", err, "pr", number)
		return
	}

	// Create the review on GitHub
	log.Info("Submitting review to GitHub", "pr", number)
	_, _, err = githubClient.PullRequests.CreateReview(
		ctx,
		owner,
		repo,
		number,
		&gh.PullRequestReviewRequest{
			Body:  &review,
			Event: gh.String("COMMENT"),
		},
	)
	if err != nil {
		log.Error("Failed to create PR review", "error", err, "pr", number)
		return
	}

	log.Info("Successfully submitted review", "pr", number)
}