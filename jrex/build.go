package jrex

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/file"
)

type Module struct {
	ID      string `json:"id"`
	Scripts string `json:"scripts"`
}

/**
* Build
* @param part Part
* @return error
**/
func (s *Jrex) Build(part Part) error {
	if s.mode != Develop {
		return errors.New("build is only available in develop mode")
	}

	ok, err := s.files.ReadJSON("package.json", &s.Pkg)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("package.json not found")
	}

	_, err = s.BumpVersion(part)
	if err != nil {
		return err
	}

	err = s.uppToStore()
	if err != nil {
		return err
	}

	for module, path := range s.Pkg.Modules {
		inf := file.ExistPath(path)
		if inf.IsDir {
			continue
		}
		data, err := s.files.ReadTextFile(path)
		if err != nil {
			return err
		}

		id := fmt.Sprintf("pkg:%s:%s:%s", s.Name, module, s.Version)
		s.store.SetModule(id, &Module{
			ID:      id,
			Scripts: data,
		})
	}

	return nil
}
