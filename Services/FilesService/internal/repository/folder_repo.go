package repository

import (
	"FilesService/internal/models"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strconv"
)

type FolderRepo struct {
	Db *sql.DB
}

func NewFolderRepository(db *sql.DB) *FolderRepo {
	return &FolderRepo{Db: db}
}

func (f *FolderRepo) CreateFolder(name, path string, parentId *int, userId int) error {
	var query string
	var args []interface{} = []interface{}{userId, name, path}

	if parentId == nil {
		query = `INSERT INTO folders (user_id, folder_name, folder_path, parent_id) VALUES(?, ?, ?, null)`
	} else {
		query = `INSERT INTO folders (user_id, folder_name, folder_path, parent_id) VALUES(?, ?, ?, ?)`
		args = append(args, *parentId)
	}
	row, err := f.Db.Exec(query, args...)
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
	_, err = f.Db.Exec(`UPDATE folders SET hash = ? WHERE id = ?`, hashedID, id)
	return err
}

func (f *FolderRepo) DeleteFolder(id int) error {
	query := `Delete from folders where id = ?`
	_, err := f.Db.Exec(query, id)
	return err
}

func (f *FolderRepo) GetFolderByHash(hash string) (*models.Folder, error) {
	query := `SELECT id, hash, user_id, folder_name, folder_path, parent_id, created_at FROM folders where hash = ?`
	row := f.Db.QueryRow(query, hash)
	folder := &models.Folder{}
	if err := row.Scan(&folder.Id, &folder.Hash, &folder.UserId, &folder.FolderName, &folder.FolderPath, &folder.ParentId, &folder.CreatedAt); err != nil {
		return nil, err
	}
	return folder, nil
}

func (f *FolderRepo) GetSiblingFolders(parent_id *int, userId int) ([]models.Folder, error) {
	var query string
	var args []interface{} = []interface{}{userId}

	if parent_id == nil {
		query = `SELECT id, hash, user_id, folder_name, folder_path, parent_id, created_at FROM folders where user_id = ? and id in (select id from folders where parent_id is null)`
	} else {
		query = `SELECT id, hash, user_id, folder_name, folder_path, parent_id, created_at FROM folders where user_id = ? and id in (select id from folders where parent_id = ?)`
		args = append(args, *parent_id)
	}

	rows, err := f.Db.Query(query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	var folders []models.Folder
	for rows.Next() {
		folder := &models.Folder{}
		if err := rows.Scan(&folder.Id, &folder.Hash, &folder.UserId, &folder.FolderName, &folder.FolderPath, &folder.ParentId, &folder.CreatedAt); err != nil {
			return nil, err
		}
		folders = append(folders, *folder)
	}
	return folders, nil
}

func (f *FolderRepo) GetFolderById(folderId int) (*models.Folder, error) {
	query := `SELECT id, hash, user_id, folder_name, folder_path, parent_id, created_at FROM folders where id = ?`
	row := f.Db.QueryRow(query, folderId)
	folder := &models.Folder{}
	if err := row.Scan(&folder.Id, &folder.Hash, &folder.UserId, &folder.FolderName, &folder.FolderPath, &folder.ParentId, &folder.CreatedAt); err != nil {
		return nil, err
	}
	return folder, nil
}
