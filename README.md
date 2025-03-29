# Glitchy

> Your AI-powered code reviewer

Glitchy is a friendly GitHub App that automatically reviews pull requests using Claude's AI capabilities. It analyzes code changes when PRs are created or updated, then leaves thoughtful comments.

## What Does Glitchy Do?

When someone opens a PR or pushes new commits, Glitchy:

1. Grabs the diff from GitHub
2. Sends it to Claude for intelligent analysis
3. Posts the review comments directly on your PR

Glitchy gives you immediate feedback on:

- Potential bugs and edge cases
- Code structure and organization
- Readability and maintainability
- Security concerns
- Best practices and design patterns

## Setup

### Prerequisites

- A GitHub account (duh!)
- An [Anthropic API key](https://console.anthropic.com/) for Claude
- [Go](https://golang.org/dl/) 1.24+ (if running locally)
- [Docker](https://www.docker.com/get-started) (if using containerized deployment)

### GitHub App Setup

1. [Create a new GitHub App](https://github.com/settings/apps/new) with these permissions:

   - Pull requests: Read & write
   - Contents: Read-only

2. Generate a private key and download it

3. Install the app on your repositories

4. Note your App ID and Installation ID (use `go run main.go --list-installations` to find installation IDs)

## Configuration

Create a `.env` file with these variables:

```
GITHUB_APP_ID=your_app_id
GITHUB_APP_INSTALLATION_ID=your_installation_id
GITHUB_APP_PRIVATE_KEY_PATH=path/to/your/private_key.pem
CLAUDE_API_KEY=your_claude_api_key
WEBHOOK_SECRET=random_secret_string
PORT=8080
```

## Running Glitchy

### Using Docker (recommended)

```bash
# Build the Docker image
$ make docker-build

# Run with Docker
$ make docker-run
```

### Running Locally

```bash
# Install dependencies
$ go mod tidy

# Run the app
$ make run
```

## Exposing Your Webhook

For GitHub to reach your local instance, you'll need to expose your webhook. Tools like [ngrok](https://ngrok.com/) work great for this:

```bash
$ ngrok http 8080
```

Then update your GitHub App's webhook URL with the ngrok URL + "/webhook".

## Want to Help?

Contributions are welcome! Feel free to open issues or PRs if you have ideas for making Glitchy even better.

## License

MIT - See the [LICENSE](./LICENSE) file for details.

---

Made with ❤️ and a bit of digital glitchiness.
