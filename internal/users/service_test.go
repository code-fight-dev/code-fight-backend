package users

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
)

func TestServiceGetProfileByUsername(t *testing.T) {
	repo := &fakeUserRepo{profile: Profile{Username: "player"}}
	service := NewService(repo, nil, nil)

	profile, err := service.GetProfileByUsername(context.Background(), " player ")
	if err != nil {
		t.Fatalf("GetProfileByUsername returned error: %v", err)
	}
	if repo.username != "player" {
		t.Fatalf("repo username = %q", repo.username)
	}
	if profile.DisplayName != "player" {
		t.Fatalf("display name = %q", profile.DisplayName)
	}

	if _, err := service.GetProfileByUsername(context.Background(), " "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty username error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceUpdateProfile(t *testing.T) {
	viewerID := uuid.New()
	repo := &fakeUserRepo{
		profile:       Profile{Username: "player", DisplayName: "Ada"},
		avatarObject:  "avatars/old.png",
		oldAvatarSeen: true,
	}
	storage := &fakeAvatarStorage{}
	service := NewService(repo, storage, nil)

	profile, err := service.UpdateProfile(context.Background(), viewerID, " player ", UpdateProfileInput{
		DisplayName:        " Ada ",
		RemoveCustomAvatar: true,
	})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}
	if profile.DisplayName != "Ada" {
		t.Fatalf("profile = %#v", profile)
	}
	if storage.deleted != "avatars/old.png" {
		t.Fatalf("deleted avatar = %q", storage.deleted)
	}

	if _, err := service.UpdateProfile(context.Background(), uuid.Nil, "player", UpdateProfileInput{DisplayName: "Ada"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("nil viewer error = %v, want ErrForbidden", err)
	}
}

func TestServiceUpdateAvatar(t *testing.T) {
	viewerID := uuid.New()
	repo := &fakeUserRepo{profile: Profile{Username: "player", DisplayName: "Ada"}, oldAvatarObject: "avatars/old.png"}
	storage := &fakeAvatarStorage{url: "https://cdn.example.com/avatar.png"}
	metrics := &fakeAvatarMetrics{}
	service := NewService(repo, storage, metrics)

	profile, err := service.UpdateAvatar(context.Background(), viewerID, " player ", UpdateAvatarInput{
		Content:     []byte("png"),
		ContentType: "image/png",
	})
	if err != nil {
		t.Fatalf("UpdateAvatar returned error: %v", err)
	}
	if profile.DisplayName != "Ada" {
		t.Fatalf("profile = %#v", profile)
	}
	if !bytes.Equal(storage.uploaded, []byte("png")) || storage.contentType != "image/png" {
		t.Fatalf("upload = %q/%q", storage.uploaded, storage.contentType)
	}
	if storage.deleted != "avatars/old.png" {
		t.Fatalf("old avatar deleted = %q", storage.deleted)
	}
	if metrics.status != "success" {
		t.Fatalf("metric status = %q", metrics.status)
	}
}

func TestServiceUpdateAvatarErrors(t *testing.T) {
	viewerID := uuid.New()
	metrics := &fakeAvatarMetrics{}
	service := NewService(&fakeUserRepo{}, nil, metrics)
	if _, err := service.UpdateAvatar(context.Background(), viewerID, "player", UpdateAvatarInput{Content: []byte("x"), ContentType: "image/png"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing storage error = %v, want ErrUnavailable", err)
	}
	if metrics.status != "unavailable" {
		t.Fatalf("metric status = %q", metrics.status)
	}

	storage := &fakeAvatarStorage{url: "https://cdn.example.com/avatar.png"}
	service = NewService(&fakeUserRepo{}, storage, metrics)
	if _, err := service.UpdateAvatar(context.Background(), uuid.Nil, "player", UpdateAvatarInput{}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("nil viewer error = %v, want ErrForbidden", err)
	}
	if _, err := service.UpdateAvatar(context.Background(), viewerID, "player", UpdateAvatarInput{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid upload error = %v, want ErrInvalidInput", err)
	}

	storage = &fakeAvatarStorage{err: errors.New("put failed")}
	service = NewService(&fakeUserRepo{}, storage, metrics)
	if _, err := service.UpdateAvatar(context.Background(), viewerID, "player", UpdateAvatarInput{Content: []byte("png"), ContentType: "image/png"}); err == nil {
		t.Fatal("storage upload error returned nil")
	}
	if metrics.status != "storage_error" {
		t.Fatalf("metric status = %q, want storage_error", metrics.status)
	}

	repo := &fakeUserRepo{updateAvatarErr: errors.New("update failed")}
	storage = &fakeAvatarStorage{url: "https://cdn.example.com/avatar.png"}
	service = NewService(repo, storage, metrics)
	if _, err := service.UpdateAvatar(context.Background(), viewerID, "player", UpdateAvatarInput{Content: []byte("png"), ContentType: "image/png"}); !errors.Is(err, repo.updateAvatarErr) {
		t.Fatalf("repository avatar error = %v, want %v", err, repo.updateAvatarErr)
	}
	if storage.deleted == "" {
		t.Fatal("uploaded avatar was not deleted after repository error")
	}
	if metrics.status != "database_error" {
		t.Fatalf("metric status = %q, want database_error", metrics.status)
	}
}

type fakeUserRepo struct {
	profile         Profile
	username        string
	avatarObject    string
	oldAvatarObject string
	oldAvatarSeen   bool
	updateAvatarErr error
}

func (r *fakeUserRepo) GetProfileByUsername(_ context.Context, username string) (Profile, error) {
	r.username = username
	return r.profile, nil
}

func (r *fakeUserRepo) UpdateProfile(_ context.Context, _ uuid.UUID, username string, input UpdateProfileInput) (Profile, error) {
	r.username = username
	r.profile.DisplayName = input.DisplayName
	return r.profile, nil
}

func (r *fakeUserRepo) GetAvatarObjectKey(context.Context, uuid.UUID) (string, error) {
	return r.avatarObject, nil
}

func (r *fakeUserRepo) UpdateCustomAvatar(_ context.Context, _ uuid.UUID, username string, avatarURL string, avatarObjectKey string) (Profile, string, error) {
	if r.updateAvatarErr != nil {
		return Profile{}, "", r.updateAvatarErr
	}
	r.username = username
	r.profile.AvatarURL = avatarURL
	r.profile.AvatarSource = AvatarSourceCustom
	r.avatarObject = avatarObjectKey
	return r.profile, r.oldAvatarObject, nil
}

type fakeAvatarStorage struct {
	url         string
	uploaded    []byte
	contentType string
	deleted     string
	err         error
}

func (s *fakeAvatarStorage) Put(_ context.Context, _ string, body io.Reader, _ int64, contentType string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	raw, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	s.uploaded = raw
	s.contentType = contentType
	return s.url, nil
}

func (s *fakeAvatarStorage) Delete(_ context.Context, key string) error {
	s.deleted = key
	return nil
}

type fakeAvatarMetrics struct {
	status string
}

func (m *fakeAvatarMetrics) RecordAvatarUpload(status string) {
	m.status = status
}
