package crontab

type Storage struct {
	Jobs    []*Job
	Version string
}

func NewStorage() *Storage {
	return &Storage{
		Jobs:    make([]*Job, 0),
		Version: "v0.0.1",
	}
}
