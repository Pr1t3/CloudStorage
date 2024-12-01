package repository

import (
	"AuthService/internal/models"
	"database/sql"
	"errors"
)

type UserRepo struct {
	Db *sql.DB
}

func NewRepository(db *sql.DB) *UserRepo {
	return &UserRepo{Db: db}
}

func (u *UserRepo) CreateUser(email, password_hash string) error {
	query := `INSERT INTO users (email, password_hash) VALUES(?, ?)`
	_, err := u.Db.Exec(query, email, password_hash)
	return err
}

func (u *UserRepo) GetUserByEmail(email string) (*models.User, error) {
	query := `SELECT id, email, password_hash, created_at, updated_at FROM users where email = ?`
	row := u.Db.QueryRow(query, email)
	user := &models.User{}
	err := row.Scan(&user.ID, &user.Email, &user.Password_hash, &user.Created_at, &user.Updated_at)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}
