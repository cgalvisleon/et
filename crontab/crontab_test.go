package crontab

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

func TestParseSpec(t *testing.T) {
	cases := []struct {
		spec, want string
		ok         bool
	}{
		{"*/5 * * * *", "0 */5 * * * *", true},
		{"0 8 * * 1-5", "0 0 8 * * 1-5", true},
		{"30 0 8 * * *", "30 0 8 * * *", true},
		{"@daily", "@daily", true},
		{"@every 1h", "@every 1h", true},
		{"61 * * * *", "", false},
		{"* * *", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, err := ParseSpec(c.spec)
		if (err == nil) != c.ok || got != c.want {
			t.Errorf("ParseSpec(%q) = %q, %v; quería %q, ok %v", c.spec, got, err, c.want, c.ok)
		}
	}
}

func TestNextOf(t *testing.T) {
	from := time.Date(2026, 10, 19, 10, 30, 0, 0, time.UTC)
	next, err := NextOf("0 8 * * *", from)
	if err != nil || !next.Equal(time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("NextOf diario a las 8: %v, %v", next, err)
	}
	if _, err := NextOf("x", from); err == nil {
		t.Fatal("expresión inválida aceptada")
	}
}

// counter: cuenta los disparos por valor de params.v
type counter struct {
	mu    sync.Mutex
	shots map[string]int
}

func (s *counter) fn(params et.Json) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shots[params.Str("v")]++
	return nil
}

func (s *counter) of(v string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shots[v]
}

// TestJobs necesita NATS (NATS_HOST); sin él se salta
func TestJobs(t *testing.T) {
	if os.Getenv("NATS_HOST") == "" {
		t.Skip("sin NATS_HOST")
	}

	if err := Load("test-"+reg.UUID(), nil); err != nil {
		t.Fatal(err)
	}

	count := &counter{shots: map[string]int{}}
	tag := "test:" + reg.UUID()

	// Un cron de cada segundo
	if _, err := CronJob(tag, "", "* * * * * *", 0, et.Json{"v": "a"}, count.fn); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3500 * time.Millisecond)
	a := count.of("a")
	if a < 2 || a > 4 {
		t.Fatalf("cron cada segundo: %d disparos en 3,5 s", a)
	}

	// Registrar el mismo tag lo reemplaza: ya no dispara "a" y "b" no se duplica
	if _, err := CronJob(tag, "", "* * * * * *", 0, et.Json{"v": "b"}, count.fn); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	a = count.of("a")
	time.Sleep(3 * time.Second)
	if count.of("a") != a {
		t.Fatalf("el job reemplazado sigue disparando: %d → %d", a, count.of("a"))
	}
	b := count.of("b")
	if b < 2 || b > 4 {
		t.Fatalf("job de reemplazo: %d disparos en ~3,5 s (duplicado o parado)", b)
	}

	// RemoveJob lo quita
	if err := RemoveJob(tag); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	b = count.of("b")
	time.Sleep(2 * time.Second)
	if count.of("b") != b {
		t.Fatalf("el job quitado sigue disparando: %d → %d", b, count.of("b"))
	}

	// Repeticiones: se detiene al llegar
	tag = "test:" + reg.UUID()
	if _, err := CronJob(tag, "", "* * * * * *", 2, et.Json{"v": "r"}, count.fn); err != nil {
		t.Fatal(err)
	}
	time.Sleep(4 * time.Second)
	if r := count.of("r"); r != 2 {
		t.Fatalf("repeticiones 2: %d disparos", r)
	}

	// Schedule: una vez en la fecha; una fecha pasada se rechaza
	at := timezone.Now().Add(1500 * time.Millisecond)
	if _, err := ScheduleJob("test:"+reg.UUID(), "", at, et.Json{"v": "s"}, count.fn); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	if s := count.of("s"); s != 1 {
		t.Fatalf("schedule: %d disparos", s)
	}
	past := timezone.Now().Add(-time.Minute)
	if _, err := ScheduleJob("test:"+reg.UUID(), "", past, nil, count.fn); err == nil {
		t.Fatal("schedule en el pasado aceptado")
	}

	// Validaciones
	if _, err := CronJob(" ", "", "* * * * *", 0, nil, count.fn); err == nil {
		t.Fatal("tag vacío aceptado")
	}
	if _, err := CronJob("test:x", "", "no es cron", 0, nil, count.fn); err == nil {
		t.Fatal("expresión inválida aceptada")
	}
}
