package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"movieexample.com/auth/pkg/model"
)

type authRepository interface {
	Put(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)

	Close() error
}

// SecretProvider defines a provider of secrets for our handler.
type SecretProvider func() []byte

// Controller defines a auth service controller.
type Controller struct {
	repo           authRepository
	secretProvider SecretProvider
}

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid token")
)

// New creates a auth service controller.
func New(repo authRepository, secretProvider SecretProvider) *Controller {
	return &Controller{repo, secretProvider}
}

// Register creates a new user account. It hashes the password using bcrypt
// before persisting the user data to the repository.
func (c *Controller) Register(ctx context.Context, email, username, password, firstName, lastName string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		ID:           uuid.Must(uuid.NewRandom()).String(),
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		FirstName:    firstName,
		LastName:     lastName,
	}

	err = c.repo.Put(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

// Login verifies user credentials and returns a JWT token if successful.
func (c *Controller) Login(ctx context.Context, email, password string) (*jwt.Token, error) {
	user, err := c.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"sub":      user.ID,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	return token, nil
}

// GetTokenString signs the JWT token using the key provided by the secretProvider.
func (c *Controller) GetTokenString(token *jwt.Token) (string, error) {
	return token.SignedString(c.secretProvider())
}

// IsValid parses and validates a JWT string. If valid, it returns the username
// associated with the token; otherwise, it returns an error.
func (c *Controller) IsValid(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return c.secretProvider(), nil
	})
	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrInvalidToken
	}

	username, ok := claims["username"].(string)
	if !ok {
		return "", ErrInvalidToken
	}

	return username, nil
}
