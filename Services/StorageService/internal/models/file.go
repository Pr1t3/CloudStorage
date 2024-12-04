package models

import "time"

type File struct {
	ID          int       `json:"ID"`
	Hash        string    `json:"Hash"`
	UserID      int       `json:"User_id"`
	FileName    string    `json:"FileName"`
	FilePath    string    `json:"FilePath"`
	FileType    string    `json:"FileType"`
	ShareStatus bool      `json:"ShareStatus"`
	UploadedAt  time.Time `json:"Uploaded_at"`
}
