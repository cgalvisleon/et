package crontab

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

const testHeartbeat = 50 * time.Millisecond

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

func TestJobStatusIsFinal(t *testing.T) {
	for status, final := range map[JobStatus]bool{
		Pending: false, Scheduled: false, Running: false, Paused: false,
		Finished: true, Missed: true, Failed: true, Canceled: true,
	} {
		if status.IsFinal() != final {
			t.Errorf("%s.IsFinal() = %v", status, !final)
		}
	}
}

// ---- Bus en memoria: varias réplicas en un proceso, sin NATS ----

// memNet: la red compartida; cada réplica usa su propio client()
type memNet struct {
	mu   sync.Mutex
	subs map[string][]*memSub
	rr   map[string]int
}

type memSub struct {
	owner *memClient
	queue string
	ch    chan et.Json
	stop  chan struct{}
}

type memClient struct {
	net *memNet
}

func newMemNet() *memNet {
	return &memNet{subs: map[string][]*memSub{}, rr: map[string]int{}}
}

func (n *memNet) client() *memClient {
	return &memClient{net: n}
}

// Publish entrega una copia (como NATS) a cada suscripción simple y a una de cada cola, en orden por suscripción
func (c *memClient) Publish(channel string, data et.Json) error {
	bt, err := json.Marshal(data)
	if err != nil {
		return err
	}
	n := c.net
	n.mu.Lock()
	var targets []*memSub
	queues := map[string][]*memSub{}
	for _, sub := range n.subs[channel] {
		if sub.queue == "" {
			targets = append(targets, sub)
		} else {
			queues[sub.queue] = append(queues[sub.queue], sub)
		}
	}
	for q, subs := range queues {
		key := channel + "|" + q
		targets = append(targets, subs[n.rr[key]%len(subs)])
		n.rr[key]++
	}
	n.mu.Unlock()

	for _, sub := range targets {
		var copy et.Json
		json.Unmarshal(bt, &copy)
		select {
		case sub.ch <- copy:
		case <-sub.stop:
		}
	}
	return nil
}

func (c *memClient) add(channel, queue string, fn func(et.Json)) error {
	sub := &memSub{owner: c, queue: queue, ch: make(chan et.Json, 4096), stop: make(chan struct{})}
	go func() {
		for {
			select {
			case data := <-sub.ch:
				fn(data)
			case <-sub.stop:
				return
			}
		}
	}()
	c.net.mu.Lock()
	c.net.subs[channel] = append(c.net.subs[channel], sub)
	c.net.mu.Unlock()
	return nil
}

func (c *memClient) Subscribe(channel string, fn func(et.Json)) error { return c.add(channel, "", fn) }

func (c *memClient) Queue(channel, queue string, fn func(et.Json)) error {
	return c.add(channel, queue, fn)
}

func (c *memClient) Unsubscribe(channel string) error {
	n := c.net
	n.mu.Lock()
	defer n.mu.Unlock()
	kept := n.subs[channel][:0]
	for _, sub := range n.subs[channel] {
		if sub.owner == c {
			close(sub.stop)
		} else {
			kept = append(kept, sub)
		}
	}
	n.subs[channel] = kept
	return nil
}

// ---- Store en memoria ----

// memStore: guarda el JSON de cada job y cuenta las escrituras
type memStore struct {
	mu     sync.Mutex
	data   map[string]et.Json
	writes atomic.Int64
}

func newMemStore() *memStore {
	return &memStore{data: map[string]et.Json{}}
}

func (s *memStore) Set(collection, id, ownerId string, obj any) error {
	bt, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	var doc et.Json
	if err := json.Unmarshal(bt, &doc); err != nil {
		return err
	}
	s.writes.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = doc
	return nil
}

func (s *memStore) Get(collection, id string, dest any) (bool, error) { return false, nil }

func (s *memStore) Delete(collection, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

func (s *memStore) Query(query et.Json) (et.Items, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := et.Items{Ok: true}
	for _, doc := range s.data {
		if doc.Str("crontab") == query.Str("crontab") {
			result.Result = append(result.Result, doc)
		}
	}
	result.Count = len(result.Result)
	return result, nil
}

func (s *memStore) get(key string) (et.Json, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.data[key]
	return doc, ok
}

// ---- Ayudas ----

// counter: cuenta los disparos por valor de params.v
type counter struct {
	mu    sync.Mutex
	shots map[string]int
}

func newCounter() *counter {
	return &counter{shots: map[string]int{}}
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

// replica: una réplica de tag sobre net, cerrada al terminar la prueba
func replica(t *testing.T, net *memNet, tag string, store Store, opts ...Option) *Crontab {
	t.Helper()
	c, err := newCrontab(tag, store, net.client(), testHeartbeat, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// waitFor espera hasta d a que cond se cumpla
func waitFor(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("no se cumplió: %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// waitStatus espera hasta d a que el job tag llegue a want en c y devuelve su estado
func waitStatus(t *testing.T, c *Crontab, tag string, want JobStatus, d time.Duration) et.Json {
	t.Helper()
	var job et.Json
	waitFor(t, d, tag+" en "+string(want), func() bool {
		job, _ = c.GetJob(tag)
		return job.Str("status") == string(want)
	})
	return job
}

func soon(d time.Duration) time.Time { return timezone.Now().Add(d) }

// ---- Pruebas sin NATS ----

// TestStates: el ciclo de vida del job y el resultado de cada corrida, en una réplica.
func TestStates(t *testing.T) {
	store := newMemStore()
	c := replica(t, newMemNet(), "states", store).SetRunTimeout(400 * time.Millisecond)

	t.Run("repetitions → finished", func(t *testing.T) {
		count := newCounter()
		if _, err := c.CronJob("rep", "", "* * * * * *", 2, et.Json{"v": "r"}, count.fn); err != nil {
			t.Fatal(err)
		}
		job := waitStatus(t, c, "rep", Finished, 4*time.Second)
		if job.Int("runs") != 2 || count.of("r") != 2 || job.Json("last_run").Str("status") != string(RunSucceeded) {
			t.Fatalf("repeticiones 2: %v, disparos %d", job, count.of("r"))
		}
		if doc, _ := store.get("states:rep"); doc.Str("status") != string(Finished) {
			t.Fatalf("el Store dice %s", doc.Str("status"))
		}
		if err := c.job("rep").start(); err == nil {
			t.Fatal("un job terminado se reanudó")
		}
	})

	t.Run("error del worker → failed", func(t *testing.T) {
		fail := func(et.Json) error { return errors.New("sin conexión") }
		if _, err := c.ScheduleJob("err", "", soon(200*time.Millisecond), nil, fail); err != nil {
			t.Fatal(err)
		}
		run := waitStatus(t, c, "err", Failed, 2*time.Second).Json("last_run")
		if run.Str("status") != string(RunFailed) || run.Str("error") != "sin conexión" || run.Str("host_name") == "" {
			t.Fatalf("corrida: %v", run)
		}
	})

	t.Run("panic → failed", func(t *testing.T) {
		boom := func(et.Json) error { panic("boom") }
		if _, err := c.ScheduleJob("panic", "", soon(200*time.Millisecond), nil, boom); err != nil {
			t.Fatal(err)
		}
		run := waitStatus(t, c, "panic", Failed, 2*time.Second).Json("last_run")
		if run.Str("status") != string(RunFailed) || !strings.Contains(run.Str("error"), "boom") {
			t.Fatalf("corrida: %v", run)
		}
	})

	t.Run("sin respuesta → timeout", func(t *testing.T) {
		slow := func(et.Json) error { time.Sleep(time.Second); return nil }
		if _, err := c.ScheduleJob("slow", "", soon(200*time.Millisecond), nil, slow); err != nil {
			t.Fatal(err)
		}
		run := waitStatus(t, c, "slow", Failed, 2*time.Second).Json("last_run")
		if run.Str("status") != string(RunTimeout) {
			t.Fatalf("corrida: %v", run)
		}
		time.Sleep(1200 * time.Millisecond) // la respuesta tardía se ignora
		if run := c.job("slow").ToJson().Json("last_run"); run.Str("status") != string(RunTimeout) {
			t.Fatalf("la respuesta tardía cambió la corrida: %v", run)
		}
	})

	t.Run("reintentos hasta lograrlo", func(t *testing.T) {
		var tries atomic.Int32
		flaky := func(et.Json) error {
			if tries.Add(1) < 3 {
				return errors.New("todavía no")
			}
			return nil
		}
		if _, err := c.ScheduleJob("retry", "", soon(100*time.Millisecond), nil, flaky, WithRetries(2, 50*time.Millisecond)); err != nil {
			t.Fatal(err)
		}
		run := waitStatus(t, c, "retry", Finished, 2*time.Second).Json("last_run")
		if run.Str("status") != string(RunSucceeded) || run.Int("attempt") != 3 {
			t.Fatalf("corrida: %v", run)
		}
	})

	t.Run("reintentos agotados → failed", func(t *testing.T) {
		fail := func(et.Json) error { return errors.New("no") }
		if _, err := c.ScheduleJob("retry-fail", "", soon(100*time.Millisecond), nil, fail, WithRetries(1, 50*time.Millisecond)); err != nil {
			t.Fatal(err)
		}
		run := waitStatus(t, c, "retry-fail", Failed, 2*time.Second).Json("last_run")
		if run.Str("status") != string(RunFailed) || run.Int("attempt") != 2 {
			t.Fatalf("corrida: %v", run)
		}
	})

	t.Run("disparo con la corrida anterior en curso → skipped", func(t *testing.T) {
		slow := func(et.Json) error { time.Sleep(2200 * time.Millisecond); return nil }
		if _, err := c.CronJob("skip", "", "* * * * * *", 0, nil, slow, WithTimeout(5*time.Second)); err != nil {
			t.Fatal(err)
		}
		time.Sleep(3500 * time.Millisecond)
		job, _ := c.GetJob("skip")
		if job.Int("skipped") < 1 {
			t.Fatalf("sin disparos omitidos: %v", job)
		}
		c.RemoveJob("skip")
		waitFor(t, time.Second, "quitado", func() bool { _, ok := c.GetJob("skip"); return !ok })
		if _, ok := store.get("states:skip"); ok {
			t.Fatal("el job quitado sigue en el Store")
		}
	})

	t.Run("stop → paused, start → scheduled", func(t *testing.T) {
		count := newCounter()
		if _, err := c.CronJob("pause", "", "* * * * * *", 0, et.Json{"v": "p"}, count.fn); err != nil {
			t.Fatal(err)
		}
		waitStatus(t, c, "pause", Scheduled, time.Second)
		c.StopJob("pause")
		waitStatus(t, c, "pause", Paused, 2*time.Second)
		time.Sleep(300 * time.Millisecond) // una corrida en curso termina
		shots := count.of("p")
		time.Sleep(2 * time.Second)
		if count.of("p") != shots {
			t.Fatalf("pausado sigue disparando: %d → %d", shots, count.of("p"))
		}
		c.StartJob("pause")
		waitFor(t, 3*time.Second, "reanudado dispara", func() bool { return count.of("p") > shots })
		c.RemoveJob("pause")
	})

	t.Run("schedule vencido al reanudar → missed", func(t *testing.T) {
		count := newCounter()
		if _, err := c.ScheduleJob("miss", "", soon(500*time.Millisecond), nil, count.fn); err != nil {
			t.Fatal(err)
		}
		c.StopJob("miss")
		waitStatus(t, c, "miss", Paused, time.Second)
		time.Sleep(700 * time.Millisecond)
		c.StartJob("miss")
		waitStatus(t, c, "miss", Missed, time.Second)
		if count.of("") != 0 {
			t.Fatal("el schedule vencido corrió")
		}
	})

	t.Run("validaciones", func(t *testing.T) {
		if _, err := c.CronJob(" ", "", "* * * * *", 0, nil, newCounter().fn); err == nil {
			t.Fatal("tag vacío aceptado")
		}
		if _, err := c.CronJob("x", "", "no es cron", 0, nil, newCounter().fn); err == nil {
			t.Fatal("expresión inválida aceptada")
		}
		if _, err := c.ScheduleJob("x", "", soon(-time.Minute), nil, newCounter().fn); err == nil {
			t.Fatal("schedule en el pasado aceptado")
		}
		if _, err := newCrontab(" ", nil, newMemNet().client(), testHeartbeat); err == nil {
			t.Fatal("instancia sin tag aceptada")
		}
	})
}

// TestInstances: dos instancias de distinto tag en la misma red, con un job del mismo tag: cada una dispara el suyo
// y quitarlo en una no toca a la otra.
func TestInstances(t *testing.T) {
	net := newMemNet()
	a, b := replica(t, net, "a", nil), replica(t, net, "b", nil)
	if !a.IsLeader() || !b.IsLeader() {
		t.Fatal("cada instancia sola es líder de su grupo")
	}

	count := newCounter()
	a.CronJob("x", "", "* * * * * *", 0, et.Json{"v": "a"}, count.fn)
	b.CronJob("x", "", "* * * * * *", 0, et.Json{"v": "b"}, count.fn)
	waitFor(t, 3*time.Second, "las dos disparan", func() bool { return count.of("a") >= 1 && count.of("b") >= 1 })

	a.RemoveJob("x")
	time.Sleep(300 * time.Millisecond)
	ca, cb := count.of("a"), count.of("b")
	time.Sleep(2 * time.Second)
	if count.of("a") != ca {
		t.Fatalf("el job quitado en a sigue disparando: %d → %d", ca, count.of("a"))
	}
	if count.of("b") <= cb {
		t.Fatal("quitar el job en a detuvo el de b")
	}
}

// TestFailover: dos réplicas del mismo tag registran el mismo job: solo la más antigua lo programa, la otra ve su
// estado, y al cerrarse la líder la otra sigue sin perder la cuenta.
func TestFailover(t *testing.T) {
	net := newMemNet()
	a := replica(t, net, "group", nil)
	b := replica(t, net, "group", nil)
	if !a.IsLeader() || b.IsLeader() {
		t.Fatalf("líder: a=%v b=%v", a.IsLeader(), b.IsLeader())
	}

	count := newCounter()
	if _, err := a.CronJob("tick", "", "* * * * * *", 0, et.Json{"v": "t"}, count.fn); err != nil {
		t.Fatal(err)
	}
	if _, err := b.CronJob("tick", "", "* * * * * *", 0, et.Json{"v": "t"}, count.fn); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3500 * time.Millisecond)
	if n := count.of("t"); n < 2 || n > 4 {
		t.Fatalf("un solo programador: %d disparos en 3,5 s", n)
	}
	ja, _ := a.GetJob("tick")
	jb, _ := b.GetJob("tick")
	if ja.Str("rev") != jb.Str("rev") || jb.Int("runs") < ja.Int("runs")-1 {
		t.Fatalf("la réplica no ve el estado: a=%v b=%v", ja.Int("runs"), jb.Int("runs"))
	}

	runs := ja.Int("runs")
	a.Close()
	waitFor(t, time.Second, "b es líder", b.IsLeader)
	waitFor(t, 3*time.Second, "b sigue disparando", func() bool {
		job, _ := b.GetJob("tick")
		return job.Int("runs") > runs
	})
}

// TestReplace: otra definición con el mismo tag, registrada en cualquier réplica, reemplaza a la anterior en todas.
func TestReplace(t *testing.T) {
	net := newMemNet()
	a := replica(t, net, "replace", nil)
	b := replica(t, net, "replace", nil)

	count := newCounter()
	a.CronJob("r", "", "* * * * * *", 0, et.Json{"v": "1"}, count.fn)
	time.Sleep(200 * time.Millisecond)
	b.CronJob("r", "", "* * * * * *", 0, et.Json{"v": "2"}, count.fn)
	waitFor(t, time.Second, "a tiene la nueva", func() bool {
		job, _ := a.GetJob("r")
		return job.Json("params").Str("v") == "2"
	})
	time.Sleep(300 * time.Millisecond)
	v1 := count.of("1")
	time.Sleep(2 * time.Second)
	if count.of("1") != v1 || count.of("2") < 1 {
		t.Fatalf("reemplazo: v1 %d → %d, v2 %d", v1, count.of("1"), count.of("2"))
	}
}

// TestPaused: WithPaused registra el job ya en pausa: una definición nueva registrada en una réplica que no es la
// líder reemplaza a la programada sin dispararse (no hay StopJob que pueda llegar antes que el job), la líder la guarda
// en pausa y StartJob la reanuda.
func TestPaused(t *testing.T) {
	store := newMemStore()
	net := newMemNet()
	a := replica(t, net, "paused", store)
	b := replica(t, net, "paused", store)
	if !a.IsLeader() || b.IsLeader() {
		t.Fatalf("líder: a=%v b=%v", a.IsLeader(), b.IsLeader())
	}

	count := newCounter()
	if _, err := a.CronJob("p", "", "* * * * * *", 0, et.Json{"v": "1"}, count.fn); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 3*time.Second, "la primera dispara", func() bool { return count.of("1") >= 1 })

	job, err := b.CronJob("p", "", "* * * * * *", 0, et.Json{"v": "2"}, count.fn, WithPaused())
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != Paused {
		t.Fatalf("registrado en %s", job.Status)
	}
	waitStatus(t, a, "p", Paused, time.Second)
	waitFor(t, time.Second, "guardado en pausa", func() bool {
		doc, ok := store.get("paused:p")
		return ok && doc.Str("status") == string(Paused) && doc.Json("params").Str("v") == "2"
	})
	time.Sleep(2 * time.Second)
	if n := count.of("2"); n != 0 {
		t.Fatalf("el job en pausa disparó %d veces", n)
	}

	if err := b.StartJob("p"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 3*time.Second, "reanudado dispara", func() bool { return count.of("2") >= 1 })

	// la misma definición con WithPaused conserva el job y su estado
	again, err := a.CronJob("p", "", "* * * * * *", 0, et.Json{"v": "2"}, count.fn, WithPaused())
	if err != nil {
		t.Fatal(err)
	}
	if again.Rev != job.Rev || again.status() == Paused {
		t.Fatalf("la misma definición: rev %s → %s, estado %s", job.Rev, again.Rev, again.status())
	}

	// un schedule nuevo en pausa en la líder no se arma
	if _, err := a.ScheduleJob("s", "", soon(300*time.Millisecond), et.Json{"v": "s"}, count.fn, WithPaused()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(800 * time.Millisecond)
	if count.of("s") != 0 {
		t.Fatal("el schedule en pausa disparó")
	}
	waitStatus(t, b, "s", Paused, time.Second)
}

// TestConcurrentStart: dos réplicas que arrancan y registran el mismo job a la vez: queda una sola programándolo.
func TestConcurrentStart(t *testing.T) {
	net := newMemNet()
	var a, b *Crontab
	var wg sync.WaitGroup
	wg.Go(func() { a = replica(t, net, "concurrent", nil) })
	wg.Go(func() { b = replica(t, net, "concurrent", nil) })
	wg.Wait()

	count := newCounter()
	wg.Go(func() { a.CronJob("c", "", "* * * * * *", 0, et.Json{"v": "c"}, count.fn) })
	wg.Go(func() { b.CronJob("c", "", "* * * * * *", 0, et.Json{"v": "c"}, count.fn) })
	wg.Wait()

	waitFor(t, time.Second, "una sola líder", func() bool { return a.IsLeader() != b.IsLeader() })
	time.Sleep(300 * time.Millisecond)
	start := count.of("c")
	time.Sleep(3 * time.Second)
	if n := count.of("c") - start; n < 2 || n > 4 {
		t.Fatalf("un solo programador: %d disparos en 3 s", n)
	}
}

// TestReload: los jobs vuelven del Store al arrancar, y registrar la misma definición conserva su estado. Solo la
// líder escribe, una vez por corrida.
func TestReload(t *testing.T) {
	store := newMemStore()
	net := newMemNet()
	first := replica(t, net, "reload", store)
	count := newCounter()
	job, err := first.CronJob("p", "", "* * * * * *", 0, et.Json{"v": "p"}, count.fn)
	if err != nil {
		t.Fatal(err)
	}
	rev := job.Rev
	waitFor(t, 4*time.Second, "dos corridas", func() bool { j, _ := first.GetJob("p"); return j.Int("runs") >= 2 })
	first.Close()
	before, _ := store.get("reload:p")
	runs := before.Int("runs")
	if w := store.writes.Load(); w > int64(runs)+3 {
		t.Fatalf("escrituras: %d para %d corridas", w, runs)
	}

	second := replica(t, net, "reload", store)
	again, err := second.CronJob("p", "", "* * * * * *", 0, et.Json{"v": "p"}, count.fn)
	if err != nil {
		t.Fatal(err)
	}
	if again.Rev != rev {
		t.Fatalf("la misma definición creó otra revisión: %s → %s", rev, again.Rev)
	}
	waitFor(t, 3*time.Second, "sigue la cuenta", func() bool { j, _ := second.GetJob("p"); return j.Int("runs") > runs })
}

// TestHandlerCatchUp: un schedule que quedó programado en el Store y cuya hora pasó con todas las réplicas caídas corre
// una vez al arrancar otra con WithHandler (catch-up dentro de New) y queda finished; registrarlo de nuevo con la
// fecha pasada conserva el job, y una fecha pasada sin ese job sigue siendo un error.
func TestHandlerCatchUp(t *testing.T) {
	store := newMemStore()
	net := newMemNet()
	before := newCounter()
	first := replica(t, net, "catchup", store, WithHandler(before.fn))
	at := soon(400 * time.Millisecond)
	params := et.Json{"v": "s"}
	job, err := first.ScheduleJob("s", "", at, params, nil)
	if err != nil {
		t.Fatal(err)
	}
	rev := job.Rev
	waitFor(t, time.Second, "guardado programado", func() bool {
		doc, ok := store.get("catchup:s")
		return ok && doc.Str("status") == string(Scheduled)
	})
	first.Close()
	time.Sleep(time.Until(at) + 200*time.Millisecond)
	if before.of("s") != 0 {
		t.Fatal("la réplica cerrada disparó")
	}

	count := newCounter()
	second := replica(t, net, "catchup", store, WithHandler(count.fn)).SetRunTimeout(time.Second)
	got := waitStatus(t, second, "s", Finished, 2*time.Second)
	if count.of("s") != 1 || got.Int("runs") != 1 || got.Json("last_run").Str("status") != string(RunSucceeded) {
		t.Fatalf("catch-up: %v, disparos %d", got, count.of("s"))
	}

	again, err := second.ScheduleJob("s", "", at, params, nil)
	if err != nil {
		t.Fatalf("la misma definición con la fecha pasada: %v", err)
	}
	if again.Rev != rev || again.ToJson().Str("status") != string(Finished) {
		t.Fatalf("no conservó el job: rev %s → %s, %s", rev, again.Rev, again.ToJson().Str("status"))
	}
	if _, err := second.ScheduleJob("s", "", at, et.Json{"v": "otro"}, nil); err == nil {
		t.Fatal("otra definición con la fecha pasada aceptada")
	}
	if _, err := second.ScheduleJob("nuevo", "", at, params, nil); err == nil {
		t.Fatal("un schedule nuevo con la fecha pasada aceptado")
	}
	time.Sleep(300 * time.Millisecond)
	if count.of("s") != 1 {
		t.Fatalf("corrió otra vez: %d", count.of("s"))
	}
}

// TestHandlerStartup: un cron recargado del Store que dispara justo al arrancar (sin que nadie lo registre otra vez)
// lo atiende el handler; y una réplica que recibe el job de la líder también lo atiende.
func TestHandlerStartup(t *testing.T) {
	store := newMemStore()
	net := newMemNet()
	first := replica(t, net, "startup", store, WithHandler(newCounter().fn))
	if _, err := first.CronJob("c", "", "* * * * * *", 0, et.Json{"v": "c"}, nil); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 3*time.Second, "una corrida", func() bool { j, _ := first.GetJob("c"); return j.Int("runs") >= 1 })
	first.Close()

	a, b := newCounter(), newCounter()
	leader := replica(t, net, "startup", store, WithHandler(a.fn)).SetRunTimeout(500 * time.Millisecond)
	waitFor(t, 2500*time.Millisecond, "la líder sola atiende el job recargado", func() bool {
		j, _ := leader.GetJob("c")
		return a.of("c") >= 1 && j.Json("last_run").Str("status") == string(RunSucceeded)
	})
	follower := replica(t, net, "startup", store, WithHandler(b.fn))
	if !leader.IsLeader() || follower.IsLeader() {
		t.Fatalf("líder: %v %v", leader.IsLeader(), follower.IsLeader())
	}
	time.Sleep(3500 * time.Millisecond)
	job, _ := leader.GetJob("c")
	if a.of("c")+b.of("c") < 3 || job.Int("skipped") != 0 || job.Json("last_run").Str("status") == string(RunTimeout) {
		t.Fatalf("disparos sin atender: líder %d, réplica %d, %v", a.of("c"), b.of("c"), job)
	}
	if b.of("c") == 0 {
		t.Fatal("la réplica no atiende el job que le difundió la líder")
	}
}

// TestHandlerRequired: sin fn ni handler, CronJob y ScheduleJob fallan con un error claro; un fn propio sigue valiendo
// en una instancia con handler.
func TestHandlerRequired(t *testing.T) {
	net := newMemNet()
	c := replica(t, net, "nohandler", nil)
	if _, err := c.CronJob("x", "", "* * * * * *", 0, nil, nil); err == nil || !strings.Contains(err.Error(), "WithHandler") {
		t.Fatalf("CronJob sin fn: %v", err)
	}
	if _, err := c.ScheduleJob("y", "", soon(time.Minute), nil, nil); err == nil || !strings.Contains(err.Error(), "WithHandler") {
		t.Fatalf("ScheduleJob sin fn: %v", err)
	}
	if _, ok := c.GetJob("x"); ok {
		t.Fatal("quedó el job sin quien lo atienda")
	}

	handler, own := newCounter(), newCounter()
	h := replica(t, net, "withhandler", nil, WithHandler(handler.fn))
	if _, err := h.CronJob("z", "", "* * * * * *", 0, et.Json{"v": "z"}, own.fn); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 3*time.Second, "fn propio", func() bool { return own.of("z") >= 1 })
	if handler.of("z") != 0 {
		t.Fatal("el handler atendió un job con fn propio")
	}
}

// ---- Con NATS ----

// TestJobs: las funciones del paquete sobre la instancia global, con NATS (NATS_HOST); sin él se salta.
func TestJobs(t *testing.T) {
	if os.Getenv("NATS_HOST") == "" {
		t.Skip("sin NATS_HOST")
	}
	if err := Load("test-"+reg.UUID(), nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { Default().Close() })

	count := newCounter()
	tag := "test:" + reg.UUID()
	if _, err := CronJob(tag, "", "* * * * * *", 0, et.Json{"v": "a"}, count.fn); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3500 * time.Millisecond)
	if a := count.of("a"); a < 2 || a > 4 {
		t.Fatalf("cron cada segundo: %d disparos en 3,5 s", a)
	}
	job, ok := GetJob(tag)
	if !ok || job.Json("last_run").Str("status") != string(RunSucceeded) {
		t.Fatalf("estado: %v", job)
	}

	if err := StopJob(tag); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, Default(), tag, Paused, 2*time.Second)
	if err := RemoveJob(tag); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 2*time.Second, "quitado", func() bool { _, ok := GetJob(tag); return !ok })

	at := soon(1500 * time.Millisecond)
	if _, err := ScheduleJob("test:"+reg.UUID(), "", at, et.Json{"v": "s"}, count.fn); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 3*time.Second, "schedule", func() bool { return count.of("s") == 1 })
}
