package service

import (
	"FilesService/internal/models"
	"FilesService/internal/repository"
)

type FilesService struct {
	repo repository.FileRepo
}

func NewFilesService(r repository.FileRepo) *FilesService {
	return &FilesService{repo: r}
}

func (s *FilesService) GetAllFiles(userId int) ([]models.File, error) {
	files, err := s.repo.GetUserFiles(userId)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (s *FilesService) GetFileByHash(hash string) (*models.File, error) {
	file, err := s.repo.GetFileByHash(hash)
	return file, err
}

func (s *FilesService) AddFile(userId int, fileName, filePath, fileType string) error {
	err := s.repo.AddFile(userId, fileName, filePath, fileType)
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
