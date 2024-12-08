package service

import (
	"FilesService/internal/models"
	"FilesService/internal/repository"
	"errors"
)

type FilesService struct {
	repo repository.FileRepo
}

func NewFilesService(r repository.FileRepo) *FilesService {
	return &FilesService{repo: r}
}

func (s *FilesService) GetFilesInFolder(folderId *int, userId int) ([]models.File, error) {
	files, err := s.repo.GetFilesInFolder(folderId, userId)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (s *FilesService) GetFileByHash(hash string) (*models.File, error) {
	file, err := s.repo.GetFileByHash(hash)
	return file, err
}

func (s *FilesService) AddFile(userId int, folderId *int, size int64, fileName, fileType string) error {
	files, err := s.GetFilesInFolder(folderId, userId)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.FileName == fileName {
			return errors.New("file already exists")
		}
	}
	err = s.repo.AddFile(userId, folderId, size, fileName, fileType)
	return err
}

func (s *FilesService) DeleteFile(hash string) error {
	err := s.repo.DeleteFile(hash)
	return err
}

func (s *FilesService) ChangeShareStatus(hash string, status bool) error {
	err := s.repo.ChangeShareStatus(hash, status)
	return err
}
