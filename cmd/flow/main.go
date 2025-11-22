package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/workflow"
)

func main() {
	flowDefinition()
	// utility.AppWait()

	logs.Logf("test", "Fin de flow")

}

func flowDefinition() {
	workflow.NewByFile("report:set", "1.0.0", "Registro de reportes", "", "./step0.js", true, "test").
		StepByFile("Step 1", "Step 1", "./step1.js", true)

	result, err := workflow.Run("1234", "report:set", 0, et.Json{}, et.Json{
		"test": "test",
	}, "test")
	if err != nil {
		logs.Error(err)
	} else {
		logs.Logf("Result:", result.ToString())
	}
}

func flowFn() {
	workflow.NewFn("ventas", "1.0.0", "Flujo de ventas", "flujo de ventas", func(flow *workflow.Instance, ctx et.Json) (et.Json, error) {
		logs.Logf("Respuesta desde step 0, contexto:", ctx.ToString())
		atrib := fmt.Sprintf("step_%d", flow.Current)
		ctx.Set(atrib, "step0")

		return ctx, nil
	}, true, "test").
		Debug().
		Retention(15*time.Minute).
		Resilence(3, 3*time.Second, "test", "1").
		StepFn("Step 1", "Step 1", func(flow *workflow.Instance, ctx et.Json) (et.Json, error) {
			logs.Logf("Respuesta desde step 1, contexto:", ctx.ToString())
			atrib := fmt.Sprintf("step_%d", flow.Current)
			ctx.Set(atrib, "step1")

			// flow.Done()
			flow.Stop()
			// flow.Goto(2)

			time.Sleep(3 * time.Second)

			return ctx, nil
		}, false).
		IfElse(`test == "test"`, 3, 2).
		StepFn("Step 2", "Step 2", func(flow *workflow.Instance, ctx et.Json) (et.Json, error) {
			logs.Logf("Respuesta desde step 2, con este contexto:", ctx.ToString())
			atrib := fmt.Sprintf("step_%d", flow.Current)
			ctx.Set(atrib, "step2")

			// guardar en el Oss

			return ctx, nil
		}, true).
		Rollback(func(flow *workflow.Instance, ctx et.Json) (et.Json, error) {
			logs.Logf("Respuesta desde rollback 2, con este contexto:", ctx.ToString())
			atrib := fmt.Sprintf("rollback_%d", flow.Current)
			ctx.Set(atrib, "step2")

			return ctx, nil
		}).
		StepFn("Step 3", "Step 3", func(flow *workflow.Instance, ctx et.Json) (et.Json, error) {
			logs.Logf("Respuesta desde step 3, con este contexto:", ctx.ToString())
			atrib := fmt.Sprintf("step_%d", flow.Current)
			ctx.Set(atrib, "step3")

			return ctx, nil
		}, false)

	result, err := workflow.Run("1234", "ventas", 0, et.Json{
		"cedula": "91499023",
	}, et.Json{
		"test": "test",
	}, "test")
	if err != nil {
		logs.Error(err)
	} else {
		logs.Logf("Result 1:", result.ToString())
	}

	// go func() {
	// 	result, err := workflow.Run("", "ventas", 2, et.Json{
	// 		"cedula": "91499023",
	// 	}, et.Json{
	// 		"test": "test",
	// 	}, "test")
	// 	if err != nil {
	// 		console.Error(err)
	// 	} else {
	// 		console.Debug("Result 2:", result.ToString())
	// 	}
	// }()

	// result, err := workflow.Continue("", et.Json{
	// 	"cedula": "91499023",
	// }, et.Json{
	// 	"test": "test",
	// }, "test")
	// if err != nil {
	// 	console.Error(err)
	// } else {
	// 	console.Debug("Result 2:", result.ToString())
	// }

	// go func() {
	// 	result, err := workflow.Run("", "ventas", 2, et.Json{
	// 		"cedula": "91499023",
	// 	}, et.Json{
	// 		"test": "test",
	// 	}, "test")
	// 	if err != nil {
	// 		console.Error(err)
	// 	} else {
	// 		console.Debug("Result:", result.ToString())
	// 	}
	// }()
}

func HttpVenta(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	tag := r.PathValue("tag")
	serviceId := body.Str("serviceId")
	tags := et.Json{
		"cedula": "91499023",
		"codigo": "112342",
	}
	step := body.Int("step")
	createdBy := claim.ClientName(r)
	result, err := workflow.Run(serviceId, tag, step, tags, body, createdBy)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result,
	})

}
