package cmds

import (
	"os/exec"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/utility"
)

type Cmd struct {
	Name string
	Args []string
}

type Step struct {
	Id          string
	Name        string
	Description string
	Commands    []*Cmd
}

/**
* NewStep
* @parms name
* @parms description
* @return *Cmds
**/
func NewStep(name, description string) *Step {
	return &Step{
		Id:          utility.UUID(),
		Name:        name,
		Description: description,
		Commands:    make([]*Cmd, 0),
	}
}

/**
* AppendCmd
* @parms args string
* @return *Step
**/
func (s *Step) AppendCmd(args string) *Step {
	list := strings.Split(args, " ")
	n := len(list)
	switch n {
	case 0:
		return s
	case 1:
		result := &Cmd{Name: list[0]}
		s.Commands = append(s.Commands, result)
		return s
	default:
		result := &Cmd{Name: list[0], Args: list[1:]}
		s.Commands = append(s.Commands, result)
		return s
	}
}

/**
* RunOS
* @parms idx int
* @parms args et.Json
* @return []byte
* @return error
**/
func (s *Step) RunOS(idx int, args et.Json) ([]byte, error) {
	if idx < 0 || idx >= len(s.Commands) {
		return nil, mistake.New(MSG_INDEX_NOT_FOUND)
	}

	cmd := s.Commands[idx]
	for k, v := range args {
		idx := slices.IndexFunc(cmd.Args, func(a string) bool { return strings.HasPrefix(a, k) })
		if idx != -1 {
			s, ok := v.(string)
			if !ok {
				return nil, mistake.New(MSG_INVALID_TYPE)
			}
			cmd.Args[idx] = s
		}
	}

	result := exec.Command(cmd.Name, cmd.Args...)

	output, err := result.Output()
	if err != nil {
		return nil, err
	}

	return output, nil
}

/**
* RunSSH
* @parms idx int
* @parms args et.Json
* @return []byte
* @return error
**/
func (s *Step) RunSSH(idx int, args et.Json) ([]byte, error) {
	if idx < 0 || idx >= len(s.Commands) {
		return nil, mistake.New(MSG_INDEX_NOT_FOUND)
	}

	cmd := s.Commands[idx]
	for k, v := range args {
		idx := slices.IndexFunc(cmd.Args, func(a string) bool { return strings.HasPrefix(a, k) })
		if idx != -1 {
			s, ok := v.(string)
			if !ok {
				return nil, mistake.New(MSG_INVALID_TYPE)
			}
			cmd.Args[idx] = s
		}
	}

	result := exec.Command(cmd.Name, cmd.Args...)

	output, err := result.Output()
	if err != nil {
		return nil, err
	}

	return output, nil
}
