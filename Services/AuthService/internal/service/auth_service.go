package service

import (
	"AuthService/internal/models"
	"AuthService/internal/repository"
	"errors"
	"time"
)

type AuthService struct {
	repo repository.UserRepo
}

func NewAuthService(r repository.UserRepo) *AuthService {
	return &AuthService{repo: r}
}

func (s *AuthService) Login(email, password string) (time.Time, string, error) {
	user, err := s.repo.GetUserByEmail(email)
	if err != nil || user == nil || !CheckPassword(user.Password_hash, password) {
		return time.Now(), "", errors.New("Invalid Credentials")
	}
	return models.GenerateToken(user.Email, user.ID)
}

func (s *AuthService) Register(email, hashedPassword string) error {
	existingUser, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return err
	}
	if existingUser != nil {
		return errors.New("User already exists")
	}

	return s.repo.CreateUser(email, hashedPassword)
}
