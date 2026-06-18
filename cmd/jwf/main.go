package main

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jwf"
	"github.com/cgalvisleon/et/logs"
)

func main() {
	wf, err := jwf.New("37860631", nil)
	if err != nil {
		logs.Panic(err)
	}

	f := wf.NewFloW("add", "add item", "1.0.0", "cgalvisl").
		Step("add", "add item", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) {
			result := et.Json{
				"step1": "step1",
			}
			instance.SetParams(result)
			return result, nil
		}).
		Step("add", "add item", func(instance *jwf.Instance, ctx et.Json) (et.Json, error) {
			result := et.Json{
				"step2": "step2",
			}
			instance.SetParams(result)
			return result, nil
		})

	result, err := wf.Run(f.ID, "add", "", "37860631", et.Json{}, et.Json{}, "cgalvisl")
	if err != nil {
		logs.Panic(err)
	}

	logs.Info(result.ToString())
}
