package jwf

import (
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
)

/**
* memoryCache: Un cacheStore en memoria para probar el binding sin Redis
* @return cacheStore, map[string]string
**/
func memoryCache() (cacheStore, map[string]string) {
	data := map[string]string{}
	return cacheStore{
		set: func(key, text string, expiration time.Duration) { data[key] = text },
		get: func(key string) (string, bool, error) {
			text, ok := data[key]
			return text, ok, nil
		},
		del: func(key string) error {
			delete(data, key)
			return nil
		},
	}, data
}

/**
* TestCacheBinding: set guarda como texto (otro tipo no), get con su valor por defecto, json/items/item y delete
**/
func TestCacheBinding(t *testing.T) {
	store, data := memoryCache()
	binding := cacheBinding(store)
	set := binding["set"].(func(string, interface{}, time.Duration) interface{})
	get := binding["get"].(func(string, string) string)
	asJson := binding["json"].(func(string) et.Json)
	items := binding["items"].(func(string) et.Items)
	remove := binding["delete"].(func(string) bool)

	if got := set("n", 7, time.Minute); got != "7" || data["n"] != "7" {
		t.Fatalf("set de un número: %v %q", got, data["n"])
	}
	if got := set("x", struct{}{}, time.Minute); got != nil {
		t.Fatalf("un tipo desconocido no se guarda: %v", got)
	}
	if _, ok := data["x"]; ok {
		t.Fatal("un tipo desconocido no se guarda")
	}
	if get("n", "d") != "7" || get("falta", "d") != "d" {
		t.Fatal("get con valor por defecto")
	}

	set("j", et.Json{"a": 1}, time.Minute)
	if asJson("j").Int("a") != 1 || len(asJson("falta")) != 0 {
		t.Fatalf("json: %v", asJson("j"))
	}
	data["malo"] = "{"
	if len(asJson("malo")) != 0 {
		t.Fatal("un json inválido da vacío")
	}

	set("i", et.Items{Ok: true, Count: 1, Result: []et.Json{{"a": 2}}}, time.Minute)
	if got := items("i"); !got.Ok || got.Count != 1 {
		t.Fatalf("items: %v", got)
	}

	if !remove("n") || get("n", "d") != "d" {
		t.Fatal("delete")
	}
}

/**
* TestCacheOfWithoutConnection: Sin conexión propia el binding es el del paquete, que sin Load no guarda nada
**/
func TestCacheOfWithoutConnection(t *testing.T) {
	binding := CacheOf(nil)
	get := binding["get"].(func(string, string) string)
	if get("k", "d") != "d" {
		t.Fatal("sin cache cargado get da el valor por defecto")
	}
}
