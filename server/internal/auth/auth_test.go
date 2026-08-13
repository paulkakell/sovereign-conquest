package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

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
			IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
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
