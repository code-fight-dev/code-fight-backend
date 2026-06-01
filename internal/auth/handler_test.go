package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestHandlerSignUpAndMe(t *testing.T) {
	repo := newFakeAuthRepo()
	app := authHandlerApp(NewService(repo, time.Hour, nil), nil)

	signUpResp := authHandlerRequest(t, app, http.MethodPost, "/sign-up", `{
		"username":"player",
		"email":"player@example.com",
		"password":"password123"
	}`, nil)
	if signUpResp.StatusCode != http.StatusCreated {
		t.Fatalf("sign-up status = %d, want 201", signUpResp.StatusCode)
	}

	sessionCookie := authHandlerCookie(signUpResp, "sid")
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("session cookie = %#v", sessionCookie)
	}

	meResp := authHandlerRequest(t, app, http.MethodGet, "/me", "", map[string]string{
		"Cookie": "sid=" + sessionCookie.Value,
	})
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d, want 200", meResp.StatusCode)
	}

	var body authResponse
	if err := json.NewDecoder(meResp.Body).Decode(&body); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if body.User.Username != "player" {
		t.Fatalf("me body = %#v", body)
	}
}

func TestHandlerSignInAndSignOut(t *testing.T) {
	repo := newFakeAuthRepo()
	hash, err := hashPassword("password123")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	repo.storeUser(User{ID: uuid.New(), Username: "player", Email: "player@example.com", PasswordHash: hash})
	app := authHandlerApp(NewService(repo, time.Hour, nil), nil)

	signInResp := authHandlerRequest(t, app, http.MethodPost, "/sign-in", `{
		"email":"player@example.com",
		"password":"password123"
	}`, nil)
	if signInResp.StatusCode != http.StatusOK {
		t.Fatalf("sign-in status = %d, want 200", signInResp.StatusCode)
	}

	sessionCookie := authHandlerCookie(signInResp, "sid")
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("session cookie = %#v", sessionCookie)
	}

	signOutResp := authHandlerRequest(t, app, http.MethodPost, "/sign-out", "", map[string]string{
		"Cookie": "sid=" + sessionCookie.Value,
	})
	if signOutResp.StatusCode != http.StatusOK {
		t.Fatalf("sign-out status = %d, want 200", signOutResp.StatusCode)
	}
	if len(repo.sessionsByHash) != 0 {
		t.Fatalf("sessions = %d, want 0", len(repo.sessionsByHash))
	}
}

func TestHandlerPasswordResetRequestVerifiesCaptcha(t *testing.T) {
	repo := newFakeAuthRepo()
	repo.storeUser(User{ID: uuid.New(), Username: "player", Email: "player@example.com"})
	mailer := &fakeMailer{}
	captcha := &fakeHandlerCaptcha{}
	app := authHandlerApp(NewService(repo, time.Hour, mailer), captcha)

	resp := authHandlerRequest(t, app, http.MethodPost, "/password-reset/request", `{
		"email":"player@example.com",
		"captchaToken":"captcha-token"
	}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if captcha.token != "captcha-token" || mailer.code == "" {
		t.Fatalf("captcha = %#v, mailer = %#v", captcha, mailer)
	}
}

func TestHandlerConfirmPasswordResetMapsInvalidCode(t *testing.T) {
	repo := newFakeAuthRepo()
	repo.storeUser(User{ID: uuid.New(), Username: "player", Email: "player@example.com"})
	app := authHandlerApp(NewService(repo, time.Hour, nil), nil)

	resp := authHandlerRequest(t, app, http.MethodPost, "/password-reset/confirm", `{
		"email":"player@example.com",
		"code":"000000",
		"password":"new-password"
	}`, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestHandlerRejectsInvalidBodiesAndServiceErrors(t *testing.T) {
	repo := newFakeAuthRepo()
	captcha := &fakeHandlerCaptcha{err: ErrCaptchaInvalid}
	app := authHandlerApp(NewService(repo, time.Hour, nil), captcha)

	tests := []struct {
		method  string
		path    string
		body    string
		headers map[string]string
		want    int
	}{
		{http.MethodPost, "/sign-up", `{`, nil, http.StatusBadRequest},
		{http.MethodPost, "/sign-in", `{`, nil, http.StatusBadRequest},
		{http.MethodPost, "/password-reset/request", `{`, nil, http.StatusBadRequest},
		{http.MethodPost, "/password-reset/request", `{"email":"player@example.com","captchaToken":"bad"}`, nil, http.StatusBadRequest},
		{http.MethodPost, "/password-reset/confirm", `{`, nil, http.StatusBadRequest},
		{http.MethodGet, "/me", "", nil, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		resp := authHandlerRequest(t, app, tt.method, tt.path, tt.body, tt.headers)
		if resp.StatusCode != tt.want {
			t.Fatalf("%s %s status = %d, want %d", tt.method, tt.path, resp.StatusCode, tt.want)
		}
	}
}

func TestHandlerRequireAuth(t *testing.T) {
	repo := newFakeAuthRepo()
	user := repo.storeUser(User{ID: uuid.New(), Username: "player", Email: "player@example.com"})
	token := "plain-token"
	repo.sessionsByHash[hashSessionToken(token)] = Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashSessionToken(token),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	handler := authTestHandler(NewService(repo, time.Hour, nil), nil)
	app := fiber.New()
	app.Get("/private", handler.RequireAuth, func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"userId": c.Locals("user_id")})
	})

	unauthorized := authHandlerRequest(t, app, http.MethodGet, "/private", "", nil)
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want 401", unauthorized.StatusCode)
	}

	authorized := authHandlerRequest(t, app, http.MethodGet, "/private", "", map[string]string{"Cookie": "sid=" + token})
	if authorized.StatusCode != http.StatusOK {
		t.Fatalf("authorized status = %d, want 200", authorized.StatusCode)
	}
}

func TestHandlerCookiesUseDomainAndRequireAuthMapsServiceError(t *testing.T) {
	repo := newFakeAuthRepo()
	user := repo.storeUser(User{ID: uuid.New(), Username: "player", Email: "player@example.com"})
	token := "plain-token"
	repo.sessionsByHash[hashSessionToken(token)] = Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashSessionToken(token),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	handler := NewHandler(
		NewService(repo, time.Hour, nil),
		nil,
		nil,
		"http://frontend.test",
		"sid",
		".example.test",
		"Strict",
		"oauth_state",
		time.Hour,
		5*time.Minute,
		true,
	)
	app := fiber.New()
	handler.RegisterRoutes(app)
	app.Get("/private", handler.RequireAuth, func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	})

	signOutResp := authHandlerRequest(t, app, http.MethodPost, "/sign-out", "", map[string]string{
		"Cookie": "sid=" + token,
	})
	if signOutResp.StatusCode != http.StatusOK {
		t.Fatalf("sign-out status = %d, want 200", signOutResp.StatusCode)
	}
	if cookie := authHandlerCookie(signOutResp, "sid"); cookie == nil || cookie.Domain != ".example.test" {
		t.Fatalf("clear cookie = %#v", cookie)
	}

	privateResp := authHandlerRequest(t, app, http.MethodGet, "/private", "", map[string]string{
		"Cookie": "sid=" + token,
	})
	if privateResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("private status = %d, want 401", privateResp.StatusCode)
	}
}

func TestHandlerStartOAuthRejectsInvalidProvider(t *testing.T) {
	app := authHandlerApp(NewService(newFakeAuthRepo(), time.Hour, nil), nil)

	resp := authHandlerRequest(t, app, http.MethodGet, "/oauth/unknown/start?mode=signup", "", nil)
	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want 307", resp.StatusCode)
	}
	if location := resp.Header.Get("Location"); !strings.Contains(location, "/signup?error=oauth_provider_invalid") {
		t.Fatalf("location = %q", location)
	}
}

func TestHandlerStartOAuthMapsUnavailableProvider(t *testing.T) {
	oauthClient := NewOAuthClient("https://backend.test", "", "", "", "")
	handler := NewHandler(
		NewService(newFakeAuthRepo(), time.Hour, nil),
		oauthClient,
		nil,
		"http://frontend.test",
		"sid",
		"",
		"Lax",
		"oauth_state",
		time.Hour,
		5*time.Minute,
		false,
	)
	app := fiber.New()
	handler.RegisterRoutes(app)

	resp := authHandlerRequest(t, app, http.MethodGet, "/oauth/github/start", "", nil)
	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want 307", resp.StatusCode)
	}
	if location := resp.Header.Get("Location"); !strings.Contains(location, "oauth_provider_unavailable") {
		t.Fatalf("location = %q", location)
	}
}

func TestHandlerOAuthStartAndCallback(t *testing.T) {
	repo := newFakeAuthRepo()
	service := NewService(repo, time.Hour, nil)
	oauthClient := NewOAuthClient("https://backend.test", "gh-id", "gh-secret", "", "")
	oauthClient.httpClient = &http.Client{Transport: fakeOAuthTransport{
		responses: map[string]string{
			githubAccessTokenURL: `{"access_token":"gh-token"}`,
			githubUserURL:        `{"id":42,"login":"octo","email":"octo@example.com","name":"Octo Cat","avatar_url":"https://avatar"}`,
		},
	}}
	handler := NewHandler(
		service,
		oauthClient,
		nil,
		"http://frontend.test",
		"sid",
		"",
		"Lax",
		"oauth_state",
		time.Hour,
		5*time.Minute,
		false,
	)
	app := fiber.New()
	handler.RegisterRoutes(app)

	startResp := authHandlerRequest(t, app, http.MethodGet, "/oauth/github/start?mode=signup", "", nil)
	if startResp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("start status = %d, want 307", startResp.StatusCode)
	}
	if location := startResp.Header.Get("Location"); !strings.Contains(location, "github.com/login/oauth/authorize") {
		t.Fatalf("start location = %q", location)
	}
	stateCookie := authHandlerCookie(startResp, "oauth_state")
	if stateCookie == nil || !strings.HasPrefix(stateCookie.Value, "github|signup|") {
		t.Fatalf("state cookie = %#v", stateCookie)
	}
	state := strings.TrimPrefix(stateCookie.Value, "github|signup|")

	callbackResp := authHandlerRequest(t, app, http.MethodGet, "/oauth/github/callback?state="+state+"&code=code", "", map[string]string{
		"Cookie": "oauth_state=" + stateCookie.Value,
	})
	if callbackResp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("callback status = %d, want 307", callbackResp.StatusCode)
	}
	if callbackResp.Header.Get("Location") != "http://frontend.test/" {
		t.Fatalf("callback location = %q", callbackResp.Header.Get("Location"))
	}
	sessionCookie := authHandlerCookie(callbackResp, "sid")
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("session cookie = %#v", sessionCookie)
	}
}

func TestHandlerOAuthCallbackErrorBranches(t *testing.T) {
	oauthClient := NewOAuthClient("https://backend.test", "gh-id", "gh-secret", "", "")
	oauthClient.httpClient = &http.Client{Transport: fakeOAuthTransport{status: http.StatusInternalServerError}}
	handler := NewHandler(
		NewService(newFakeAuthRepo(), time.Hour, nil),
		oauthClient,
		nil,
		"http://frontend.test",
		"sid",
		"",
		"Lax",
		"oauth_state",
		time.Hour,
		5*time.Minute,
		false,
	)
	app := fiber.New()
	handler.RegisterRoutes(app)

	tests := []struct {
		path       string
		cookie     string
		want       string
		statusCode int
	}{
		{"/oauth/unknown/callback", "", "oauth_provider_invalid", http.StatusTemporaryRedirect},
		{"/oauth/github/callback?state=bad&code=code", "oauth_state=github|signin|expected", "oauth_state_invalid", http.StatusTemporaryRedirect},
		{"/oauth/github/callback?state=expected", "oauth_state=github|weird|expected", "oauth_code_missing", http.StatusTemporaryRedirect},
		{"/oauth/github/callback?state=expected&code=code", "oauth_state=github|signin|expected", "oauth_exchange_failed", http.StatusTemporaryRedirect},
	}
	for _, tt := range tests {
		resp := authHandlerRequest(t, app, http.MethodGet, tt.path, "", map[string]string{
			"Cookie": tt.cookie,
		})
		if resp.StatusCode != tt.statusCode {
			t.Fatalf("%s status = %d, want %d", tt.path, resp.StatusCode, tt.statusCode)
		}
		if location := resp.Header.Get("Location"); !strings.Contains(location, tt.want) {
			t.Fatalf("%s location = %q, want %q", tt.path, location, tt.want)
		}
	}
}

func TestAuthMapServiceError(t *testing.T) {
	app := fiber.New()
	app.Get("/invalid", func(c *fiber.Ctx) error { return mapServiceError(ErrInvalidInput) })
	app.Get("/conflict", func(c *fiber.Ctx) error { return mapServiceError(ErrEmailTaken) })
	app.Get("/oauth", func(c *fiber.Ctx) error { return mapServiceError(ErrOAuthUnavailable) })
	app.Get("/captcha", func(c *fiber.Ctx) error { return mapServiceError(ErrCaptchaUnavailable) })
	app.Get("/credentials", func(c *fiber.Ctx) error { return mapServiceError(ErrInvalidCredentials) })
	app.Get("/bad-captcha", func(c *fiber.Ctx) error { return mapServiceError(ErrCaptchaInvalid) })
	app.Get("/expired-reset", func(c *fiber.Ctx) error { return mapServiceError(ErrResetCodeExpired) })

	tests := map[string]int{
		"/invalid":       http.StatusBadRequest,
		"/conflict":      http.StatusConflict,
		"/oauth":         http.StatusServiceUnavailable,
		"/captcha":       http.StatusServiceUnavailable,
		"/credentials":   http.StatusUnauthorized,
		"/bad-captcha":   http.StatusBadRequest,
		"/expired-reset": http.StatusUnauthorized,
	}
	for path, want := range tests {
		resp := authHandlerRequest(t, app, http.MethodGet, path, "", nil)
		if resp.StatusCode != want {
			t.Fatalf("%s status = %d, want %d", path, resp.StatusCode, want)
		}
	}
}

func authHandlerApp(service *Service, captcha CaptchaVerifier) *fiber.App {
	app := fiber.New()
	authTestHandler(service, captcha).RegisterRoutes(app)
	return app
}

func authTestHandler(service *Service, captcha CaptchaVerifier) *Handler {
	return NewHandler(
		service,
		nil,
		captcha,
		"http://frontend.test",
		"sid",
		"",
		"Lax",
		"oauth_state",
		time.Hour,
		5*time.Minute,
		false,
	)
}

func authHandlerRequest(t *testing.T, app *fiber.App, method, path, body string, headers map[string]string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	return resp
}

func authHandlerCookie(resp *http.Response, name string) *http.Cookie {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

type fakeHandlerCaptcha struct {
	token string
	ip    string
	err   error
}

func (c *fakeHandlerCaptcha) Verify(_ context.Context, token, ip string) error {
	c.token = token
	c.ip = ip
	return c.err
}
