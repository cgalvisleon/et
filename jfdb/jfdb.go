package jfdb

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/google/uuid"
)

var rdms Jfdb

type Jfdb struct {
	Created_date time.Time
	Update_date  time.Time
	Id           string
	Filename     string
	Data         et.Json
	Databases    []*Database
	Users        []*User
}

func init() {

}

func (s *Jfdb) content() []byte {
	return []byte("")
}

func (s *Jfdb) save() error {
	s.Update_date = time.Now()

	return nil
}

func (s *Jfdb) indexDatabase(name string) Number {
	for _, database := range s.Databases {
		if database.Name == name {
			return database.Index
		}
	}

	return -1
}

func (s *Jfdb) NewDatabase(name, description string) (*Database, error) {
	idx := s.indexDatabase(name)
	if idx == -1 {
		return nil, Errorm(T("MSG_DATABASE_EXISTS"))
	}

	id := uuid.NewString()
	fileName := Format(`%s.db`, id)
	idx = Number(s.Index() + 1)
	result := &Database{
		Created_date: time.Now(),
		Update_date:  time.Now(),
		Id:           id,
		Name:         name,
		Description:  description,
		Filename:     fileName,
		Data:         et.Json{},
		Schemas:      []*Schema{},
		Index:        idx,
	}
	err := result.save()
	if err != nil {
		return nil, err
	}

	s.Databases = append(s.Databases, result)
	err = s.save()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Jfdb) NewUser(name, description, filename string) (*User, error) {

	err := s.save()
	if err != nil {
		return nil, err
	}

	return &User{}, nil
}

func (s *Jfdb) NewSchema(name, description string) (*Schema, error) {
	return &Schema{}, nil
}

func (s *Jfdb) NewCollection(name, description string) (*Collection, error) {
	return &Collection{}, nil
}
