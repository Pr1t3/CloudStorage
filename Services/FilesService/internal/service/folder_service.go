package service

import (
	"FilesService/internal/models"
	"FilesService/internal/repository"
)

type FolderService struct {
	repo repository.FolderRepo
}

func NewFolderService(r repository.FolderRepo) *FolderService {
	return &FolderService{repo: r}
}

func (s *FolderService) GetFolderByHash(hash string) (*models.Folder, error) {
	folder, err := s.repo.GetFolderByHash(hash)
	return folder, err
}

func (s *FolderService) CreateFolder(name, path string, parentId *int, userId int) error {
	err := s.repo.CreateFolder(name, path, parentId, userId)
	return err
}

func (s *FolderService) DeleteFolder(hash string) error {
	folder, err := s.repo.GetFolderByHash(hash)
	if err != nil {
		return err
	}
	err = s.repo.DeleteFolder(folder.Id)
	return err
}

func (s *FolderService) GetSiblingFolders(parent_id *int, userId int) ([]models.Folder, error) {
	return s.repo.GetSiblingFolders(parent_id, userId)
}

func (s *FolderService) GetFolderById(folderId int) (*models.Folder, error) {
	folder, err := s.repo.GetFolderById(folderId)
	return folder, err
}
