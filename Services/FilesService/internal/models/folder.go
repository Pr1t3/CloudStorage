package models

import (
	"database/sql"
	"time"
)

type Folder struct {
	Id         int
	Hash       string
	UserId     int
	FolderName string
	FolderPath string
	ParentId   sql.NullInt32
	CreatedAt  time.Time
}
