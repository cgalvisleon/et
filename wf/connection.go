package workflow

type Port string

const (
	PortInput  Port = "input"
	PortOutput Port = "output"
)

type StepConnection struct {
	SteperId string `json:"steper_id"`
	Port     Port   `json:"port"`
	Index    int    `json:"index"`
}

type Connection struct {
	ID     string         `json:"id"`
	Source StepConnection `json:"source"`
	Target StepConnection `json:"target"`
}
