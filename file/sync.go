package file

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type SyncFile struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	Id        string
	Name      string
	Path      string
	Data      []byte
	mutex     sync.Mutex
}

func NewSyncFile(name string, def any) (*SyncFile, error) {
	path, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			now := timezone.NowTime()
			id := utility.UUID()
			bt, err := json.Marshal(def)
			if err != nil {
				return nil, err
			}

			result := &SyncFile{
				CreatedAt: now,
				UpdatedAt: now,
				Id:        id,
				Name:      filepath.Base(path),
				Path:      path,
				Data:      bt,
				mutex:     sync.Mutex{},
			}
			result.Save()

			return result, nil
		} else {
			return nil, err
		}
	}

	var result SyncFile
	bytesData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBuffer(bytesData)
	decoder := gob.NewDecoder(buffer)
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}

	result.Name = filepath.Base(path)
	result.Path = path

	return &result, nil
}

/**
* Save
* @return error
**/
func (s *SyncFile) Save() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.UpdatedAt = timezone.NowTime()
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(s)
	if err != nil {
		return err
	}

	return os.WriteFile(s.Path, buffer.Bytes(), 0644)
}

/**
* Set
* @param data []byte
* @return error
**/
func (s *SyncFile) Set(data []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Data = data
	s.UpdatedAt = timezone.NowTime()

	return s.Save()
}

/**
* Get
* @return []byte, error
**/
func (s *SyncFile) Get() ([]byte, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, err
	}

	return data, nil
}

/**
* Unmarshal
* @param v any
* @return error
**/
func (s *SyncFile) Unmarshal(v any) error {
	return json.Unmarshal(s.Data, v)
}

/**
* Empty
* @return error
**/
func (s *SyncFile) Empty() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Data = []byte("")
	s.UpdatedAt = timezone.NowTime()
	return s.Save()
}

/**
* Delete
* @return error
**/
func (s *SyncFile) Delete() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := os.Remove(s.Path)
	if err != nil {
		return err
	}

	return nil
}
