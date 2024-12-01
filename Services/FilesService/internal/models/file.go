package models

import (
	"time"
)

type File struct {
	ID          int
	User_id     int
	Hash        string
	FileName    string
	FilePath    string
	FileType    string
	ShareStatus bool
	Uploaded_at time.Time
}
