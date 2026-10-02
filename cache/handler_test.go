package cache

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
)

// loadMemory starts the in-memory cache for one test.
func loadMemory(t *testing.T) {
	t.Helper()
	t.Setenv("REDIS_HOST", "")
	if err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	t.Cleanup(Close)
}

// TestGetReportsExistence checks that Get tells a missing key from a stored
// one without an error, including a stored empty string.
func TestGetReportsExistence(t *testing.T) {
	loadMemory(t)

	Set("k:empty", "", time.Hour)
	got, exists, err := Get("k:empty", "def")
	if err != nil || !exists || got != "" {
		t.Fatalf("Get of an empty value = %q, %v, %v; want \"\", true, nil", got, exists, err)
	}

	got, exists, err = Get("k:missing", "def")
	if err != nil || exists || got != "def" {
		t.Fatalf("Get of a missing key = %q, %v, %v; want def, false, nil", got, exists, err)
	}
}

// TestGetWithoutLoad checks that the getters fail when no cache is loaded.
func TestGetWithoutLoad(t *testing.T) {
	Close()

	if got, exists, err := Get("k", "def"); err == nil || exists || got != "def" {
		t.Fatalf("Get = %q, %v, %v; want def, false, error", got, exists, err)
	}

	if got, exists, err := GetInt("k", 7); err == nil || exists || got != 7 {
		t.Fatalf("GetInt = %d, %v, %v; want 7, false, error", got, exists, err)
	}
}

// TestBasicTypesRoundTrip checks that every basic type stored with Set is read
// back by its typed getter.
func TestBasicTypesRoundTrip(t *testing.T) {
	loadMemory(t)

	now := time.Now().Truncate(time.Second)
	Set("k:str", "hello", time.Hour)
	Set("k:int", 42, time.Hour)
	Set("k:int64", int64(1)<<40, time.Hour)
	Set("k:float", 0.0000001, time.Hour)
	Set("k:bool", true, time.Hour)
	Set("k:time", now, time.Hour)
	Set("k:duration", 90*time.Second, time.Hour)
	Set("k:bytes", []byte("raw"), time.Hour)

	if v, ok, err := GetStr("k:str", ""); err != nil || !ok || v != "hello" {
		t.Errorf("GetStr = %q, %v, %v", v, ok, err)
	}
	if v, ok, err := GetInt("k:int", 0); err != nil || !ok || v != 42 {
		t.Errorf("GetInt = %d, %v, %v", v, ok, err)
	}
	if v, ok, err := GetInt64("k:int64", 0); err != nil || !ok || v != int64(1)<<40 {
		t.Errorf("GetInt64 = %d, %v, %v", v, ok, err)
	}
	if v, ok, err := GetFloat("k:float", 0); err != nil || !ok || v != 0.0000001 {
		t.Errorf("GetFloat = %v, %v, %v", v, ok, err)
	}
	if v, ok, err := GetBool("k:bool", false); err != nil || !ok || !v {
		t.Errorf("GetBool = %v, %v, %v", v, ok, err)
	}
	if v, ok, err := GetTime("k:time", time.Time{}); err != nil || !ok || !v.Equal(now) {
		t.Errorf("GetTime = %v, %v, %v; want %v", v, ok, err, now)
	}
	if v, ok, err := GetDuration("k:duration", 0); err != nil || !ok || v != 90*time.Second {
		t.Errorf("GetDuration = %v, %v, %v", v, ok, err)
	}
	if v, ok, err := GetBytes("k:bytes"); err != nil || !ok || string(v) != "raw" {
		t.Errorf("GetBytes = %q, %v, %v", v, ok, err)
	}
}

// TestBasicTypesMissingAndInvalid checks the typed getters return the default
// with exists=false for a missing key, and exists=true plus an error for a
// value that does not parse.
func TestBasicTypesMissingAndInvalid(t *testing.T) {
	loadMemory(t)

	if v, ok, err := GetInt("k:missing", 7); err != nil || ok || v != 7 {
		t.Errorf("GetInt missing = %d, %v, %v; want 7, false, nil", v, ok, err)
	}
	if v, ok, err := GetFloat("k:missing", 1.5); err != nil || ok || v != 1.5 {
		t.Errorf("GetFloat missing = %v, %v, %v; want 1.5, false, nil", v, ok, err)
	}
	if v, ok, err := GetBool("k:missing", true); err != nil || ok || !v {
		t.Errorf("GetBool missing = %v, %v, %v; want true, false, nil", v, ok, err)
	}

	Set("k:text", "not-a-number", time.Hour)
	if v, ok, err := GetInt("k:text", 7); err == nil || !ok || v != 7 {
		t.Errorf("GetInt invalid = %d, %v, %v; want 7, true, error", v, ok, err)
	}
	if v, ok, err := GetDuration("k:text", time.Second); err == nil || !ok || v != time.Second {
		t.Errorf("GetDuration invalid = %v, %v, %v; want 1s, true, error", v, ok, err)
	}
}

// TestStructuredTypes checks GetJson, GetItem, GetItems, GetList and
// GetObject for stored and missing keys.
func TestStructuredTypes(t *testing.T) {
	loadMemory(t)

	Set("k:json", et.Json{"name": "a"}, time.Hour)
	Set("k:item", et.Item{Ok: true, Result: et.Json{"name": "b"}}, time.Hour)
	Set("k:items", et.Items{Ok: true, Count: 1, Result: []et.Json{{"name": "c"}}}, time.Hour)
	Set("k:list", et.List{Rows: 1, All: 1, Count: 1, Page: 1, Result: []et.Json{{"name": "d"}}}, time.Hour)

	if v, ok, err := GetJson("k:json"); err != nil || !ok || v.Str("name") != "a" {
		t.Errorf("GetJson = %v, %v, %v", v, ok, err)
	}
	if v, ok, err := GetItem("k:item"); err != nil || !ok || !v.Ok || v.Result.Str("name") != "b" {
		t.Errorf("GetItem = %v, %v, %v", v, ok, err)
	}
	if v, ok, err := GetItems("k:items"); err != nil || !ok || v.Count != 1 || v.Result[0].Str("name") != "c" {
		t.Errorf("GetItems = %v, %v, %v", v, ok, err)
	}
	if v, ok, err := GetList("k:list"); err != nil || !ok || v.All != 1 || v.Result[0].Str("name") != "d" {
		t.Errorf("GetList = %v, %v, %v", v, ok, err)
	}

	var dest et.Json
	if ok, err := GetObject("k:json", &dest); err != nil || !ok || dest.Str("name") != "a" {
		t.Errorf("GetObject = %v, %v, %v", dest, ok, err)
	}

	if _, ok, err := GetJson("k:missing"); err != nil || ok {
		t.Errorf("GetJson missing: exists = %v, err = %v; want false, nil", ok, err)
	}
	if _, ok, err := GetItem("k:missing"); err != nil || ok {
		t.Errorf("GetItem missing: exists = %v, err = %v; want false, nil", ok, err)
	}
	if _, ok, err := GetItems("k:missing"); err != nil || ok {
		t.Errorf("GetItems missing: exists = %v, err = %v; want false, nil", ok, err)
	}
	if ok, err := GetObject("k:missing", &dest); err != nil || ok {
		t.Errorf("GetObject missing: exists = %v, err = %v; want false, nil", ok, err)
	}
}

// TestGetVerifyConsumesKey checks that GetVerify returns the value once and
// then reports it as missing.
func TestGetVerifyConsumesKey(t *testing.T) {
	loadMemory(t)

	SetVerify("device", "otp", "1234", time.Hour)

	if v, ok, err := GetVerify("device", "otp"); err != nil || !ok || v != "1234" {
		t.Fatalf("first GetVerify = %q, %v, %v; want 1234, true, nil", v, ok, err)
	}
	if v, ok, err := GetVerify("device", "otp"); err != nil || ok || v != "" {
		t.Fatalf("second GetVerify = %q, %v, %v; want \"\", false, nil", v, ok, err)
	}
}

// TestCollectionFindAndObjetGet checks that hash lookups report whether the
// field exists, and ObjetGet no longer fails decoding a missing field.
func TestCollectionFindAndObjetGet(t *testing.T) {
	loadMemory(t)

	if err := ObjetSet("users", "u1", et.Json{"name": "ana"}); err != nil {
		t.Fatalf("ObjetSet: %v", err)
	}

	if _, ok, err := CollectionFind("users", "u1"); err != nil || !ok {
		t.Errorf("CollectionFind = %v, %v; want true, nil", ok, err)
	}
	if v, ok, err := CollectionFind("users", "u2"); err != nil || ok || v != "" {
		t.Errorf("CollectionFind missing = %q, %v, %v; want \"\", false, nil", v, ok, err)
	}

	if v, ok, err := ObjetGet("users", "u1"); err != nil || !ok || v.Str("name") != "ana" {
		t.Errorf("ObjetGet = %v, %v, %v", v, ok, err)
	}
	if _, ok, err := ObjetGet("users", "u2"); err != nil || ok {
		t.Errorf("ObjetGet missing: exists = %v, err = %v; want false, nil", ok, err)
	}
}

// TestHttpGetReportsExistence checks that the GET /cache/{key} response says
// whether the key exists.
func TestHttpGetReportsExistence(t *testing.T) {
	loadMemory(t)

	Set("k:http", "value", time.Hour)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /cache/{key}", HttpGet)

	cases := []struct {
		key    string
		exists bool
		value  string
	}{
		{"k:http", true, "value"},
		{"k:missing", false, ""},
	}

	for _, c := range cases {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cache/"+c.key, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", c.key, rec.Code)
		}

		var body struct {
			Result et.Item `json:"result"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: decoding %q: %v", c.key, rec.Body.String(), err)
		}

		if body.Result.Ok != c.exists || body.Result.Result.Str("value") != c.value || body.Result.Result.Str("key") != c.key {
			t.Errorf("%s: response = %+v; want ok=%v value=%q", c.key, body.Result, c.exists, c.value)
		}
	}
}
