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
	captchaVerifier  CaptchaVerifier
	frontendBaseURL  string
	cookieName       string
	cookieDomain     string
	cookieSameSite   string
	oauthStateCookie string
	sessionTTL       time.Duration
	oauthStateTTL    time.Duration
	secureCookies    bool
}

type authResponse struct {
	User PublicUser `json:"user"`
}

func NewHandler(service *Service, oauthClient *OAuthClient, captchaVerifier CaptchaVerifier, frontendBaseURL string, cookieName string, cookieDomain string, cookieSameSite string, oauthStateCookie string, sessionTTL time.Duration, oauthStateTTL time.Duration, secureCookies bool) *Handler {
	return &Handler{
		service:          service,
		oauthClient:      oauthClient,
		captchaVerifier:  captchaVerifier,
		frontendBaseURL:  strings.TrimRight(frontendBaseURL, "/"),
		cookieName:       cookieName,
		cookieDomain:     strings.TrimSpace(cookieDomain),
		cookieSameSite:   cookieSameSite,
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
	router.Post("/password-reset/request", h.requestPasswordReset)
	router.Post("/password-reset/confirm", h.confirmPasswordReset)
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

func (h *Handler) requestPasswordReset(c *fiber.Ctx) error {
	var input PasswordResetRequestInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if h.captchaVerifier != nil {
		if err := h.captchaVerifier.Verify(c.UserContext(), input.CaptchaToken, c.IP()); err != nil {
			return mapServiceError(err)
		}
	}

	if err := h.service.RequestPasswordReset(c.UserContext(), input); err != nil {
		return mapServiceError(err)
	}

	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) confirmPasswordReset(c *fiber.Ctx) error {
	var input PasswordResetConfirmInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.service.ConfirmPasswordReset(c.UserContext(), input); err != nil {
		return mapServiceError(err)
	}

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
	cookie := &fiber.Cookie{
		Name:     h.cookieName,
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		SameSite: h.cookieSameSite,
		Secure:   h.secureCookies,
		Expires:  expiresAt,
		MaxAge:   int(h.sessionTTL.Seconds()),
	}
	h.applyCookieDomain(cookie)
	c.Cookie(cookie)
}

func (h *Handler) setOAuthStateCookie(c *fiber.Ctx, provider OAuthProvider, mode string, state string) {
	cookie := &fiber.Cookie{
		Name:     h.oauthStateCookie,
		Value:    string(provider) + "|" + mode + "|" + state,
		Path:     "/",
		HTTPOnly: true,
		SameSite: h.cookieSameSite,
		Secure:   h.secureCookies,
		Expires:  time.Now().Add(h.oauthStateTTL),
		MaxAge:   int(h.oauthStateTTL.Seconds()),
	}
	h.applyCookieDomain(cookie)
	c.Cookie(cookie)
}

func (h *Handler) clearOAuthStateCookie(c *fiber.Ctx) {
	cookie := &fiber.Cookie{
		Name:     h.oauthStateCookie,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		SameSite: h.cookieSameSite,
		Secure:   h.secureCookies,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
	h.applyCookieDomain(cookie)
	c.Cookie(cookie)
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
	cookie := &fiber.Cookie{
		Name:     h.cookieName,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		SameSite: h.cookieSameSite,
		Secure:   h.secureCookies,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
	h.applyCookieDomain(cookie)
	c.Cookie(cookie)
}

func (h *Handler) applyCookieDomain(cookie *fiber.Cookie) {
	if h.cookieDomain != "" {
		cookie.Domain = h.cookieDomain
	}
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
	case errors.Is(err, ErrCaptchaUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrUnauthorized):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrCaptchaInvalid):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, ErrResetCodeInvalid), errors.Is(err, ErrResetCodeExpired):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	default:
		return err
	}
}

func (h *Handler) RequireAuth(c *fiber.Ctx) error {
	token := c.Cookies(h.cookieName)
	if strings.TrimSpace(token) == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	user, err := h.service.CurrentUser(c.UserContext(), token)
	if err != nil {
		return mapServiceError(err)
	}

	c.Locals("user_id", user.ID)
	c.Locals("user", user)

	return c.Next()
}
