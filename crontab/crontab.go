// Package crontab schedules jobs (cron spec or a single time) across the replicas of a service through NATS.
//
// A Crontab is an instance with its own tag, Store and jobs: New creates as many as needed, and Load keeps one as the
// package's global instance behind the package-level functions (CronJob, ScheduleJob…). Replicas (processes) that use
// the same tag form one group: they announce themselves every heartbeat and the oldest live one is the leader, which
// schedules every job, saves it in the Store and broadcasts its state; the others keep a copy and take over when the
// leader stops announcing itself (see cluster.go). Each fire is handled by one replica that registered the job's fn
// or that has the instance handler (WithHandler, subscribed before the replica can lead) (queue QUEUE_JOBS).
//
// Lifecycle of a job (JobStatus): pending → scheduled ⇄ running → finished | failed; stop → paused, start → scheduled;
// a schedule whose time passed → missed; replaced or removed → canceled. Each fire is a Run: dispatched on the job's
// channel and closed by the worker's reply on the done channel (succeeded/failed) or by the timeout, retried up to
// Retries times. A fire while the previous run is in flight is skipped (Job.Skipped).
//
// Invariants:
//   - Every mutable Job field is read and written under Job.mu, including save(); the definition never changes.
//   - Lock order: Crontab.electMu → Job.mu → Crontab.mu; Job.mu is never taken while holding Crontab.mu.
//   - Only the leader arms timers, dispatches runs, writes the Store and broadcasts state; trigger, endAttempt and
//     retry check isLeader under Job.mu, and elect sets the flag before calling lead/follow.
//   - A newer definition (greater Rev, UUID v7) replaces an older one everywhere; registering the same definition again
//     keeps the job and its state (so a service can register its jobs on every start).
package crontab

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/timezone"
	"github.com/robfig/cron/v3"
)

const (
	packageName = "crontab"
	storeJobs   = "jobs"
	// QUEUE_JOBS: la cola de los canales de jobs; entre varias réplicas, cada disparo lo atiende una sola
	QUEUE_JOBS = "crontab:jobs"
)

var (
	hostName, _ = os.Hostname()
)

/**
* Store: Dónde se guardan los jobs para sobrevivir a un reinicio. Set y Delete usan la clave "<crontab>:<job>";
* Query recibe {"collection": "jobs", "crontab": tag} y debe devolver en Result el JSON de cada job guardado con ese
* campo "crontab" igual a tag. Get no se usa.
**/
type Store interface {
	Set(collection, id, ownerId string, obj any) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	Query(query et.Json) (et.Items, error)
}

/**
* Crontab: Una instancia de programación de jobs con su tag, su Store y sus jobs.
**/
type Crontab struct {
	Tag       string
	HostName  string
	id        string // único por réplica; empieza por la hora de arranque, así el menor es el más antiguo
	bus       bus
	jobs      map[string]*Job
	cron      *cron.Cron
	store     Store
	handler   func(params et.Json) error // el de WithHandler: atiende los jobs sin fn propio
	isDebug   bool
	timeout   time.Duration
	catchUp   time.Duration
	heartbeat time.Duration
	startedAt time.Time
	members   map[string]time.Time // las otras réplicas y cuándo se anunciaron
	leader    bool
	closed    bool
	channels  []string        // las suscripciones de la instancia (sin los canales de jobs)
	listening map[string]bool // los canales de jobs que atiende esta réplica (true: con el handler de la instancia)
	out       chan outMsg     // lo que se difunde sin esperar (desde dentro de Job.mu)
	done      chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	electMu   sync.Mutex
	prefix    string
	evJob     string
	evRemove  string
	evStop    string
	evStart   string
	evDone    string
	evHello   string
	evBye     string
}

// Option: ajustes opcionales de una instancia (New y Load los aceptan al final).
type Option func(*Crontab)

/**
* WithHandler: Un handler para todos los jobs de la instancia que no traen fn propio (CronJob o ScheduleJob con fn
* nil). Queda suscrito al canal de cada job antes de elegir líder: para los jobs recargados del Store dentro de New y
* para los que llegan de la líder, así ningún disparo (ni el de un schedule atrasado al tomar el liderazgo) sale sin
* quien lo atienda. Recibe los params del job; si necesita saber de qué job es, inclúyalo en los params al registrarlo.
* @param fn func(params et.Json) error
* @return Option
**/
func WithHandler(fn func(params et.Json) error) Option {
	return func(c *Crontab) {
		c.handler = fn
	}
}

// outMsg: un mensaje de la cola de salida.
type outMsg struct {
	channel string
	data    et.Json
}

/**
* New: Crea una instancia de Crontab sobre NATS (event.Load) y recarga sus jobs del Store. Sus eventos y canales
* llevan tag: las réplicas con el mismo tag forman un grupo con una líder, y las de otro tag no se ven. Tarda 1,5
* heartbeats (CRONTAB_HEARTBEAT, 1s) en volver: el tiempo de recibir los jobs de la líder. Con WithHandler, los jobs
* recargados quedan atendidos antes de que esta réplica pueda liderar.
* @param tag string, store Store (puede ser nil: entonces el estado vive solo en las réplicas), opts ...Option
* @return *Crontab, error
**/
func New(tag string, store Store, opts ...Option) (*Crontab, error) {
	if err := event.Load(); err != nil {
		return nil, err
	}
	return newCrontab(tag, store, natsBus{}, envar.GetDuration("CRONTAB_HEARTBEAT", time.Second), opts...)
}

/**
* newCrontab: New con el bus y el heartbeat dados (las pruebas usan un bus en memoria y heartbeats cortos).
* @param tag string, store Store, b bus, heartbeat time.Duration, opts ...Option
* @return *Crontab, error
**/
func newCrontab(tag string, store Store, b bus, heartbeat time.Duration, opts ...Option) (*Crontab, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, errors.New(MSG_ERROR_TAG_REQUIRED)
	}

	now := time.Now()
	prefix := fmt.Sprintf("crontab:%s", tag)
	result := &Crontab{
		Tag:       tag,
		HostName:  hostName,
		id:        fmt.Sprintf("%020d-%s", now.UnixNano(), reg.ULID()),
		bus:       b,
		jobs:      make(map[string]*Job),
		cron:      cron.New(cron.WithSeconds(), cron.WithLocation(timezone.Location())),
		store:     store,
		timeout:   envar.GetDuration("CRONTAB_RUN_TIMEOUT", time.Hour),
		catchUp:   envar.GetDuration("CRONTAB_CATCH_UP", time.Hour),
		heartbeat: heartbeat,
		startedAt: now,
		members:   make(map[string]time.Time),
		listening: make(map[string]bool),
		out:       make(chan outMsg, 1024),
		done:      make(chan struct{}),
		prefix:    prefix,
		evJob:     prefix + ":job",
		evRemove:  prefix + ":remove",
		evStop:    prefix + ":stop",
		evStart:   prefix + ":start",
		evDone:    prefix + ":done",
		evHello:   prefix + ":hello",
		evBye:     prefix + ":bye",
	}
	for _, opt := range opts {
		opt(result)
	}

	if err := result.reload(); err != nil {
		return nil, err
	}
	// Los jobs recargados quedan atendidos por el handler antes de que esta réplica pueda liderar y dispararlos
	if err := result.listenReloaded(); err != nil {
		result.unsubscribe()
		return nil, err
	}
	if err := result.subscribe(); err != nil {
		result.unsubscribe()
		return nil, err
	}
	result.cron.Start()
	result.wg.Add(2)
	go func() { defer result.wg.Done(); result.sendLoop() }()
	go func() { defer result.wg.Done(); result.heartbeatLoop() }()

	// Entrar al grupo: la líder envía sus jobs al primer hello. Esperar a tenerlos evita que registrar la misma
	// definición cree una revisión nueva y pierda el estado; al volver ya se sabe si esta réplica es la líder.
	time.Sleep(heartbeat * 3 / 2)
	result.elect()
	return result, nil
}

/**
* reload: Trae los jobs del Store (los cancelados no). La líder los programa cuando lo sea.
* @return error
**/
func (s *Crontab) reload() error {
	if s.store == nil {
		return nil
	}
	items, err := s.store.Query(et.Json{"collection": storeJobs, "crontab": s.Tag})
	if err != nil {
		return err
	}
	for _, data := range items.Result {
		job, err := jobOf(s, data)
		if err != nil {
			logs.Error(err)
			continue
		}
		if job.Crontab == s.Tag && job.Status != Canceled {
			s.jobs[job.ID] = job
		}
	}
	return nil
}

/**
* listenReloaded: Con handler, atiende los disparos de los jobs recargados del Store (sin handler, cada servicio los
* atiende al registrarlos con su fn).
* @return error
**/
func (s *Crontab) listenReloaded() error {
	if s.handler == nil {
		return nil
	}
	s.mu.Lock()
	jobs := s.jobList()
	s.mu.Unlock()
	for _, job := range jobs {
		if err := s.listen(job.Tag, nil); err != nil {
			return err
		}
	}
	return nil
}

/**
* Debug: Registra en el log cada guardado de los jobs.
* @return *Crontab
**/
func (s *Crontab) Debug() *Crontab {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isDebug = true
	return s
}

/**
* SetRunTimeout: Cuánto se espera la respuesta del worker antes de dar un intento por timeout, en los jobs sin
* WithTimeout (CRONTAB_RUN_TIMEOUT, 1h por omisión).
* @param d time.Duration
* @return *Crontab
**/
func (s *Crontab) SetRunTimeout(d time.Duration) *Crontab {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timeout = d
	return s
}

/**
* runTimeout
* @return time.Duration
**/
func (s *Crontab) runTimeout() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.timeout
}

/**
* Close: Deja de programar los jobs, se despide de las réplicas (otra toma el liderazgo sin esperar) y cancela las
* suscripciones, sin cambiar el estado guardado.
* @return error
**/
func (s *Crontab) Close() error {
	s.electMu.Lock()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.electMu.Unlock()
		return nil
	}
	s.closed = true
	s.leader = false
	jobs := s.jobList()
	s.mu.Unlock()
	s.electMu.Unlock()

	for _, job := range jobs {
		job.follow()
	}
	<-s.cron.Stop().Done()
	close(s.done)
	s.wg.Wait()
	s.publish(s.evBye, et.Json{"id": s.id})
	return s.unsubscribe()
}

/**
* publish
* @param channel string, data et.Json
* @return error
**/
func (s *Crontab) publish(channel string, data et.Json) error {
	return s.bus.Publish(channel, data)
}

/**
* send: Encola un mensaje para difundirlo en orden sin esperar (se puede llamar con Job.mu tomado).
* @param channel string, data et.Json
**/
func (s *Crontab) send(channel string, data et.Json) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	data["from"] = s.id
	select {
	case s.out <- outMsg{channel, data}:
	default:
		logs.Alertf(MSG_ERROR_OUT_FULL, s.Tag)
	}
}

/**
* sendLoop: Publica la cola de salida hasta Close (lo pendiente se publica antes de salir).
**/
func (s *Crontab) sendLoop() {
	for {
		select {
		case msg := <-s.out:
			if err := s.publish(msg.channel, msg.data); err != nil {
				logs.Error(err)
			}
		case <-s.done:
			for {
				select {
				case msg := <-s.out:
					s.publish(msg.channel, msg.data)
				default:
					return
				}
			}
		}
	}
}

/**
* saveJob: Guarda el job en el Store con la clave "<crontab>:<job>". Con Job.mu tomado.
* @param job *Job
* @return error
**/
func (s *Crontab) saveJob(job *Job) error {
	s.mu.Lock()
	debug := s.isDebug
	s.mu.Unlock()
	if debug {
		logs.Log(packageName, "save:", job.toJson().ToString())
	}
	if s.store == nil {
		return nil
	}
	return s.store.Set(storeJobs, s.storeKey(job.ID), job.OwnerId, job)
}

/**
* storeKey
* @param id string
* @return string
**/
func (s *Crontab) storeKey(id string) string {
	return s.Tag + ":" + id
}

/**
* jobChannel: El canal en que se publica cada disparo del job tag.
* @param tag string
* @return string
**/
func (s *Crontab) jobChannel(tag string) string {
	return fmt.Sprintf("%s:job:%s", s.prefix, tag)
}

/**
* CronJob: Programa un job repetitivo; el id del job es tag. Registrar otra definición con el mismo tag la reemplaza;
* registrar la misma conserva el job y su estado. En cada disparo fn recibe params (en una sola réplica); repetitions
* 0 es sin límite. fn nil usa el handler de la instancia (WithHandler); sin ninguno, error.
* @param tag, ownerId, spec string, repetitions int, params et.Json, fn func(params et.Json) error, opts ...JobOption
* @return *Job, error
**/
func (s *Crontab) CronJob(tag, ownerId, spec string, repetitions int, params et.Json, fn func(params et.Json) error, opts ...JobOption) (*Job, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, errors.New(MSG_ERROR_TAG_REQUIRED)
	}

	spec, err := ParseSpec(spec)
	if err != nil {
		return nil, err
	}

	job := newJob(s, CRONJOB, tag, ownerId, params, repetitions, opts)
	job.Spec = spec
	return s.register(job, fn)
}

/**
* ScheduleJob: Programa un job de una sola vez en at; el id del job es tag, con las mismas reglas de reemplazo y de fn
* que CronJob. Una fecha pasada solo se acepta si ya existe el job con la misma definición (recargado del Store o
* difundido por la líder): se conserva con su estado; si no, error.
* @param tag, ownerId string, at time.Time, params et.Json, fn func(params et.Json) error, opts ...JobOption
* @return *Job, error
**/
func (s *Crontab) ScheduleJob(tag, ownerId string, at time.Time, params et.Json, fn func(params et.Json) error, opts ...JobOption) (*Job, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, errors.New(MSG_ERROR_TAG_REQUIRED)
	}

	job := newJob(s, SCHEDULEJOB, tag, ownerId, params, 0, opts)
	job.At = at
	if !at.After(timezone.Now()) && s.sameJob(job) == nil {
		return nil, fmt.Errorf(MSG_ERROR_SCHEDULE_PAST, at.Format(time.RFC3339))
	}
	return s.register(job, fn)
}

/**
* RemoveJob: Quita el job de tag en todas las réplicas, lo borra del Store y deja de atender sus disparos en esta.
* @param tag string
* @return error
**/
func (s *Crontab) RemoveJob(tag string) error {
	if err := s.unlisten(tag); err != nil {
		return err
	}
	if s.store != nil {
		if err := s.store.Delete(storeJobs, s.storeKey(tag)); err != nil {
			return err
		}
	}
	return s.publish(s.evRemove, et.Json{"id": tag})
}

/**
* StopJob: Pausa el job de tag en todas las réplicas.
* @param tag string
* @return error
**/
func (s *Crontab) StopJob(tag string) error {
	return s.publish(s.evStop, et.Json{"id": tag})
}

/**
* StartJob: Reanuda el job de tag (en pausa) en todas las réplicas.
* @param tag string
* @return error
**/
func (s *Crontab) StartJob(tag string) error {
	return s.publish(s.evStart, et.Json{"id": tag})
}

/**
* GetJob: El estado del job de tag según esta réplica (la líder lo difunde a las demás).
* @param tag string
* @return et.Json, bool
**/
func (s *Crontab) GetJob(tag string) (et.Json, bool) {
	job := s.job(tag)
	if job == nil {
		return nil, false
	}
	return job.ToJson(), true
}

/**
* Jobs: El estado de los jobs según esta réplica.
* @return []et.Json
**/
func (s *Crontab) Jobs() []et.Json {
	s.mu.Lock()
	jobs := s.jobList()
	s.mu.Unlock()

	result := make([]et.Json, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, job.ToJson())
	}
	return result
}

/**
* HttpRemoveJob: El id del job en la URL.
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Crontab) HttpRemoveJob(w http.ResponseWriter, r *http.Request) {
	s.httpSend(w, r, s.RemoveJob, MSG_SEND_JOB_REMOVED)
}

/**
* HttpStopJob
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Crontab) HttpStopJob(w http.ResponseWriter, r *http.Request) {
	s.httpSend(w, r, s.StopJob, MSG_SEND_JOB_STOPPED)
}

/**
* HttpStartJob
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Crontab) HttpStartJob(w http.ResponseWriter, r *http.Request) {
	s.httpSend(w, r, s.StartJob, MSG_SEND_JOB_STARTED)
}

/**
* httpSend: Toma el id de la URL, envía la orden y responde su resultado.
* @param w http.ResponseWriter, r *http.Request, send func(tag string) error, message string
**/
func (s *Crontab) httpSend(w http.ResponseWriter, r *http.Request, send func(tag string) error, message string) {
	id := strings.TrimSpace(request.URLParam(r, "id").Str())
	if id == "" {
		response.HTTPError(w, r, http.StatusBadRequest, MSG_ERROR_TAG_REQUIRED)
		return
	}
	if err := send(id); err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: et.Json{"message": message},
	})
}

/**
* job
* @param id string
* @return *Job (nil si esta réplica no lo conoce)
**/
func (s *Crontab) job(id string) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[id]
}

/**
* jobList: Los jobs de la instancia. Con mu tomado.
* @return []*Job
**/
func (s *Crontab) jobList() []*Job {
	result := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		result = append(result, job)
	}
	return result
}

/**
* register: Atiende los disparos del job con fn (nil: el handler de la instancia) en esta réplica. Si ya hay un job
* con la misma definición lo devuelve (con su estado); si no, lo pone en lugar del anterior, lo difunde y, si esta es
* la líder, lo programa.
* @param job *Job, fn func(params et.Json) error
* @return *Job, error
**/
func (s *Crontab) register(job *Job, fn func(params et.Json) error) (*Job, error) {
	if err := s.listen(job.Tag, fn); err != nil {
		return nil, err
	}

	if current := s.sameJob(job); current != nil {
		return current, nil
	}

	if !s.put(job) {
		return s.job(job.ID), nil
	}
	job.mu.Lock()
	s.send(s.evJob, job.toJson())
	job.mu.Unlock()
	job.lead()

	logs.Log(packageName, fmt.Sprintf(MSG_ADD_JOB, job.ID, job.Tag, s.jobChannel(job.Tag), job.Type, job.Spec))
	return job, nil
}

/**
* sameJob: El job registrado con la misma definición que job, sin cancelar.
* @param job *Job
* @return *Job (nil si no hay uno así)
**/
func (s *Crontab) sameJob(job *Job) *Job {
	current := s.job(job.ID)
	if current != nil && current.sameDef(job) && current.status() != Canceled {
		return current
	}
	return nil
}

/**
* put: Pone job en lugar del anterior con el mismo id si es más reciente (Rev mayor) y cancela el anterior.
* @param job *Job
* @return bool (false si ya había uno igual o más reciente)
**/
func (s *Crontab) put(job *Job) bool {
	s.mu.Lock()
	current := s.jobs[job.ID]
	if current != nil && current.Rev >= job.Rev {
		s.mu.Unlock()
		return false
	}
	s.jobs[job.ID] = job
	s.mu.Unlock()

	if current != nil {
		current.cancel()
	}
	return true
}

/**
* status
* @return JobStatus
**/
func (s *Job) status() JobStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Status
}

/**
* removeJob: Cancela y quita el job; si lo atendía el handler de la instancia, deja de atender su canal.
* @param id string
**/
func (s *Crontab) removeJob(id string) {
	s.mu.Lock()
	job, exists := s.jobs[id]
	delete(s.jobs, id)
	s.mu.Unlock()
	if !exists {
		return
	}
	if s.listensByHandler(job.Tag) {
		if err := s.unlisten(job.Tag); err != nil {
			logs.Error(err)
		}
	}

	job.cancel()
	logs.Log(packageName, fmt.Sprintf(MSG_REMOVE_JOB, id))
}
