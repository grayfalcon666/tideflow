package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"tideflow/internal/models"
	"tideflow/internal/repository"
)

var (
	ErrUserExists      = errors.New("username already exists")
	ErrInvalidPassword = errors.New("invalid password")
	ErrInvalidToken    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token expired")
)

type AuthService struct {
	repo      *repository.Repository
	secret    []byte
	accessExp time.Duration
	refreshExp time.Duration
}

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func NewAuthService(repo *repository.Repository, secret string, accessExp, refreshExp time.Duration) *AuthService {
	return &AuthService{
		repo:      repo,
		secret:    []byte(secret),
		accessExp: accessExp,
		refreshExp: refreshExp,
	}
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (s *AuthService) Register(ctx context.Context, username, password string) (*TokenPair, uint, error) {
	_, err := s.repo.GetAccountByUsername(ctx, username)
	if err == nil {
		return nil, 0, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, 0, err
	}

	acc := &models.Account{
		Username: username,
		Password: string(hash),
	}
	if err := s.repo.CreateAccount(ctx, acc); err != nil {
		return nil, 0, err
	}

	return s.generateTokens(ctx, acc.ID)
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*TokenPair, uint, error) {
	acc, err := s.repo.GetAccountByUsername(ctx, username)
	if err != nil {
		return nil, 0, ErrInvalidPassword
	}

	if err := bcrypt.CompareHashAndPassword([]byte(acc.Password), []byte(password)); err != nil {
		return nil, 0, ErrInvalidPassword
	}

	return s.generateTokens(ctx, acc.ID)
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, uint, error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return nil, 0, err
	}

	acc, err := s.repo.GetAccountByID(ctx, claims.UserID)
	if err != nil {
		return nil, 0, ErrInvalidToken
	}

	return s.generateTokens(ctx, acc.ID)
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *AuthService) generateTokens(ctx context.Context, userID uint) (*TokenPair, uint, error) {
	now := time.Now()
	accessExp := now.Add(s.accessExp)
	refreshExp := now.Add(s.refreshExp)

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})

	accessStr, err := accessToken.SignedString(s.secret)
	if err != nil {
		return nil, 0, err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})

	refreshStr, err := refreshToken.SignedString(s.secret)
	if err != nil {
		return nil, 0, err
	}

	s.repo.UpdateAccount(ctx, userID, map[string]interface{}{
		"token":        accessStr,
		"refresh_token": refreshStr,
	})

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresIn:   int64(s.accessExp.Seconds()),
	}, userID, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error {
	acc, err := s.repo.GetAccountByUsername(ctx, username)
	if err != nil {
		return ErrInvalidPassword
	}

	if err := bcrypt.CompareHashAndPassword([]byte(acc.Password), []byte(oldPassword)); err != nil {
		return ErrInvalidPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.repo.UpdateAccount(ctx, acc.ID, map[string]interface{}{"password": string(hash)})
}

func (s *AuthService) Logout(ctx context.Context, userID uint) error {
	return s.repo.UpdateAccount(ctx, userID, map[string]interface{}{
		"token":        "",
		"refresh_token": "",
	})
}