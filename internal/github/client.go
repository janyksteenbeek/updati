package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client
type Client struct {
	client *github.Client
	owner  string
}

// Repository represents a GitHub repository
type Repository struct {
	Owner       string
	Name        string
	FullName    string
	CloneURL    string
	DefaultRef  string
	IsLaravel   bool
	HasComposer bool
	HasNPM      bool
}

// NewClient creates a new GitHub client
func NewClient(token, owner string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		client: github.NewClient(tc),
		owner:  owner,
	}
}

// ListRepositories lists all repositories for the configured owner
func (c *Client) ListRepositories(ctx context.Context) ([]*Repository, error) {
	var allRepos []*Repository

	opts := &github.RepositoryListByUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Type:        "owner",
	}

	for {
		repos, resp, err := c.client.Repositories.ListByUser(ctx, c.owner, opts)
		if err != nil {
			// Try as organization
			orgOpts := &github.RepositoryListByOrgOptions{
				ListOptions: github.ListOptions{PerPage: 100},
				Type:        "all",
			}
			repos, resp, err = c.client.Repositories.ListByOrg(ctx, c.owner, orgOpts)
			if err != nil {
				return nil, fmt.Errorf("failed to list repositories: %w", err)
			}

			for _, repo := range repos {
				allRepos = append(allRepos, convertRepo(repo))
			}

			if resp.NextPage == 0 {
				break
			}
			orgOpts.Page = resp.NextPage
			continue
		}

		for _, repo := range repos {
			allRepos = append(allRepos, convertRepo(repo))
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

func convertRepo(repo *github.Repository) *Repository {
	defaultRef := "main"
	if repo.DefaultBranch != nil {
		defaultRef = *repo.DefaultBranch
	}

	return &Repository{
		Owner:      repo.GetOwner().GetLogin(),
		Name:       repo.GetName(),
		FullName:   repo.GetFullName(),
		CloneURL:   repo.GetCloneURL(),
		DefaultRef: defaultRef,
	}
}

// CheckIfLaravel checks if a repository is a Laravel project
func (c *Client) CheckIfLaravel(ctx context.Context, repo *Repository) error {
	// Check for composer.json
	composerContent, _, _, err := c.client.Repositories.GetContents(
		ctx, repo.Owner, repo.Name, "composer.json",
		&github.RepositoryContentGetOptions{Ref: repo.DefaultRef},
	)
	if err == nil && composerContent != nil {
		repo.HasComposer = true

		// Check if it contains laravel/framework
		content, err := composerContent.GetContent()
		if err == nil && strings.Contains(content, "laravel/framework") {
			repo.IsLaravel = true
		}
	}

	// Check for package.json
	_, _, _, err = c.client.Repositories.GetContents(
		ctx, repo.Owner, repo.Name, "package.json",
		&github.RepositoryContentGetOptions{Ref: repo.DefaultRef},
	)
	if err == nil {
		repo.HasNPM = true
	}

	return nil
}

// GetDefaultBranch gets the default branch for a repository
func (c *Client) GetDefaultBranch(ctx context.Context, repo *Repository) (string, error) {
	r, _, err := c.client.Repositories.Get(ctx, repo.Owner, repo.Name)
	if err != nil {
		return "", fmt.Errorf("failed to get repository: %w", err)
	}

	return r.GetDefaultBranch(), nil
}

// CreateBranch creates a new branch from the default branch
func (c *Client) CreateBranch(ctx context.Context, repo *Repository, branchName string) error {
	// Get the SHA of the default branch
	ref, _, err := c.client.Git.GetRef(ctx, repo.Owner, repo.Name, "refs/heads/"+repo.DefaultRef)
	if err != nil {
		return fmt.Errorf("failed to get default branch ref: %w", err)
	}

	// Create the new branch
	newRef := &github.Reference{
		Ref:    github.String("refs/heads/" + branchName),
		Object: &github.GitObject{SHA: ref.Object.SHA},
	}

	_, _, err = c.client.Git.CreateRef(ctx, repo.Owner, repo.Name, newRef)
	if err != nil {
		// Check if branch already exists
		if strings.Contains(err.Error(), "Reference already exists") {
			// Update existing branch
			_, _, err = c.client.Git.UpdateRef(ctx, repo.Owner, repo.Name, newRef, true)
			if err != nil {
				return fmt.Errorf("failed to update existing branch: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to create branch: %w", err)
	}

	return nil
}

// CreatePullRequest creates a pull request
func (c *Client) CreatePullRequest(ctx context.Context, repo *Repository, title, body, head, base string, labels []string) (*github.PullRequest, error) {
	// Check if PR already exists
	prs, _, err := c.client.PullRequests.List(ctx, repo.Owner, repo.Name, &github.PullRequestListOptions{
		Head:  fmt.Sprintf("%s:%s", repo.Owner, head),
		Base:  base,
		State: "open",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list existing PRs: %w", err)
	}

	if len(prs) > 0 {
		// PR already exists, update it
		pr := prs[0]
		pr, _, err = c.client.PullRequests.Edit(ctx, repo.Owner, repo.Name, pr.GetNumber(), &github.PullRequest{
			Title: github.String(title),
			Body:  github.String(body),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update existing PR: %w", err)
		}
		return pr, nil
	}

	// Create new PR
	pr, _, err := c.client.PullRequests.Create(ctx, repo.Owner, repo.Name, &github.NewPullRequest{
		Title: github.String(title),
		Body:  github.String(body),
		Head:  github.String(head),
		Base:  github.String(base),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	// Add labels if specified
	if len(labels) > 0 {
		_, _, err = c.client.Issues.AddLabelsToIssue(ctx, repo.Owner, repo.Name, pr.GetNumber(), labels)
		if err != nil {
			// Non-fatal, just log
			fmt.Printf("Warning: failed to add labels to PR: %v\n", err)
		}
	}

	return pr, nil
}

// GetRawClient returns the underlying GitHub client for advanced operations
func (c *Client) GetRawClient() *github.Client {
	return c.client
}
