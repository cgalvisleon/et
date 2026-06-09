package workflow

type Port string

const (
	PortInput  Port = "input"
	PortOutput Port = "output"
)

type StepConnection struct {
	StepId string `json:"steper_id"`
	Port   Port   `json:"port"`
	Index  int    `json:"index"`
}

type Connection struct {
	ID     string         `json:"id"`
	Tag    string         `json:"tag"`
	Source StepConnection `json:"source"`
	Target StepConnection `json:"target"`
	Kind   string         `json:"kind"`
}
