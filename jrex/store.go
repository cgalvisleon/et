package jrex

type Store interface {
	SetModule(module string, source any) error
	GetModule(module string, source any) (bool, error)
	DeleteModule(module string) error
}

type FileStore struct {
	BaseDir string
}

func NewFileStore(baseDir string) (*FileStore, error) {
	return &FileStore{
		BaseDir: baseDir,
	}, nil
}

func (s *FileStore) SetModule(module string, source any) error {
	return nil
}

func (s *FileStore) GetModule(module string, source any) (bool, error) {
	return false, nil
}

func (s *FileStore) DeleteModule(module string) error {
	return nil
}
