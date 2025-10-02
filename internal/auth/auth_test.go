package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---- Password tests ----

func TestHashAndCheckPassword(t *testing.T) {
	password := "supersecret"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	ok, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash failed: %v", err)
	}
	if !ok {
		t.Error("expected password check to succeed, but it failed")
	}

	// Wrong password should fail
	ok, _ = CheckPasswordHash("wrongpassword", hash)
	if ok {
		t.Error("expected password check to fail, but it succeeded")
	}
}

// ---- JWT tests ----

func TestMakeAndValidateJWT(t *testing.T) {
	secret := "testsecret"
	userID := uuid.New()

	// Create a token
	token, err := MakeJWT(userID, secret, time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	// Validate token
	gotUserID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}
	if gotUserID != userID {
		t.Errorf("expected userID %v, got %v", userID, gotUserID)
	}
}

func TestValidateJWTWithWrongSecret(t *testing.T) {
	secret := "correctsecret"
	userID := uuid.New()

	token, err := MakeJWT(userID, secret, time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	_, err = ValidateJWT(token, "wrongsecret")
	if err == nil {
		t.Error("expected validation to fail with wrong secret, but it succeeded")
	}
}

func TestValidateExpiredJWT(t *testing.T) {
	secret := "testsecret"
	userID := uuid.New()

	// Token expires immediately
	token, err := MakeJWT(userID, secret, -time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Error("expected validation to fail with expired token, but it succeeded")
	}
}

func TestGetBearerToken(t *testing.T) {
	cases := []struct {
		name     string
		headers  http.Header
		expected string
		wantErr  bool
	}{
		{
			name:     "no authorization header",
			headers:  http.Header{},
			expected: "",
			wantErr:  true,
		},
		{
			name: "empty authorization header",
			headers: http.Header{
				"Authorization": []string{""},
			},
			expected: "",
			wantErr:  true,
		},
		{
			name: "valid bearer token",
			headers: http.Header{
				"Authorization": []string{"Bearer mytoken123"},
			},
			expected: "mytoken123",
			wantErr:  false,
		},
		{
			name: "authorization without Bearer prefix",
			headers: http.Header{
				"Authorization": []string{"mytoken123"},
			},
			expected: "mytoken123",
			wantErr:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := GetBearerToken(tc.headers)
			if (err != nil) != tc.wantErr {
				t.Errorf("expected error: %v, got: %v", tc.wantErr, err)
			}
			if token != tc.expected {
				t.Errorf("expected token: %q, got: %q", tc.expected, token)
			}
		})
	}
}
