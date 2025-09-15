package file

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type SyncFile struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	Id        string
	Name      string
	Dir       string
	Path      string
	Data      []byte
	mutex     sync.Mutex
}

/**
* NewSyncFile
* @param dataDirectory, name string, initialData any
* @return *SyncFile, error
**/
func NewSyncFile(dataDirectory, name string, initialData any) (*SyncFile, error) {
	if !utility.ValidStr(dataDirectory, 1, []string{"/", ""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "dataDirectory")
	}

	dir, err := MakeFolder(dataDirectory)
	if err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf("%s/%s.dt", dir, strs.Lowcase(name))
	path, err := filepath.Abs(fileName)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			now := timezone.NowTime()
			id := utility.UUID()
			bt, err := json.Marshal(initialData)
			if err != nil {
				return nil, err
			}

			result := &SyncFile{
				CreatedAt: now,
				UpdatedAt: now,
				Id:        id,
				Dir:       dir,
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
* @param data []byte, saved bool
* @return error
**/
func (s *SyncFile) Set(data []byte, saved bool) error {
	s.Data = data
	if saved {
		return s.Save()
	}

	return nil
}

/**
* Load
* @param v any
* @return error
**/
func (s *SyncFile) Load(v any) error {
	return json.Unmarshal(s.Data, v)
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
