package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"code-fight-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestHandlerProfileByUsername(t *testing.T) {
	userID := uuid.New()
	repo := &fakeUserHandlerRepo{
		profile: Profile{ID: userID, Username: "ada", DisplayName: "Ada"},
	}
	app := fiber.New()
	NewHandler(NewService(repo, nil, nil), nil, "sid").RegisterRoutes(app)

	resp := userHandlerRequest(t, app, http.MethodGet, "/ada", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body profileResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Profile.ID != userID || body.Profile.Username != "ada" {
		t.Fatalf("body = %#v", body)
	}
}

func TestHandlerUpdateProfile(t *testing.T) {
	userID := uuid.New()
	repo := &fakeUserHandlerRepo{
		profile: Profile{ID: userID, Username: "ada", DisplayName: "Ada Lovelace"},
	}
	authReader := &fakeCurrentUserReader{user: auth.PublicUser{ID: userID, Username: "ada"}}
	app := fiber.New()
	NewHandler(NewService(repo, nil, nil), authReader, "sid").RegisterRoutes(app)

	resp := userHandlerRequest(
		t,
		app,
		http.MethodPatch,
		"/ada",
		`{"displayName":"Ada Lovelace","bio":"first programmer"}`,
		map[string]string{"Cookie": "sid=session-token"},
	)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if repo.updated.DisplayName != "Ada Lovelace" || repo.updated.Bio != "first programmer" {
		t.Fatalf("updated input = %#v", repo.updated)
	}
}

func TestHandlerUpdateProfileRejectsOtherUser(t *testing.T) {
	authReader := &fakeCurrentUserReader{user: auth.PublicUser{ID: uuid.New(), Username: "ada"}}
	app := fiber.New()
	NewHandler(NewService(&fakeUserHandlerRepo{}, nil, nil), authReader, "sid").RegisterRoutes(app)

	resp := userHandlerRequest(t, app, http.MethodPatch, "/grace", `{"displayName":"Grace"}`, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestHandlerUpdateProfileErrorBranches(t *testing.T) {
	userID := uuid.New()
	authReader := &fakeCurrentUserReader{user: auth.PublicUser{ID: userID, Username: "ada"}}
	app := fiber.New()
	NewHandler(NewService(&fakeUserHandlerRepo{updateErr: ErrInvalidInput}, nil, nil), authReader, "sid").RegisterRoutes(app)

	invalidBody := userHandlerRequest(t, app, http.MethodPatch, "/ada", `{`, nil)
	if invalidBody.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid body status = %d, want 400", invalidBody.StatusCode)
	}

	serviceErr := userHandlerRequest(t, app, http.MethodPatch, "/ada", `{"displayName":"Ada"}`, nil)
	if serviceErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("service error status = %d, want 400", serviceErr.StatusCode)
	}

	authApp := fiber.New()
	NewHandler(NewService(&fakeUserHandlerRepo{}, nil, nil), &fakeCurrentUserReader{err: auth.ErrUnauthorized}, "sid").RegisterRoutes(authApp)
	authResp := userHandlerRequest(t, authApp, http.MethodPatch, "/ada", `{"displayName":"Ada"}`, nil)
	if authResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("auth status = %d, want 401", authResp.StatusCode)
	}
}

func TestHandlerUpdateAvatar(t *testing.T) {
	userID := uuid.New()
	repo := &fakeUserHandlerRepo{
		profile: Profile{ID: userID, Username: "ada", DisplayName: "Ada", AvatarURL: "https://cdn/avatar.png"},
	}
	storage := &fakeAvatarStorage{url: "https://cdn/avatar.png"}
	authReader := &fakeCurrentUserReader{user: auth.PublicUser{ID: userID, Username: "ada"}}
	app := fiber.New()
	NewHandler(NewService(repo, storage, nil), authReader, "sid").RegisterRoutes(app)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	resp := userHandlerRequest(t, app, http.MethodPost, "/ada/avatar", body.String(), map[string]string{
		"Content-Type": writer.FormDataContentType(),
		"Cookie":       "sid=session-token",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if storage.contentType != "image/png" || repo.avatarURL != "https://cdn/avatar.png" {
		t.Fatalf("storage content type = %q, avatar url = %q", storage.contentType, repo.avatarURL)
	}
}

func TestHandlerUpdateAvatarRequiresFile(t *testing.T) {
	authReader := &fakeCurrentUserReader{user: auth.PublicUser{ID: uuid.New(), Username: "ada"}}
	app := fiber.New()
	NewHandler(NewService(&fakeUserHandlerRepo{}, &fakeAvatarStorage{}, nil), authReader, "sid").RegisterRoutes(app)

	resp := userHandlerRequest(t, app, http.MethodPost, "/ada/avatar", "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHandlerUpdateAvatarErrorBranches(t *testing.T) {
	userID := uuid.New()
	authReader := &fakeCurrentUserReader{user: auth.PublicUser{ID: userID, Username: "ada"}}
	app := fiber.New()
	NewHandler(NewService(&fakeUserHandlerRepo{avatarErr: ErrUnavailable}, &fakeAvatarStorage{url: "https://cdn/avatar.png"}, nil), authReader, "sid").RegisterRoutes(app)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	forbidden := userHandlerRequest(t, app, http.MethodPost, "/grace/avatar", body.String(), map[string]string{
		"Content-Type": writer.FormDataContentType(),
	})
	if forbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("forbidden status = %d, want 403", forbidden.StatusCode)
	}

	serviceErr := userHandlerRequest(t, app, http.MethodPost, "/ada/avatar", body.String(), map[string]string{
		"Content-Type": writer.FormDataContentType(),
	})
	if serviceErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("service error status = %d, want 503", serviceErr.StatusCode)
	}
}

func TestHandlerMapServiceError(t *testing.T) {
	app := fiber.New()
	app.Get("/invalid", func(c *fiber.Ctx) error { return mapServiceError(ErrInvalidInput) })
	app.Get("/missing", func(c *fiber.Ctx) error { return mapServiceError(ErrNotFound) })
	app.Get("/forbidden", func(c *fiber.Ctx) error { return mapServiceError(ErrForbidden) })
	app.Get("/unavailable", func(c *fiber.Ctx) error { return mapServiceError(ErrUnavailable) })
	app.Get("/unauthorized", func(c *fiber.Ctx) error { return mapServiceError(auth.ErrUnauthorized) })
	app.Get("/unknown", func(c *fiber.Ctx) error { return mapServiceError(errors.New("boom")) })

	tests := map[string]int{
		"/invalid":      http.StatusBadRequest,
		"/missing":      http.StatusNotFound,
		"/forbidden":    http.StatusForbidden,
		"/unavailable":  http.StatusServiceUnavailable,
		"/unauthorized": http.StatusUnauthorized,
		"/unknown":      http.StatusInternalServerError,
	}
	for path, want := range tests {
		resp := userHandlerRequest(t, app, http.MethodGet, path, "", nil)
		if resp.StatusCode != want {
			t.Fatalf("%s status = %d, want %d", path, resp.StatusCode, want)
		}
	}
}

func userHandlerRequest(t *testing.T, app *fiber.App, method, path, body string, headers map[string]string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" && headers["Content-Type"] == "" {
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

type fakeUserHandlerRepo struct {
	profile   Profile
	updated   UpdateProfileInput
	avatarURL string
	err       error
	updateErr error
	avatarErr error
}

func (r *fakeUserHandlerRepo) GetProfileByUsername(context.Context, string) (Profile, error) {
	if r.err != nil {
		return Profile{}, r.err
	}
	return r.profile, nil
}

func (r *fakeUserHandlerRepo) UpdateProfile(_ context.Context, _ uuid.UUID, _ string, input UpdateProfileInput) (Profile, error) {
	if r.updateErr != nil {
		return Profile{}, r.updateErr
	}
	r.updated = input
	return r.profile, nil
}

func (r *fakeUserHandlerRepo) GetAvatarObjectKey(context.Context, uuid.UUID) (string, error) {
	return "", nil
}

func (r *fakeUserHandlerRepo) UpdateCustomAvatar(_ context.Context, _ uuid.UUID, _ string, avatarURL string, _ string) (Profile, string, error) {
	if r.avatarErr != nil {
		return Profile{}, "", r.avatarErr
	}
	r.avatarURL = avatarURL
	return r.profile, "", nil
}

type fakeCurrentUserReader struct {
	user auth.PublicUser
	err  error
}

func (r *fakeCurrentUserReader) CurrentUser(context.Context, string) (auth.PublicUser, error) {
	if r.err != nil {
		return auth.PublicUser{}, r.err
	}
	if r.user.ID == uuid.Nil {
		return auth.PublicUser{}, auth.ErrUnauthorized
	}
	return r.user, nil
}
