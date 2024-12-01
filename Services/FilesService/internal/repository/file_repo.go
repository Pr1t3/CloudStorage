package repository

import (
	"FilesService/internal/models"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strconv"
)

type FileRepo struct {
	Db *sql.DB
}

func NewRepository(db *sql.DB) *FileRepo {
	return &FileRepo{Db: db}
}

func (f *FileRepo) AddFile(userId int, fileName, filePath, fileType string) error {
	query := `INSERT INTO files (user_id, filename, filepath, filetype) VALUES(?, ?, ?, ?)`
	row, err := f.Db.Exec(query, userId, fileName, filePath, fileType)
	if err != nil {
		return err
	}
	id, err := row.LastInsertId()
	if err != nil {
		return err
	}
	hash := sha256.New()
	hash.Write([]byte(strconv.FormatInt(id, 10)))
	hashedID := hex.EncodeToString(hash.Sum(nil))
	_, err = f.Db.Exec(`UPDATE files SET hash = ? WHERE id = ?`, hashedID, id)
	return err
}

func (f *FileRepo) DeleteFile(hash string) error {
	query := `Delete from files where hash = ?`
	_, err := f.Db.Exec(query, hash)
	return err
}

func (f *FileRepo) GetFileByHash(hash string) (*models.File, error) {
	query := `SELECT id, hash, user_id, filename, filepath, filetype, sharestatus, uploaded_at FROM files where hash = ?`
	row := f.Db.QueryRow(query, hash)
	file := &models.File{}
	if err := row.Scan(&file.ID, &file.Hash, &file.User_id, &file.FileName, &file.FilePath, &file.FileType, &file.ShareStatus, &file.Uploaded_at); err != nil {
		return nil, err
	}
	return file, nil
}

func (f *FileRepo) GetUserFiles(userId int) ([]models.File, error) {
	query := `SELECT id, hash, user_id, filename, filepath, filetype, sharestatus, uploaded_at FROM files where user_id = ?`
	rows, err := f.Db.Query(query, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	var files []models.File
	for rows.Next() {
		file := &models.File{}
		if err := rows.Scan(&file.ID, &file.Hash, &file.User_id, &file.FileName, &file.FilePath, &file.FileType, &file.ShareStatus, &file.Uploaded_at); err != nil {
			return nil, err
		}
		files = append(files, *file)
	}
	return files, nil
}

func (f *FileRepo) ChangeShareStatus(hash string, status bool) error {
	query := `UPDATE files SET sharestatus = ? WHERE hash = ?`
	_, err := f.Db.Exec(query, status, hash)
	return err
}
