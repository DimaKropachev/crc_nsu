package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenService struct {
	secret []byte
	ttl    time.Duration
}
type Claims struct {
	UserID string `json:"uid"`
	VKID   int64  `json:"vk_id"`
	jwt.RegisteredClaims
}

func NewTokenService(secret string, ttl time.Duration) *TokenService {
	return &TokenService{secret: []byte(secret), ttl: ttl}
}
func (s *TokenService) Issue(userID uuid.UUID, vkID int64) (string, int64, error) {
	now := time.Now()
	claims := Claims{UserID: userID.String(), VKID: vkID, RegisteredClaims: jwt.RegisteredClaims{Issuer: "career-consultations", Subject: userID.String(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)), ID: uuid.NewString()}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	encoded, err := token.SignedString(s.secret)
	return encoded, int64(s.ttl.Seconds()), err
}
func (s *TokenService) Parse(encoded string) (uuid.UUID, int64, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(encoded, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	}, jwt.WithIssuer("career-consultations"), jwt.WithExpirationRequired(), jwt.WithLeeway(30*time.Second))
	if err != nil || token == nil || !token.Valid {
		return uuid.Nil, 0, errors.New("invalid token")
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil || claims.VKID <= 0 {
		return uuid.Nil, 0, errors.New("invalid token claims")
	}
	return id, claims.VKID, nil
}
