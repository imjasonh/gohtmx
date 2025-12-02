package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v66/github"
)

type githubApp struct {
	client *github.Client
}

func newGithubApp() *githubApp {
	var httpClient *http.Client

	// Check for GitHub token in environment
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		httpClient = &http.Client{
			Transport: &tokenTransport{
				token: token,
			},
		}
	}

	return &githubApp{
		client: github.NewClient(httpClient),
	}
}

// tokenTransport adds the GitHub token to requests
type tokenTransport struct {
	token string
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(req)
}

// parseRepoPath parses owner/repo from various input formats
func parseRepoPath(input string) (owner, repo string, err error) {
	input = strings.TrimSpace(input)

	// Handle full GitHub URLs
	if strings.Contains(input, "github.com") {
		parts := strings.Split(input, "github.com/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub URL")
		}
		input = parts[1]
	}

	// Remove leading/trailing slashes
	input = strings.Trim(input, "/")

	// Split on first slash
	parts := strings.SplitN(input, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format: expected owner/repo")
	}

	// Clean up any extra path components
	repoParts := strings.Split(parts[1], "/")
	return parts[0], repoParts[0], nil
}

// getRepository fetches repository information
func (g *githubApp) getRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	repository, _, err := g.client.Repositories.Get(ctx, owner, repo)
	return repository, err
}

// getDefaultBranch returns the default branch name for a repository
func (g *githubApp) getDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	repository, err := g.getRepository(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	if repository.DefaultBranch != nil {
		return *repository.DefaultBranch, nil
	}
	return "main", nil
}

// listBranches returns all branches for a repository
func (g *githubApp) listBranches(ctx context.Context, owner, repo string) ([]*github.Branch, error) {
	opts := &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	branches, _, err := g.client.Repositories.ListBranches(ctx, owner, repo, opts)
	return branches, err
}

// listTags returns all tags for a repository
func (g *githubApp) listTags(ctx context.Context, owner, repo string) ([]*github.RepositoryTag, error) {
	opts := &github.ListOptions{PerPage: 100}
	tags, _, err := g.client.Repositories.ListTags(ctx, owner, repo, opts)
	return tags, err
}

// getDirectoryContents returns the contents of a directory
func (g *githubApp) getDirectoryContents(ctx context.Context, owner, repo, path, ref string) ([]*github.RepositoryContent, error) {
	opts := &github.RepositoryContentGetOptions{Ref: ref}
	_, contents, _, err := g.client.Repositories.GetContents(ctx, owner, repo, path, opts)
	return contents, err
}

// getFileContent returns the content of a file
func (g *githubApp) getFileContent(ctx context.Context, owner, repo, path, ref string) (*github.RepositoryContent, error) {
	opts := &github.RepositoryContentGetOptions{Ref: ref}
	content, _, _, err := g.client.Repositories.GetContents(ctx, owner, repo, path, opts)
	return content, err
}
