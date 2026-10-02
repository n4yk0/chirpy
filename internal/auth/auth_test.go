package auth_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/n4yk0/chirpy/internal/auth"
)

const testSecret = "test-secret-do-not-use-in-production"

func TestHashPasswordProducesVerifiableHash(t *testing.T) {
	const password = "correct horse battery staple"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v, want nil", err)
	}
	if hash == password {
		t.Fatal("HashPassword() returned the password in clear")
	}

	match, err := auth.CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash() error = %v, want nil", err)
	}
	if !match {
		t.Error("CheckPasswordHash() = false, want true for the original password")
	}
}

func TestHashPasswordUsesRandomSalt(t *testing.T) {
	const password = "same password twice"

	first, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v, want nil", err)
	}
	second, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v, want nil", err)
	}

	if first == second {
		t.Error("hashing the same password twice produced identical hashes, salt is not random")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	hash, err := auth.HashPassword("right-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v, want nil", err)
	}

	emptyHash, err := auth.HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword() error = %v, want nil", err)
	}

	tests := []struct {
		name      string
		password  string
		hash      string
		wantMatch bool
		wantErr   bool
	}{
		{
			name:      "matching password",
			password:  "right-password",
			hash:      hash,
			wantMatch: true,
		},
		{
			name:     "wrong password",
			password: "wrong-password",
			hash:     hash,
		},
		{
			name:     "password differing only by case",
			password: "Right-Password",
			hash:     hash,
		},
		{
			name:     "empty password against a real hash",
			password: "",
			hash:     hash,
		},
		{
			name:      "empty password against its own hash",
			password:  "",
			hash:      emptyHash,
			wantMatch: true,
		},
		{
			name:     "malformed stored hash",
			password: "right-password",
			hash:     "not-a-phc-string",
			wantErr:  true,
		},
		{
			name:     "empty stored hash",
			password: "right-password",
			hash:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := auth.CheckPasswordHash(tt.password, tt.hash)
			if tt.wantErr {
				if err == nil {
					t.Fatal("CheckPasswordHash() error = nil, want an error")
				}
				if match {
					t.Error("CheckPasswordHash() = true on error, want false")
				}
				return
			}
			if err != nil {
				t.Fatalf("CheckPasswordHash() error = %v, want nil", err)
			}
			if match != tt.wantMatch {
				t.Errorf("CheckPasswordHash() = %v, want %v", match, tt.wantMatch)
			}
		})
	}
}

func TestMakeJWTValidateJWTRoundTrip(t *testing.T) {
	userID := uuid.New()

	token, err := auth.MakeJWT(userID, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v, want nil", err)
	}

	got, err := auth.ValidateJWT(token, testSecret)
	if err != nil {
		t.Fatalf("ValidateJWT() error = %v, want nil", err)
	}
	if got != userID {
		t.Errorf("ValidateJWT() = %v, want %v", got, userID)
	}
}

func TestMakeJWTDoesNotLeakTheSecret(t *testing.T) {
	token, err := auth.MakeJWT(uuid.New(), testSecret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v, want nil", err)
	}

	if strings.Contains(token, testSecret) {
		t.Error("the signing secret appears in the token")
	}
	if n := len(strings.Split(token, ".")); n != 3 {
		t.Errorf("token has %d dot-separated segments, want 3", n)
	}
}

func TestValidateJWTRejectsExpiredToken(t *testing.T) {
	token, err := auth.MakeJWT(uuid.New(), testSecret, -time.Second)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v, want nil", err)
	}

	got, err := auth.ValidateJWT(token, testSecret)
	if err == nil {
		t.Fatal("ValidateJWT() error = nil, want an error for an expired token")
	}
	if !errors.Is(err, jwt.ErrTokenExpired) {
		t.Errorf("ValidateJWT() error = %v, want it to wrap jwt.ErrTokenExpired", err)
	}
	if got != uuid.Nil {
		t.Errorf("ValidateJWT() = %v on error, want uuid.Nil", got)
	}
}

func TestValidateJWTRejectsWrongSecret(t *testing.T) {
	token, err := auth.MakeJWT(uuid.New(), testSecret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v, want nil", err)
	}

	got, err := auth.ValidateJWT(token, "another-secret")
	if err == nil {
		t.Fatal("ValidateJWT() error = nil, want an error for a token signed with another secret")
	}
	if got != uuid.Nil {
		t.Errorf("ValidateJWT() = %v on error, want uuid.Nil", got)
	}
}

func TestValidateJWTRejectsUnexpectedSigningMethod(t *testing.T) {
	now := time.Now().UTC()
	forged := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		Subject:   uuid.New().String(),
	})
	token, err := forged.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("SignedString() error = %v, want nil", err)
	}

	if _, err := auth.ValidateJWT(token, testSecret); err == nil {
		t.Fatal("ValidateJWT() accepted an HS512 token, want only HS256")
	}
}

func TestValidateJWTRejectsMalformedToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{name: "empty string", token: ""},
		{name: "not a jwt", token: "gibberish"},
		{name: "two segments only", token: "header.payload"},
		{name: "empty signature", token: "header.payload."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := auth.ValidateJWT(tt.token, testSecret)
			if err == nil {
				t.Fatal("ValidateJWT() error = nil, want an error")
			}
			if got != uuid.Nil {
				t.Errorf("ValidateJWT() = %v on error, want uuid.Nil", got)
			}
		})
	}
}

func TestGetAPIKey(t *testing.T) {
	cases := []struct {
		name string
		header string
		want string
		wantErr bool
	}{
		{"valid", "ApiKey abc123", "abc123", false},
		{"missing header", "", "", true},
		{"bearer scheme", "Bearer abc123", "", true},
		{"no space", "ApiKeyabc123", "", true},
		{"empty key", "ApiKey ", "", true},
		{"lowercase scheme", "apikey abc123", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			headers := http.Header{}
			if tc.header != "" {
				headers.Set("Authorization", tc.header)
			}

			got, err := auth.GetAPIKey(headers)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
