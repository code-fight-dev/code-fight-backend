package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type OAuthClient struct {
	backendBaseURL string
	httpClient     *http.Client
	githubClientID string
	githubSecret   string
	googleClientID string
	googleSecret   string
}

func NewOAuthClient(backendBaseURL string, githubClientID string, githubSecret string, googleClientID string, googleSecret string) *OAuthClient {
	return &OAuthClient{
		backendBaseURL: strings.TrimRight(strings.TrimSpace(backendBaseURL), "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		githubClientID: strings.TrimSpace(githubClientID),
		githubSecret:   strings.TrimSpace(githubSecret),
		googleClientID: strings.TrimSpace(googleClientID),
		googleSecret:   strings.TrimSpace(googleSecret),
	}
}

func (c *OAuthClient) AuthorizationURL(provider OAuthProvider, state string) (string, error) {
	switch provider {
	case OAuthProviderGitHub:
		if c.githubClientID == "" || c.githubSecret == "" {
			return "", ErrOAuthUnavailable
		}

		values := url.Values{}
		values.Set("client_id", c.githubClientID)
		values.Set("redirect_uri", c.callbackURL(provider))
		values.Set("scope", "read:user user:email")
		values.Set("state", state)

		return "https://github.com/login/oauth/authorize?" + values.Encode(), nil
	case OAuthProviderGoogle:
		if c.googleClientID == "" || c.googleSecret == "" {
			return "", ErrOAuthUnavailable
		}

		values := url.Values{}
		values.Set("client_id", c.googleClientID)
		values.Set("redirect_uri", c.callbackURL(provider))
		values.Set("response_type", "code")
		values.Set("scope", "openid email profile")
		values.Set("state", state)
		values.Set("prompt", "select_account")

		return "https://accounts.google.com/o/oauth2/v2/auth?" + values.Encode(), nil
	default:
		return "", ErrInvalidInput
	}
}

func (c *OAuthClient) ExchangeCode(ctx context.Context, provider OAuthProvider, code string) (OAuthIdentity, error) {
	switch provider {
	case OAuthProviderGitHub:
		return c.exchangeGitHub(ctx, code)
	case OAuthProviderGoogle:
		return c.exchangeGoogle(ctx, code)
	default:
		return OAuthIdentity{}, ErrInvalidInput
	}
}

func (c *OAuthClient) callbackURL(provider OAuthProvider) string {
	return c.backendBaseURL + "/auth/oauth/" + string(provider) + "/callback"
}

func (c *OAuthClient) exchangeGitHub(ctx context.Context, code string) (OAuthIdentity, error) {
	values := url.Values{}
	values.Set("client_id", c.githubClientID)
	values.Set("client_secret", c.githubSecret)
	values.Set("code", code)
	values.Set("redirect_uri", c.callbackURL(OAuthProviderGitHub))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(values.Encode()))
	if err != nil {
		return OAuthIdentity{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.doJSON(req, &tokenResponse); err != nil {
		return OAuthIdentity{}, err
	}
	if tokenResponse.AccessToken == "" {
		return OAuthIdentity{}, ErrUnauthorized
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return OAuthIdentity{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+tokenResponse.AccessToken)

	var githubUser struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := c.doJSON(req, &githubUser); err != nil {
		return OAuthIdentity{}, err
	}

	email := strings.TrimSpace(githubUser.Email)
	verified := email != ""

	if email == "" {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
		if err != nil {
			return OAuthIdentity{}, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", "Bearer "+tokenResponse.AccessToken)

		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := c.doJSON(req, &emails); err != nil {
			return OAuthIdentity{}, err
		}

		for _, entry := range emails {
			if entry.Primary && entry.Verified {
				email = strings.TrimSpace(entry.Email)
				verified = true
				break
			}
		}

		if email == "" {
			for _, entry := range emails {
				if entry.Verified {
					email = strings.TrimSpace(entry.Email)
					verified = true
					break
				}
			}
		}
	}

	return OAuthIdentity{
		Provider:       OAuthProviderGitHub,
		ProviderUserID: strconv.FormatInt(githubUser.ID, 10),
		Email:          strings.ToLower(email),
		EmailVerified:  verified,
		Username:       githubUser.Login,
		DisplayName:    githubUser.Name,
	}, nil
}

func (c *OAuthClient) exchangeGoogle(ctx context.Context, code string) (OAuthIdentity, error) {
	values := url.Values{}
	values.Set("code", code)
	values.Set("client_id", c.googleClientID)
	values.Set("client_secret", c.googleSecret)
	values.Set("redirect_uri", c.callbackURL(OAuthProviderGoogle))
	values.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(values.Encode()))
	if err != nil {
		return OAuthIdentity{}, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.doJSON(req, &tokenResponse); err != nil {
		return OAuthIdentity{}, err
	}
	if tokenResponse.AccessToken == "" {
		return OAuthIdentity{}, ErrUnauthorized
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return OAuthIdentity{}, err
	}
	req.Header.Set("Authorization", "Bearer "+tokenResponse.AccessToken)

	var googleUser struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
	}
	if err := c.doJSON(req, &googleUser); err != nil {
		return OAuthIdentity{}, err
	}

	username := googleUser.GivenName
	if username == "" {
		username = strings.Split(strings.TrimSpace(googleUser.Email), "@")[0]
	}

	return OAuthIdentity{
		Provider:       OAuthProviderGoogle,
		ProviderUserID: googleUser.Sub,
		Email:          strings.ToLower(strings.TrimSpace(googleUser.Email)),
		EmailVerified:  googleUser.EmailVerified,
		Username:       username,
		DisplayName:    googleUser.Name,
	}, nil
}

func (c *OAuthClient) doJSON(req *http.Request, destination any) error {
	response, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("oauth request failed: status=%d body=%s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		return fmt.Errorf("decode oauth response: %w", err)
	}

	return nil
}
