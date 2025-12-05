package crontab

type SetFn func(*Job) error
type GetFn func(string) (*Job, error)
type DeleteFn func(string) error

var (
	set    SetFn
	get    GetFn
	delete DeleteFn
)

func OnSet(f SetFn) {
	set = f
}

func OnGet(f GetFn) {
	get = f
}

func OnDelete(f DeleteFn) {
	delete = f
}
