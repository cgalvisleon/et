package brevo

import "github.com/cgalvisleon/et/et"

type SetFn func(id string, data et.Json) error
type DeleteFn func(id string) error

var (
	set    SetFn
	delete DeleteFn
)

func OnSet(f SetFn) {
	set = f
}

func OnDelete(f DeleteFn) {
	delete = f
}
