package users

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxDisplayNameLength   = 60
	maxBioLength           = 255
	maxCountryLength       = 80
	maxCountryCodeLength   = 2
	maxStateProvinceLength = 80
	maxCityLength          = 80
	maxAvatarDataURLLength = 3_000_000
	maxAvatarFileSize      = 2_000_000
)

type AvatarStorage interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) (string, error)
	Delete(ctx context.Context, key string) error
}

type AvatarMetrics interface {
	RecordAvatarUpload(status string)
}

var allowedAvatarDataURLPrefixes = []string{
	"data:image/jpeg;base64,",
	"data:image/jpg;base64,",
	"data:image/png;base64,",
	"data:image/gif;base64,",
}

var avatarFileExtensions = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
}

type Service struct {
	repository    UserRepository
	avatarStorage AvatarStorage
	avatarMetrics AvatarMetrics
}

type UserRepository interface {
	GetProfileByUsername(ctx context.Context, username string) (Profile, error)
	UpdateProfile(ctx context.Context, viewerID uuid.UUID, username string, input UpdateProfileInput) (Profile, error)
	GetAvatarObjectKey(ctx context.Context, userID uuid.UUID) (string, error)
	UpdateCustomAvatar(ctx context.Context, viewerID uuid.UUID, username string, avatarURL string, avatarObjectKey string) (Profile, string, error)
}

func NewService(repository UserRepository, avatarStorage AvatarStorage, avatarMetrics AvatarMetrics) *Service {
	return &Service{
		repository:    repository,
		avatarStorage: avatarStorage,
		avatarMetrics: avatarMetrics,
	}
}

func (s *Service) GetProfileByUsername(ctx context.Context, username string) (Profile, error) {
	normalizedUsername := strings.TrimSpace(username)
	if normalizedUsername == "" {
		return Profile{}, fmt.Errorf("%w: username is required", ErrInvalidInput)
	}

	profile, err := s.repository.GetProfileByUsername(ctx, normalizedUsername)
	if err != nil {
		return Profile{}, err
	}

	return applyProfileDefaults(profile), nil
}

func (s *Service) UpdateProfile(ctx context.Context, viewerID uuid.UUID, username string, input UpdateProfileInput) (Profile, error) {
	if viewerID == uuid.Nil {
		return Profile{}, fmt.Errorf("%w: viewer is required", ErrForbidden)
	}

	normalizedUsername := strings.TrimSpace(username)
	if normalizedUsername == "" {
		return Profile{}, fmt.Errorf("%w: username is required", ErrInvalidInput)
	}

	normalizedInput, err := normalizeUpdateProfileInput(input)
	if err != nil {
		return Profile{}, err
	}

	oldAvatarObjectKey := ""
	if normalizedInput.RemoveCustomAvatar {
		oldAvatarObjectKey, err = s.repository.GetAvatarObjectKey(ctx, viewerID)
		if err != nil {
			return Profile{}, err
		}
	}

	profile, err := s.repository.UpdateProfile(ctx, viewerID, normalizedUsername, normalizedInput)
	if err != nil {
		return Profile{}, err
	}
	if normalizedInput.RemoveCustomAvatar && s.avatarStorage != nil && oldAvatarObjectKey != "" {
		_ = s.avatarStorage.Delete(ctx, oldAvatarObjectKey)
	}

	return applyProfileDefaults(profile), nil
}

func (s *Service) UpdateAvatar(ctx context.Context, viewerID uuid.UUID, username string, input UpdateAvatarInput) (Profile, error) {
	if viewerID == uuid.Nil {
		s.recordAvatarUpload("forbidden")
		return Profile{}, fmt.Errorf("%w: viewer is required", ErrForbidden)
	}
	if s.avatarStorage == nil {
		s.recordAvatarUpload("unavailable")
		return Profile{}, fmt.Errorf("%w: avatar storage is not configured", ErrUnavailable)
	}

	normalizedUsername := strings.TrimSpace(username)
	if normalizedUsername == "" {
		return Profile{}, fmt.Errorf("%w: username is required", ErrInvalidInput)
	}

	contentType, extension, err := normalizeAvatarUpload(input)
	if err != nil {
		s.recordAvatarUpload("invalid")
		return Profile{}, err
	}

	key := fmt.Sprintf("avatars/%s/%s.%s", viewerID.String(), uuid.NewString(), extension)
	avatarURL, err := s.avatarStorage.Put(
		ctx,
		key,
		bytes.NewReader(input.Content),
		int64(len(input.Content)),
		contentType,
	)
	if err != nil {
		s.recordAvatarUpload("storage_error")
		return Profile{}, fmt.Errorf("upload avatar: %w", err)
	}

	profile, oldAvatarObjectKey, err := s.repository.UpdateCustomAvatar(
		ctx,
		viewerID,
		normalizedUsername,
		avatarURL,
		key,
	)
	if err != nil {
		_ = s.avatarStorage.Delete(ctx, key)
		s.recordAvatarUpload("database_error")
		return Profile{}, err
	}
	if oldAvatarObjectKey != "" && oldAvatarObjectKey != key {
		_ = s.avatarStorage.Delete(ctx, oldAvatarObjectKey)
	}

	s.recordAvatarUpload("success")
	return applyProfileDefaults(profile), nil
}

func (s *Service) recordAvatarUpload(status string) {
	if s.avatarMetrics != nil {
		s.avatarMetrics.RecordAvatarUpload(status)
	}
}

func normalizeAvatarUpload(input UpdateAvatarInput) (string, string, error) {
	if len(input.Content) == 0 {
		return "", "", fmt.Errorf("%w: avatar file is required", ErrInvalidInput)
	}
	if len(input.Content) > maxAvatarFileSize {
		return "", "", fmt.Errorf("%w: avatar file must be 2 MB or smaller", ErrInvalidInput)
	}

	contentType := strings.ToLower(strings.TrimSpace(input.ContentType))
	extension, ok := avatarFileExtensions[contentType]
	if !ok {
		return "", "", fmt.Errorf("%w: avatar must be jpg, jpeg, png, or gif", ErrInvalidInput)
	}

	return contentType, extension, nil
}

func normalizeUpdateProfileInput(input UpdateProfileInput) (UpdateProfileInput, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return UpdateProfileInput{}, fmt.Errorf("%w: display name is required", ErrInvalidInput)
	}
	if utf8.RuneCountInString(displayName) > maxDisplayNameLength {
		return UpdateProfileInput{}, fmt.Errorf(
			"%w: display name must be at most %d characters",
			ErrInvalidInput,
			maxDisplayNameLength,
		)
	}

	bio := strings.TrimSpace(input.Bio)
	if utf8.RuneCountInString(bio) > maxBioLength {
		return UpdateProfileInput{}, fmt.Errorf("%w: bio must be at most %d characters", ErrInvalidInput, maxBioLength)
	}

	country := strings.TrimSpace(input.Country)
	if utf8.RuneCountInString(country) > maxCountryLength {
		return UpdateProfileInput{}, fmt.Errorf("%w: country must be at most %d characters", ErrInvalidInput, maxCountryLength)
	}

	countryCode, err := normalizeCountryCode(input.CountryCode)
	if err != nil {
		return UpdateProfileInput{}, err
	}
	if country == "" && countryCode != "" {
		return UpdateProfileInput{}, fmt.Errorf("%w: country is required when specifying a country code", ErrInvalidInput)
	}
	if country != "" && countryCode == "" {
		return UpdateProfileInput{}, fmt.Errorf("%w: country code is required when specifying a country", ErrInvalidInput)
	}

	stateProvince := strings.TrimSpace(input.StateProvince)
	if utf8.RuneCountInString(stateProvince) > maxStateProvinceLength {
		return UpdateProfileInput{}, fmt.Errorf(
			"%w: state/province must be at most %d characters",
			ErrInvalidInput,
			maxStateProvinceLength,
		)
	}

	city := strings.TrimSpace(input.City)
	if utf8.RuneCountInString(city) > maxCityLength {
		return UpdateProfileInput{}, fmt.Errorf("%w: city must be at most %d characters", ErrInvalidInput, maxCityLength)
	}
	if country == "" && (stateProvince != "" || city != "") {
		return UpdateProfileInput{}, fmt.Errorf(
			"%w: country is required when specifying state/province or city",
			ErrInvalidInput,
		)
	}

	avatarDataURL := strings.TrimSpace(input.AvatarDataURL)
	if input.RemoveCustomAvatar && avatarDataURL != "" {
		return UpdateProfileInput{}, fmt.Errorf("%w: avatar upload and removal cannot be requested together", ErrInvalidInput)
	}

	if avatarDataURL != "" {
		if len(avatarDataURL) > maxAvatarDataURLLength {
			return UpdateProfileInput{}, fmt.Errorf("%w: avatar file is too large", ErrInvalidInput)
		}
		if !hasAllowedAvatarPrefix(avatarDataURL) {
			return UpdateProfileInput{}, fmt.Errorf("%w: avatar must be jpg, jpeg, png, or gif", ErrInvalidInput)
		}
	}

	return UpdateProfileInput{
		DisplayName:        displayName,
		Bio:                bio,
		Country:            country,
		CountryCode:        countryCode,
		StateProvince:      stateProvince,
		City:               city,
		AvatarDataURL:      avatarDataURL,
		RemoveCustomAvatar: input.RemoveCustomAvatar,
	}, nil
}

func normalizeCountryCode(value string) (string, error) {
	normalizedCountryCode := strings.ToUpper(strings.TrimSpace(value))
	if normalizedCountryCode == "" {
		return "", nil
	}

	if utf8.RuneCountInString(normalizedCountryCode) != maxCountryCodeLength {
		return "", fmt.Errorf("%w: country code must be a 2-letter ISO code", ErrInvalidInput)
	}

	for _, char := range normalizedCountryCode {
		if char < 'A' || char > 'Z' {
			return "", fmt.Errorf("%w: country code must contain only letters", ErrInvalidInput)
		}
	}

	return normalizedCountryCode, nil
}

func hasAllowedAvatarPrefix(value string) bool {
	for _, prefix := range allowedAvatarDataURLPrefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}

	return false
}

func applyProfileDefaults(profile Profile) Profile {
	if profile.DisplayName == "" {
		profile.DisplayName = profile.Username
	}
	if profile.AvatarSource == "" {
		profile.AvatarSource = AvatarSourceNone
	}
	if profile.EloHistory == nil {
		profile.EloHistory = []EloHistoryPoint{}
	}
	if profile.TopLanguages == nil {
		profile.TopLanguages = []TopLanguage{}
	}
	if profile.RecentMatches == nil {
		profile.RecentMatches = []RecentMatch{}
	}

	return profile
}
