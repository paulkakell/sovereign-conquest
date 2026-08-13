package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const passwordHashCost = 12

type Claims struct {
	UserID         string `json:"uid"`
	PlayerID       string `json:"pid"`
	SessionVersion int64  `json:"sv"`
	jwt.RegisteredClaims
}

func HashPassword(pw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), passwordHashCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func MintToken(secret, userID, playerID string, ttl time.Duration) (string, error) {
	return MintTokenForSession(secret, userID, playerID, 0, ttl)
}

func MintTokenForSession(secret, userID, playerID string, sessionVersion int64, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret is empty")
	}
	if ttl <= 0 {
		return "", errors.New("token ttl must be positive")
	}

	now := time.Now().UTC().Truncate(time.Second)
	claims := Claims{
		UserID:         userID,
		PlayerID:       playerID,
		SessionVersion: sessionVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(secret, tokenString string) (*Claims, error) {
	if secret == "" {
		return nil, errors.New("jwt secret is empty")
	}
	parsed, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid || claims.UserID == "" || claims.PlayerID == "" || claims.IssuedAt == nil {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
