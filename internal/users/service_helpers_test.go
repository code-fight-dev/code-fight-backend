package users

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeAvatarUpload(t *testing.T) {
	contentType, extension, err := normalizeAvatarUpload(UpdateAvatarInput{
		Content:     []byte("image-bytes"),
		ContentType: " IMAGE/PNG ",
	})
	if err != nil {
		t.Fatalf("normalizeAvatarUpload returned error: %v", err)
	}
	if contentType != "image/png" || extension != "png" {
		t.Fatalf("content type/extension = %q/%q", contentType, extension)
	}

	tests := []UpdateAvatarInput{
		{ContentType: "image/png"},
		{Content: make([]byte, maxAvatarFileSize+1), ContentType: "image/png"},
		{Content: []byte("x"), ContentType: "image/webp"},
	}
	for _, input := range tests {
		if _, _, err := normalizeAvatarUpload(input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("input %#v error = %v, want ErrInvalidInput", input, err)
		}
	}
}

func TestNormalizeUpdateProfileInput(t *testing.T) {
	input, err := normalizeUpdateProfileInput(UpdateProfileInput{
		DisplayName:   " Ada ",
		Bio:           " Solver ",
		Country:       " Germany ",
		CountryCode:   " de ",
		StateProvince: " Berlin ",
		City:          " Berlin ",
		AvatarDataURL: " data:image/png;base64,abc ",
	})
	if err != nil {
		t.Fatalf("normalizeUpdateProfileInput returned error: %v", err)
	}
	if input.DisplayName != "Ada" || input.CountryCode != "DE" || input.City != "Berlin" {
		t.Fatalf("normalized input = %#v", input)
	}
	if !hasAllowedAvatarPrefix(input.AvatarDataURL) {
		t.Fatalf("avatar prefix rejected: %q", input.AvatarDataURL)
	}
}

func TestNormalizeUpdateProfileInputRejectsInvalidValues(t *testing.T) {
	tests := []UpdateProfileInput{
		{DisplayName: ""},
		{DisplayName: strings.Repeat("x", maxDisplayNameLength+1)},
		{DisplayName: "Ada", Bio: strings.Repeat("x", maxBioLength+1)},
		{DisplayName: "Ada", Country: strings.Repeat("x", maxCountryLength+1), CountryCode: "DE"},
		{DisplayName: "Ada", CountryCode: "D"},
		{DisplayName: "Ada", CountryCode: "D1"},
		{DisplayName: "Ada", CountryCode: "DE"},
		{DisplayName: "Ada", Country: "Germany"},
		{DisplayName: "Ada", Country: "Germany", CountryCode: "DE", StateProvince: strings.Repeat("x", maxStateProvinceLength+1)},
		{DisplayName: "Ada", Country: "Germany", CountryCode: "DE", City: strings.Repeat("x", maxCityLength+1)},
		{DisplayName: "Ada", StateProvince: "Berlin"},
		{DisplayName: "Ada", AvatarDataURL: "data:image/webp;base64,abc"},
		{DisplayName: "Ada", AvatarDataURL: strings.Repeat("x", maxAvatarDataURLLength+1)},
		{DisplayName: "Ada", AvatarDataURL: "data:image/png;base64,abc", RemoveCustomAvatar: true},
	}

	for _, input := range tests {
		if _, err := normalizeUpdateProfileInput(input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("input %#v error = %v, want ErrInvalidInput", input, err)
		}
	}
}

func TestNormalizeCountryCode(t *testing.T) {
	got, err := normalizeCountryCode(" us ")
	if err != nil {
		t.Fatalf("normalizeCountryCode returned error: %v", err)
	}
	if got != "US" {
		t.Fatalf("country code = %q", got)
	}
	if got, err := normalizeCountryCode(""); err != nil || got != "" {
		t.Fatalf("empty country code = %q, %v", got, err)
	}
}

func TestApplyProfileDefaults(t *testing.T) {
	profile := applyProfileDefaults(Profile{Username: "player"})
	if profile.DisplayName != "player" {
		t.Fatalf("display name = %q", profile.DisplayName)
	}
	if profile.AvatarSource != AvatarSourceNone {
		t.Fatalf("avatar source = %q", profile.AvatarSource)
	}
	if profile.EloHistory == nil || profile.TopLanguages == nil || profile.RecentMatches == nil {
		t.Fatalf("default slices are nil: %#v", profile)
	}
}
