package models

import (
	"database/sql"
	"time"
)

type File struct {
	ID          int
	User_id     int
	Hash        string
	FileName    string
	FileType    string
	Size        int64
	FolderId    sql.NullInt32
	ShareStatus bool
	Uploaded_at time.Time
}
