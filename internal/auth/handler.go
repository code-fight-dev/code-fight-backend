package auth

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service          *Service
	oauthClient      *OAuthClient
	frontendBaseURL  string
	cookieName       string
	oauthStateCookie string
	sessionTTL       time.Duration
	oauthStateTTL    time.Duration
	secureCookies    bool
}

type authResponse struct {
	User PublicUser `json:"user"`
}

func NewHandler(service *Service, oauthClient *OAuthClient, frontendBaseURL string, cookieName string, oauthStateCookie string, sessionTTL time.Duration, oauthStateTTL time.Duration, secureCookies bool) *Handler {
	return &Handler{
		service:          service,
		oauthClient:      oauthClient,
		frontendBaseURL:  strings.TrimRight(frontendBaseURL, "/"),
		cookieName:       cookieName,
		oauthStateCookie: oauthStateCookie,
		sessionTTL:       sessionTTL,
		oauthStateTTL:    oauthStateTTL,
		secureCookies:    secureCookies,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/sign-up", h.signUp)
	router.Post("/sign-in", h.signIn)
	router.Post("/sign-out", h.signOut)
	router.Get("/me", h.me)
	router.Get("/oauth/:provider/start", h.startOAuth)
	router.Get("/oauth/:provider/callback", h.completeOAuth)
}

func (h *Handler) signUp(c *fiber.Ctx) error {
	var input SignUpInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	result, err := h.service.SignUp(c.UserContext(), input, SessionMeta{
		UserAgent: c.Get(fiber.HeaderUserAgent),
		IPAddress: c.IP(),
	})
	if err != nil {
		return mapServiceError(err)
	}

	h.setSessionCookie(c, result.SessionToken, result.ExpiresAt)

	return c.Status(fiber.StatusCreated).JSON(authResponse{User: result.User})
}

func (h *Handler) signIn(c *fiber.Ctx) error {
	var input SignInInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	result, err := h.service.SignIn(c.UserContext(), input, SessionMeta{
		UserAgent: c.Get(fiber.HeaderUserAgent),
		IPAddress: c.IP(),
	})
	if err != nil {
		return mapServiceError(err)
	}

	h.setSessionCookie(c, result.SessionToken, result.ExpiresAt)

	return c.JSON(authResponse{User: result.User})
}

func (h *Handler) signOut(c *fiber.Ctx) error {
	if err := h.service.SignOut(c.UserContext(), c.Cookies(h.cookieName)); err != nil {
		return mapServiceError(err)
	}

	h.clearSessionCookie(c)

	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) me(c *fiber.Ctx) error {
	user, err := h.service.CurrentUser(c.UserContext(), c.Cookies(h.cookieName))
	if err != nil {
		return mapServiceError(err)
	}

	return c.JSON(authResponse{User: user})
}

func (h *Handler) startOAuth(c *fiber.Ctx) error {
	provider, err := ParseOAuthProvider(c.Params("provider"))
	if err != nil {
		return c.Redirect(h.oauthErrorRedirect(h.authMode(c), "oauth_provider_invalid"), fiber.StatusTemporaryRedirect)
	}

	state, _, err := generateSessionToken()
	if err != nil {
		return c.Redirect(h.oauthErrorRedirect(h.authMode(c), "oauth_start_failed"), fiber.StatusTemporaryRedirect)
	}

	redirectURL, err := h.oauthClient.AuthorizationURL(provider, state)
	if err != nil {
		if errors.Is(err, ErrOAuthUnavailable) {
			return c.Redirect(h.oauthErrorRedirect(h.authMode(c), "oauth_provider_unavailable"), fiber.StatusTemporaryRedirect)
		}

		return c.Redirect(h.oauthErrorRedirect(h.authMode(c), "oauth_start_failed"), fiber.StatusTemporaryRedirect)
	}

	h.setOAuthStateCookie(c, provider, h.authMode(c), state)

	return c.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

func (h *Handler) completeOAuth(c *fiber.Ctx) error {
	provider, err := ParseOAuthProvider(c.Params("provider"))
	if err != nil {
		return c.Redirect(h.oauthErrorRedirect("signin", "oauth_provider_invalid"), fiber.StatusTemporaryRedirect)
	}

	mode, expectedState, err := h.readOAuthStateCookie(c, provider)
	h.clearOAuthStateCookie(c)
	if err != nil {
		return c.Redirect(h.oauthErrorRedirect("signin", "oauth_state_invalid"), fiber.StatusTemporaryRedirect)
	}

	if c.Query("state") == "" || c.Query("state") != expectedState {
		return c.Redirect(h.oauthErrorRedirect(mode, "oauth_state_invalid"), fiber.StatusTemporaryRedirect)
	}

	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		return c.Redirect(h.oauthErrorRedirect(mode, "oauth_code_missing"), fiber.StatusTemporaryRedirect)
	}

	identity, err := h.oauthClient.ExchangeCode(c.UserContext(), provider, code)
	if err != nil {
		return c.Redirect(h.oauthErrorRedirect(mode, "oauth_exchange_failed"), fiber.StatusTemporaryRedirect)
	}

	result, err := h.service.SignInWithOAuth(c.UserContext(), identity, SessionMeta{
		UserAgent: c.Get(fiber.HeaderUserAgent),
		IPAddress: c.IP(),
	})
	if err != nil {
		if errors.Is(err, ErrOAuthEmailRequired) {
			return c.Redirect(h.oauthErrorRedirect(mode, "oauth_email_required"), fiber.StatusTemporaryRedirect)
		}

		return c.Redirect(h.oauthErrorRedirect(mode, "oauth_auth_failed"), fiber.StatusTemporaryRedirect)
	}

	h.setSessionCookie(c, result.SessionToken, result.ExpiresAt)

	return c.Redirect(h.frontendBaseURL+"/", fiber.StatusTemporaryRedirect)
}

func (h *Handler) setSessionCookie(c *fiber.Ctx, token string, expiresAt time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     h.cookieName,
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		SameSite: "lax",
		Secure:   h.secureCookies,
		Expires:  expiresAt,
		MaxAge:   int(h.sessionTTL.Seconds()),
	})
}

func (h *Handler) setOAuthStateCookie(c *fiber.Ctx, provider OAuthProvider, mode string, state string) {
	c.Cookie(&fiber.Cookie{
		Name:     h.oauthStateCookie,
		Value:    string(provider) + "|" + mode + "|" + state,
		Path:     "/",
		HTTPOnly: true,
		SameSite: "lax",
		Secure:   h.secureCookies,
		Expires:  time.Now().Add(h.oauthStateTTL),
		MaxAge:   int(h.oauthStateTTL.Seconds()),
	})
}

func (h *Handler) clearOAuthStateCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     h.oauthStateCookie,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		SameSite: "lax",
		Secure:   h.secureCookies,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func (h *Handler) readOAuthStateCookie(c *fiber.Ctx, provider OAuthProvider) (string, string, error) {
	raw := strings.TrimSpace(c.Cookies(h.oauthStateCookie))
	parts := strings.Split(raw, "|")
	if len(parts) != 3 {
		return "", "", ErrOAuthStateInvalid
	}
	if parts[0] != string(provider) {
		return "", "", ErrOAuthStateInvalid
	}

	mode := parts[1]
	if mode != "signin" && mode != "signup" {
		mode = "signin"
	}

	return mode, parts[2], nil
}

func (h *Handler) clearSessionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     h.cookieName,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		SameSite: "lax",
		Secure:   h.secureCookies,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func (h *Handler) authMode(c *fiber.Ctx) string {
	mode := strings.TrimSpace(c.Query("mode"))
	if mode == "signup" {
		return "signup"
	}

	return "signin"
}

func (h *Handler) oauthErrorRedirect(mode string, errorCode string) string {
	values := url.Values{}
	values.Set("error", errorCode)

	path := "/signin"
	if mode == "signup" {
		path = "/signup"
	}

	return h.frontendBaseURL + path + "?" + values.Encode()
}

func mapServiceError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, ErrEmailTaken), errors.Is(err, ErrUsernameTaken):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, ErrOAuthUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrUnauthorized):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	default:
		return err
	}
}
