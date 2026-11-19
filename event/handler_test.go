package event

import (
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
)

// Pruebas de las suscripciones por clave: necesitan un NATS (NATS_HOST, p. ej. uno desechable con
// docker run --rm -p 14225:4222 nats); sin NATS_HOST se saltan

/**
* connectOrSkip
* @param t *testing.T
**/
func connectOrSkip(t *testing.T) {
	t.Helper()
	if os.Getenv("NATS_HOST") == "" {
		t.Skip("NATS_HOST no definido")
	}

	if err := Load(); err != nil {
		t.Fatal(err)
	}
}

/**
* waitFor: Espera hasta que cond sea cierta o pase el plazo
* @param cond func() bool
* @return bool
**/
func waitFor(cond func() bool) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

func TestWithoutConnection(t *testing.T) {
	if conn != nil {
		t.Skip("con conexión")
	}

	if err := SubscribeWith("k", "c", nil); err == nil {
		t.Fatal("SubscribeWith sin conexión debe fallar")
	}
	if err := UnsubscribeKey("k"); err == nil {
		t.Fatal("UnsubscribeKey sin conexión debe fallar")
	}
	if HasKey("k") {
		t.Fatal("HasKey sin conexión")
	}
}

func TestSubscribeWithSameChannel(t *testing.T) {
	connectOrSkip(t)
	channel := "test:keyed:" + time.Now().Format("150405.000000")
	var a, b atomic.Int64
	if err := SubscribeWith("a", channel, func(Message) { a.Add(1) }); err != nil {
		t.Fatal(err)
	}
	if err := SubscribeWith("b", channel, func(Message) { b.Add(1) }); err != nil {
		t.Fatal(err)
	}
	conn.Flush()

	Publish(channel, et.Json{"n": 1})
	if !waitFor(func() bool { return a.Load() == 1 && b.Load() == 1 }) {
		t.Fatalf("los dos oyentes reciben: a=%d b=%d", a.Load(), b.Load())
	}

	// Cortar uno deja vivo el otro
	if err := UnsubscribeKey("a"); err != nil {
		t.Fatal(err)
	}
	if HasKey("a") || !HasKey("b") {
		t.Fatal("UnsubscribeKey debe quitar solo su clave")
	}
	Publish(channel, et.Json{"n": 2})
	if !waitFor(func() bool { return b.Load() == 2 }) {
		t.Fatalf("b sigue recibiendo: b=%d", b.Load())
	}
	time.Sleep(100 * time.Millisecond)
	if a.Load() != 1 {
		t.Fatalf("a ya no recibe: a=%d", a.Load())
	}

	// La misma clave reemplaza la suscripción anterior
	var c atomic.Int64
	if err := SubscribeWith("b", channel, func(Message) { c.Add(1) }); err != nil {
		t.Fatal(err)
	}
	conn.Flush()
	Publish(channel, et.Json{"n": 3})
	if !waitFor(func() bool { return c.Load() == 1 }) {
		t.Fatal("la suscripción nueva de b recibe")
	}
	time.Sleep(100 * time.Millisecond)
	if b.Load() != 2 {
		t.Fatalf("la suscripción vieja de b ya no recibe: b=%d", b.Load())
	}

	UnsubscribeKey("b")
	if err := UnsubscribeKey("no-existe"); err != nil {
		t.Fatal(err)
	}
}

func TestQueueWithAndStackWith(t *testing.T) {
	connectOrSkip(t)
	channel := "test:queue:" + time.Now().Format("150405.000000")
	var q1, q2, other, s1, s2 atomic.Int64
	steps := []error{
		QueueWith("q1", channel, "grupo", func(Message) { q1.Add(1) }),
		QueueWith("q2", channel, "grupo", func(Message) { q2.Add(1) }),
		QueueWith("other", channel, "otro", func(Message) { other.Add(1) }),
		StackWith("s1", channel, func(Message) { s1.Add(1) }),
		StackWith("s2", channel, func(Message) { s2.Add(1) }),
	}
	for _, err := range steps {
		if err != nil {
			t.Fatal(err)
		}
	}
	conn.Flush()

	const total = 20
	for i := range total {
		Publish(channel, et.Json{"n": i})
	}

	ok := waitFor(func() bool {
		return q1.Load()+q2.Load() == total && other.Load() == total && s1.Load()+s2.Load() == total
	})
	if !ok {
		t.Fatalf("cada grupo recibe cada mensaje una vez: grupo=%d otro=%d stack=%d", q1.Load()+q2.Load(), other.Load(), s1.Load()+s2.Load())
	}

	if err := QueueWith("q3", channel, "", nil); err == nil {
		t.Fatal("QueueWith sin cola debe fallar")
	}
	for _, key := range []string{"q1", "q2", "other", "s1", "s2"} {
		UnsubscribeKey(key)
	}
}

func TestSubscribeByChannelUnchanged(t *testing.T) {
	connectOrSkip(t)
	channel := "test:channel:" + time.Now().Format("150405.000000")
	var keyed, legacy atomic.Int64
	if err := SubscribeWith(channel, channel, func(Message) { keyed.Add(1) }); err != nil {
		t.Fatal(err)
	}
	if err := Subscribe(channel, func(m Message) {
		if m.Myself {
			legacy.Add(1)
		}
	}); err != nil {
		t.Fatal(err)
	}
	conn.Flush()

	// Unsubscribe por canal no toca la suscripción con una clave igual al canal
	Publish(channel, et.Json{})
	waitFor(func() bool { return legacy.Load() == 1 && keyed.Load() == 1 })
	if err := Unsubscribe(channel); err != nil {
		t.Fatal(err)
	}
	Publish(channel, et.Json{})
	if !waitFor(func() bool { return keyed.Load() == 2 }) {
		t.Fatalf("la suscripción por clave sigue: %d", keyed.Load())
	}
	time.Sleep(100 * time.Millisecond)
	if legacy.Load() != 1 {
		t.Fatalf("Subscribe por canal recibe con Myself y Unsubscribe la corta: %d", legacy.Load())
	}
	UnsubscribeKey(channel)
}
