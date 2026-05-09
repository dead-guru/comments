package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHubUser struct {
	ID        int64   `json:"id"`
	Login     string  `json:"login"`
	Email     *string `json:"email"`
	Name      *string `json:"name"`
	AvatarURL *string `json:"avatar_url"`
}

type GitHubOAuth struct {
	config *oauth2.Config
}

func NewGitHubOAuth(clientID, clientSecret, redirectURL string) *GitHubOAuth {
	return &GitHubOAuth{config: &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     github.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       []string{"read:user", "user:email"},
	}}
}

func (g *GitHubOAuth) AuthCodeURL(state string) string {
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (g *GitHubOAuth) Configured() bool {
	return g.config.ClientID != "" && g.config.ClientSecret != ""
}

func (g *GitHubOAuth) ExchangeUser(ctx context.Context, code string) (*GitHubUser, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	client := g.config.Client(ctx, token)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("github user request failed")
	}
	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}
