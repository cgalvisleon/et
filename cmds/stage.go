package cmds

import "github.com/cgalvisleon/et/utility"

type Stage struct {
	Id          string
	Name        string
	Description string
	Steps       []*Step
}

/**
* NewStage
* @parms name
* @parms description
* @return *Stage
**/
func NewStage(name, description string) *Stage {
	return &Stage{
		Id:          utility.UUID(),
		Name:        name,
		Description: description,
		Steps:       make([]*Step, 0),
	}
}

/**
* AppendStep
* @parms name
* @parms description
* @return *Stage
**/
func (s *Stage) AppendStep(step *Step) *Stage {
	s.Steps = append(s.Steps, step)
	return s
}
