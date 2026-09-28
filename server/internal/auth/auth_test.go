package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordLengthBoundariesMatchBcrypt(t *testing.T) {
	for _, input := range []string{strings.Repeat("a", 8), strings.Repeat("a", 72), strings.Repeat("é", 36)} {
		if err := ValidatePasswordLength(input); err != nil {
			t.Fatal(err)
		}
		hash, err := HashPassword(input)
		if err != nil || !CheckPassword(hash, input) {
			t.Fatal("valid boundary password did not round-trip")
		}
	}
	for _, input := range []string{strings.Repeat("a", 7), strings.Repeat("a", 73), strings.Repeat("é", 37)} {
		if err := ValidatePasswordLength(input); err == nil {
			t.Fatal("invalid password length accepted")
		}
	}
	if _, err := HashPassword(strings.Repeat("a", 73)); !errors.Is(err, bcrypt.ErrPasswordTooLong) {
		t.Fatalf("unexpected bcrypt upper limit: %v", err)
	}
}

func TestHashRoundTripAndCost(t *testing.T) {
	input := strings.Repeat("a", 24)
	hash, err := HashPassword(input)
	if err != nil {
		t.Fatalf("hash input: %v", err)
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("read cost: %v", err)
	}
	if cost != passwordHashCost {
		t.Fatalf("cost=%d want=%d", cost, passwordHashCost)
	}
	if !CheckPassword(hash, input) || CheckPassword(hash, input+"b") {
		t.Fatal("hash verification mismatch")
	}
}

func TestTokenAlgorithmAndSessionVersion(t *testing.T) {
	key := strings.Repeat("k", 32)
	signed, err := MintTokenForSession(key, "user", "player", 7, time.Hour)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	claims, err := ParseToken(key, signed)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.SessionVersion != 7 {
		t.Fatalf("session version=%d want=7", claims.SessionVersion)
	}

	bad := jwt.NewWithClaims(jwt.SigningMethodHS384, Claims{
		UserID: "user", PlayerID: "player",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
		},
	})
	badSigned, err := bad.SignedString([]byte(key))
	if err != nil {
		t.Fatalf("sign alternate algorithm: %v", err)
	}
	if _, err := ParseToken(key, badSigned); err == nil {
		t.Fatal("alternate algorithm was accepted")
	}
}
