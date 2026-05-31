package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestOAuthClientAuthorizationURL(t *testing.T) {
	client := NewOAuthClient(" https://api.example.com/ ", "gh-id", "gh-secret", "google-id", "google-secret")

	githubURL, err := client.AuthorizationURL(OAuthProviderGitHub, "state-1")
	if err != nil {
		t.Fatalf("github AuthorizationURL returned error: %v", err)
	}
	parsed, err := url.Parse(githubURL)
	if err != nil {
		t.Fatalf("parse github URL: %v", err)
	}
	if parsed.Query().Get("client_id") != "gh-id" ||
		parsed.Query().Get("redirect_uri") != "https://api.example.com/auth/oauth/github/callback" ||
		parsed.Query().Get("state") != "state-1" {
		t.Fatalf("github URL = %s", githubURL)
	}

	googleURL, err := client.AuthorizationURL(OAuthProviderGoogle, "state-2")
	if err != nil {
		t.Fatalf("google AuthorizationURL returned error: %v", err)
	}
	parsed, err = url.Parse(googleURL)
	if err != nil {
		t.Fatalf("parse google URL: %v", err)
	}
	if parsed.Query().Get("client_id") != "google-id" ||
		parsed.Query().Get("response_type") != "code" ||
		parsed.Query().Get("state") != "state-2" {
		t.Fatalf("google URL = %s", googleURL)
	}

	unconfigured := NewOAuthClient("https://api.example.com", "", "", "", "")
	if _, err := unconfigured.AuthorizationURL(OAuthProviderGitHub, "state"); !errors.Is(err, ErrOAuthUnavailable) {
		t.Fatalf("unconfigured github error = %v, want ErrOAuthUnavailable", err)
	}
	if _, err := unconfigured.AuthorizationURL(OAuthProviderGoogle, "state"); !errors.Is(err, ErrOAuthUnavailable) {
		t.Fatalf("unconfigured google error = %v, want ErrOAuthUnavailable", err)
	}
}

func TestOAuthClientExchangeGitHubUsesPrimaryVerifiedEmail(t *testing.T) {
	client := NewOAuthClient("https://api.example.com", "gh-id", "gh-secret", "", "")
	client.httpClient = &http.Client{Transport: fakeOAuthTransport{
		responses: map[string]string{
			githubAccessTokenURL: `{"access_token":"gh-token"}`,
			githubUserURL:        `{"id":42,"login":"octo","email":"","name":"Octo Cat","avatar_url":"https://avatar"}`,
			githubEmailsURL:      `[{"email":"fallback@example.com","verified":true},{"email":"primary@example.com","primary":true,"verified":true}]`,
		},
	}}

	identity, err := client.ExchangeCode(context.Background(), OAuthProviderGitHub, "code")
	if err != nil {
		t.Fatalf("ExchangeCode returned error: %v", err)
	}
	if identity.Provider != OAuthProviderGitHub ||
		identity.ProviderUserID != "42" ||
		identity.Email != "primary@example.com" ||
		!identity.EmailVerified ||
		identity.Username != "octo" {
		t.Fatalf("identity = %#v", identity)
	}
}

func TestOAuthClientExchangeGoogle(t *testing.T) {
	client := NewOAuthClient("https://api.example.com", "", "", "google-id", "google-secret")
	client.httpClient = &http.Client{Transport: fakeOAuthTransport{
		responses: map[string]string{
			googleTokenURL:    `{"access_token":"google-token"}`,
			googleUserInfoURL: `{"sub":"sub-1","email":"Ada@Example.COM","email_verified":true,"name":"Ada Lovelace","given_name":"","picture":" https://avatar "}`,
		},
	}}

	identity, err := client.ExchangeCode(context.Background(), OAuthProviderGoogle, "code")
	if err != nil {
		t.Fatalf("ExchangeCode returned error: %v", err)
	}
	if identity.Provider != OAuthProviderGoogle ||
		identity.ProviderUserID != "sub-1" ||
		identity.Email != "ada@example.com" ||
		identity.Username != "Ada" ||
		identity.AvatarURL != "https://avatar" {
		t.Fatalf("identity = %#v", identity)
	}
}

func TestOAuthClientExchangeCodeErrors(t *testing.T) {
	client := NewOAuthClient("https://api.example.com", "gh-id", "gh-secret", "", "")
	client.httpClient = &http.Client{Transport: fakeOAuthTransport{
		responses: map[string]string{githubAccessTokenURL: `{}`},
	}}
	if _, err := client.ExchangeCode(context.Background(), OAuthProviderGitHub, "code"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("empty token error = %v, want ErrUnauthorized", err)
	}

	client.httpClient = &http.Client{Transport: fakeOAuthTransport{status: http.StatusBadGateway}}
	if _, err := client.ExchangeCode(context.Background(), OAuthProviderGitHub, "code"); err == nil {
		t.Fatal("non-2xx exchange returned nil error")
	}

	client.httpClient = &http.Client{Transport: fakeOAuthTransport{err: errors.New("network down")}}
	if _, err := client.ExchangeCode(context.Background(), OAuthProviderGitHub, "code"); err == nil {
		t.Fatal("transport error returned nil error")
	}

	client.httpClient = &http.Client{Transport: fakeOAuthTransport{responses: map[string]string{githubAccessTokenURL: `{`}}}
	if _, err := client.ExchangeCode(context.Background(), OAuthProviderGitHub, "code"); err == nil {
		t.Fatal("invalid token json returned nil error")
	}

	client.httpClient = &http.Client{Transport: fakeOAuthTransport{
		responses: map[string]string{
			githubAccessTokenURL: `{"access_token":"token"}`,
			githubUserURL:        `{"id":42,"login":"octo","email":"","name":"Octo"}`,
			githubEmailsURL:      `{`,
		},
	}}
	if _, err := client.ExchangeCode(context.Background(), OAuthProviderGitHub, "code"); err == nil {
		t.Fatal("invalid github emails json returned nil error")
	}

	google := NewOAuthClient("https://api.example.com", "", "", "google-id", "google-secret")
	google.httpClient = &http.Client{Transport: fakeOAuthTransport{responses: map[string]string{googleTokenURL: `{}`}}}
	if _, err := google.ExchangeCode(context.Background(), OAuthProviderGoogle, "code"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("empty google token error = %v, want ErrUnauthorized", err)
	}

	if _, err := client.ExchangeCode(context.Background(), OAuthProvider("bad"), "code"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid provider error = %v, want ErrInvalidInput", err)
	}
}

func TestOAuthHelpers(t *testing.T) {
	req, err := newOAuthFormRequest(context.Background(), http.MethodPost, "https://example.com", url.Values{"code": {"abc"}})
	if err != nil {
		t.Fatalf("newOAuthFormRequest returned error: %v", err)
	}
	if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Fatalf("content type = %q", req.Header.Get("Content-Type"))
	}

	setBearerAuthorization(req, "token")
	if req.Header.Get("Authorization") != "Bearer token" {
		t.Fatalf("authorization = %q", req.Header.Get("Authorization"))
	}
}

type fakeOAuthTransport struct {
	responses map[string]string
	status    int
	err       error
}

func (t fakeOAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.err != nil {
		return nil, t.err
	}
	status := t.status
	if status == 0 {
		status = http.StatusOK
	}
	body := t.responses[req.URL.String()]
	if body == "" {
		body = `{}`
	}
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}
