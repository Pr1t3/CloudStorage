package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID            int
	Email         string
	Password_hash string
	Photo_path    sql.NullString
	Photo_type    sql.NullString
	Created_at    time.Time
	Updated_at    time.Time
}
