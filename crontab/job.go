package crontab

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/robfig/cron/v3"
)

// JobStatus: el ciclo de vida de un job, aparte del resultado de cada corrida (Run).
type JobStatus string

const (
	Pending   JobStatus = "pending"   // registrado, todavía sin programar
	Scheduled JobStatus = "scheduled" // programado, esperando su próxima hora
	Running   JobStatus = "running"   // disparado, esperando la respuesta del worker
	Paused    JobStatus = "paused"    // detenido con stop; start lo reanuda
	Finished  JobStatus = "finished"  // terminó y su última corrida salió bien
	Missed    JobStatus = "missed"    // schedule cuya hora pasó sin correr
	Failed    JobStatus = "failed"    // no se pudo programar, o terminó con su última corrida fallida
	Canceled  JobStatus = "canceled"  // quitado o reemplazado
)

/**
* IsFinal: Un job en un estado final no vuelve a correr; para repetirlo se quita y se registra de nuevo.
* @return bool
**/
func (s JobStatus) IsFinal() bool {
	return s == Finished || s == Missed || s == Failed || s == Canceled
}

// RunStatus: el resultado de una corrida.
type RunStatus string

const (
	RunDispatched RunStatus = "dispatched" // publicada, sin respuesta todavía
	RunRetrying   RunStatus = "retrying"   // falló y espera RetryDelay para el siguiente intento
	RunSucceeded  RunStatus = "succeeded"  // el worker respondió sin error
	RunFailed     RunStatus = "failed"     // falló en todos sus intentos (error del worker o al publicar)
	RunTimeout    RunStatus = "timeout"    // su último intento no tuvo respuesta antes del timeout
)

type TypeJob string

const (
	CRONJOB     TypeJob = "cronJob"
	SCHEDULEJOB TypeJob = "scheduleJob"
)

// errMissed: la hora de un schedule ya pasó.
var errMissed = errors.New("schedule time has passed")

// Run: una corrida de un job; Attempt cuenta sus intentos (1 + reintentos).
type Run struct {
	ID        string    `json:"id"`
	Status    RunStatus `json:"status"`
	Attempt   int       `json:"attempt"`
	HostName  string    `json:"host_name"` // la instancia que la ejecutó (o la que falló al publicarla)
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Error     string    `json:"error"`
}

/**
* toJson
* @return et.Json
**/
func (s *Run) toJson() et.Json {
	result := et.Json{
		"id":         s.ID,
		"status":     s.Status,
		"attempt":    s.Attempt,
		"host_name":  s.HostName,
		"started_at": s.StartedAt,
		"error":      s.Error,
	}
	if !s.EndedAt.IsZero() {
		result["ended_at"] = s.EndedAt
	}
	return result
}

// JobOption: ajustes opcionales de un job (CronJob y ScheduleJob los aceptan al final).
type JobOption func(*Job)

/**
* WithRetries: Reintenta una corrida fallida (error, panic o timeout) hasta n veces, esperando delay entre intentos.
* @param n int, delay time.Duration
* @return JobOption
**/
func WithRetries(n int, delay time.Duration) JobOption {
	return func(j *Job) {
		j.Retries = max(n, 0)
		j.RetryDelay = max(delay, 0)
	}
}

/**
* WithTimeout: Cuánto se espera la respuesta del worker en este job (si no, el de la instancia).
* @param d time.Duration
* @return JobOption
**/
func WithTimeout(d time.Duration) JobOption {
	return func(j *Job) {
		j.Timeout = max(d, 0)
	}
}

/**
* WithPaused: Registra el job ya en pausa (sin una ventana programado antes de un StopJob); StartJob lo reanuda. Solo
* aplica a una definición nueva: si ya hay un job con la misma definición se conserva con su estado.
* @return JobOption
**/
func WithPaused() JobOption {
	return func(j *Job) {
		j.Status = Paused
	}
}

/**
* Job: Un job de una instancia de Crontab. La definición (ID, Rev, Type, Tag, OwnerId, Spec, At, Params, Repetitions,
* Retries, RetryDelay, Timeout) no cambia después de crearlo; el resto se lee y escribe bajo mu, incluido save().
* Cada réplica tiene su copia: la líder la programa, la guarda y la difunde; las demás aplican lo que difunde.
**/
type Job struct {
	Crontab     string        `json:"crontab"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	OwnerId     string        `json:"owner_id"`
	ID          string        `json:"id"`
	Rev         string        `json:"rev"`
	Type        TypeJob       `json:"type"`
	Tag         string        `json:"tag"`
	Spec        string        `json:"spec"` // la expresión cron (vacía en un schedule)
	At          time.Time     `json:"at"`   // la hora de un schedule (cero en un cron)
	Params      et.Json       `json:"params"`
	Repetitions int           `json:"repetitions"`
	Retries     int           `json:"retries"`
	RetryDelay  time.Duration `json:"retry_delay"`
	Timeout     time.Duration `json:"timeout"`
	Status      JobStatus     `json:"status"`
	HostName    string        `json:"host_name"` // la instancia que lo registró
	Runs        int           `json:"runs"`      // corridas lanzadas
	Skipped     int           `json:"skipped"`   // disparos omitidos porque la corrida anterior seguía en curso
	Duration    time.Duration `json:"duration"`
	LastRun     *Run          `json:"last_run"`
	AuditLog    []et.Json     `json:"audit_log"`
	idx         cron.EntryID
	shot        *time.Timer
	runTimer    *time.Timer
	retryTimer  *time.Timer
	crontab     *Crontab
	mu          sync.Mutex
}

/**
* newJob: El id del job es su tag (registrar otra definición con el mismo tag la reemplaza); rev distingue cada
* registro y ordena los reemplazos (UUID v7: la mayor es la más reciente).
* @param c *Crontab, tp TypeJob, tag, ownerId string, params et.Json, repetitions int, opts []JobOption
* @return *Job
**/
func newJob(c *Crontab, tp TypeJob, tag, ownerId string, params et.Json, repetitions int, opts []JobOption) *Job {
	now := timezone.Now()
	result := &Job{
		Crontab:     c.Tag,
		CreatedAt:   now,
		UpdatedAt:   now,
		OwnerId:     ownerId,
		ID:          tag,
		Rev:         reg.UUID(),
		Type:        tp,
		Tag:         tag,
		Params:      params,
		Status:      Pending,
		HostName:    c.HostName,
		Repetitions: repetitions,
		crontab:     c,
	}
	for _, opt := range opts {
		opt(result)
	}
	if result.Status == Paused {
		result.addAuditLog(string(Paused))
	}
	return result
}

/**
* jobOf: Rehace un job desde su JSON (el del Store o el que difunde la líder).
* @param c *Crontab, data et.Json
* @return *Job, error
**/
func jobOf(c *Crontab, data et.Json) (*Job, error) {
	bt, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	result := &Job{}
	if err := json.Unmarshal(bt, result); err != nil {
		return nil, err
	}
	if result.ID == "" || result.Rev == "" {
		return nil, errors.New(MSG_ERROR_JOB_INVALID)
	}
	result.crontab = c
	return result, nil
}

/**
* sameDef: Las dos definiciones programan lo mismo (registrar una igual conserva el job y su estado).
* @param other *Job
* @return bool
**/
func (s *Job) sameDef(other *Job) bool {
	return s.Type == other.Type && s.Tag == other.Tag && s.OwnerId == other.OwnerId && s.Spec == other.Spec &&
		s.At.Equal(other.At) && s.Params.ToString() == other.Params.ToString() && s.Repetitions == other.Repetitions &&
		s.Retries == other.Retries && s.RetryDelay == other.RetryDelay && s.Timeout == other.Timeout
}

/**
* ToJson: Una copia del job.
* @return et.Json
**/
func (s *Job) ToJson() et.Json {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.toJson()
}

/**
* toJson: Como ToJson, con mu ya tomado.
* @return et.Json
**/
func (s *Job) toJson() et.Json {
	var lastRun et.Json
	if s.LastRun != nil {
		lastRun = s.LastRun.toJson()
	}
	result := et.Json{
		"crontab":     s.Crontab,
		"created_at":  s.CreatedAt,
		"updated_at":  s.UpdatedAt,
		"owner_id":    s.OwnerId,
		"id":          s.ID,
		"rev":         s.Rev,
		"type":        s.Type,
		"tag":         s.Tag,
		"spec":        s.Spec,
		"params":      s.Params,
		"repetitions": s.Repetitions,
		"retries":     s.Retries,
		"retry_delay": s.RetryDelay,
		"timeout":     s.Timeout,
		"status":      s.Status,
		"host_name":   s.HostName,
		"runs":        s.Runs,
		"skipped":     s.Skipped,
		"duration":    s.Duration,
		"last_run":    lastRun,
		"audit_log":   s.AuditLog,
	}
	if !s.At.IsZero() {
		result["at"] = s.At
	}
	return result
}

/**
* ToString
* @return string
**/
func (s *Job) ToString() string {
	return s.ToJson().ToString()
}

/**
* applyState: Copia el estado que difundió la líder (no la definición). Con mu tomado.
* @param from *Job
**/
func (s *Job) applyState(from *Job) {
	s.Status = from.Status
	s.Runs = from.Runs
	s.Skipped = from.Skipped
	s.Duration = from.Duration
	s.LastRun = from.LastRun
	s.AuditLog = from.AuditLog
	s.UpdatedAt = from.UpdatedAt
}

/**
* addAuditLog: Con mu tomado.
* @param action string
**/
func (s *Job) addAuditLog(action string) {
	now := timezone.Now()
	s.UpdatedAt = now
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    s.crontab.HostName,
		"action":     action,
	})
	maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
}

/**
* save: Guarda el job en el Store y difunde su estado a las réplicas; solo la líder. Con mu tomado.
**/
func (s *Job) save() {
	if !s.crontab.isLeader() {
		return
	}
	s.broadcast()
	if err := s.crontab.saveJob(s); err != nil {
		logs.Error(err)
	}
}

/**
* broadcast: Difunde el estado a las réplicas sin guardarlo (cambios de cada disparo). Con mu tomado.
**/
func (s *Job) broadcast() {
	if s.crontab.isLeader() {
		s.crontab.send(s.crontab.evJob, s.toJson())
	}
}

/**
* setStatus: Cambia el estado. Los pasos scheduled ⇄ running de cada disparo no van a la auditoría ni al log, para
* no escribir dos entradas por corrida. Con mu tomado.
* @param status JobStatus
**/
func (s *Job) setStatus(status JobStatus) {
	if s.Status == status {
		return
	}
	perRun := (s.Status == Scheduled && status == Running) || (s.Status == Running && status == Scheduled)
	s.Status = status
	s.UpdatedAt = timezone.Now()
	if !perRun {
		s.addAuditLog(string(status))
		logs.Logf(packageName, MSG_JOB_STATUS, s.Tag, status, s.crontab.HostName, s.Runs, s.Repetitions)
	}
}

/**
* inFlight: Hay una corrida en curso (publicada o esperando reintento). Con mu tomado.
* @return bool
**/
func (s *Job) inFlight() bool {
	return s.LastRun != nil && (s.LastRun.Status == RunDispatched || s.LastRun.Status == RunRetrying)
}

/**
* isLastRun: Ya se lanzaron todas las corridas: un schedule corre una vez, un cron hasta repetitions (0 = sin límite).
* Con mu tomado.
* @return bool
**/
func (s *Job) isLastRun() bool {
	if s.Type != CRONJOB {
		return s.Runs >= 1
	}
	return s.Repetitions != 0 && s.Runs >= s.Repetitions
}

/**
* timeout: El de este job o el de la instancia.
* @return time.Duration
**/
func (s *Job) timeout() time.Duration {
	if s.Timeout > 0 {
		return s.Timeout
	}
	return s.crontab.runTimeout()
}

/**
* arm: Programa los disparos que le quedan al job. Con catchUp, un schedule atrasado menos de CRONTAB_CATCH_UP corre
* ya (la hora pasó sin líder); si no, errMissed. Con mu tomado.
* @param catchUp bool
* @return error
**/
func (s *Job) arm(catchUp bool) error {
	s.disarm()
	if s.isLastRun() {
		return nil
	}

	if s.Type == CRONJOB {
		idx, err := s.crontab.cron.AddFunc(s.Spec, s.trigger)
		if err != nil {
			return err
		}
		s.idx = idx
		return nil
	}

	wait := time.Until(s.At)
	if wait <= 0 && !(catchUp && -wait <= s.crontab.catchUp) {
		return errMissed
	}
	s.Duration = max(wait, 0)
	s.shot = time.AfterFunc(max(wait, 0), s.trigger)
	return nil
}

/**
* disarm: Quita los disparos pendientes. Con mu tomado.
**/
func (s *Job) disarm() {
	if s.idx != 0 {
		s.crontab.cron.Remove(s.idx)
		s.idx = 0
	}
	if s.shot != nil {
		s.shot.Stop()
		s.shot = nil
	}
}

/**
* stopTimers: Quita los disparos y los timers de la corrida en curso. Con mu tomado.
**/
func (s *Job) stopTimers() {
	s.disarm()
	for _, t := range []*time.Timer{s.runTimer, s.retryTimer} {
		if t != nil {
			t.Stop()
		}
	}
	s.runTimer, s.retryTimer = nil, nil
}

/**
* failArm: Lleva al job a missed o failed según el error de arm. Con mu tomado.
* @param err error
**/
func (s *Job) failArm(err error) {
	if errors.Is(err, errMissed) {
		s.setStatus(Missed)
	} else {
		s.setStatus(Failed)
		logs.Error(err)
	}
}

/**
* lead: Esta réplica pasó a líder (o el job es nuevo y ella lo es): programa lo que le toca al job según su estado.
* Una corrida en curso de la líder anterior se sigue esperando (su respuesta puede llegar) hasta el timeout. Uno en
* pausa no se programa, pero se guarda (un job registrado en pausa llega así al Store).
**/
func (s *Job) lead() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.crontab.isLeader() || s.Status.IsFinal() {
		return
	}
	if s.Status == Paused {
		s.save()
		return
	}

	catchUp := s.Status != Pending
	if err := s.arm(catchUp); err != nil {
		s.failArm(err)
		s.save()
		return
	}
	if s.inFlight() {
		s.setStatus(Running)
		s.waitRun(s.LastRun)
	} else {
		s.setStatus(Scheduled)
	}
	s.save()
}

/**
* follow: Esta réplica dejó de ser líder: deja de programar el job y conserva su estado.
**/
func (s *Job) follow() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopTimers()
}

/**
* start: Reanuda un job en pausa (en todas las réplicas; la líder lo programa). Un job en estado final no se
* reanuda; uno que no estaba en pausa no cambia.
* @return error
**/
func (s *Job) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status.IsFinal() {
		return fmt.Errorf(MSG_JOB_ENDED, s.Tag, s.Status)
	}
	if s.Status != Paused {
		return nil
	}

	if s.crontab.isLeader() {
		if err := s.arm(false); err != nil {
			s.failArm(err)
			s.save()
			return err
		}
	}
	if s.inFlight() {
		s.setStatus(Running)
	} else {
		s.setStatus(Scheduled)
	}
	s.save()
	return nil
}

/**
* stop: Pausa el job en todas las réplicas; una corrida en curso termina y queda registrada.
**/
func (s *Job) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status.IsFinal() {
		return
	}
	s.disarm()
	s.setStatus(Paused)
	s.save()
}

/**
* cancel: El job se quitó o se reemplazó: deja de programarse y no se vuelve a guardar.
**/
func (s *Job) cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopTimers()
	s.setStatus(Canceled)
}

/**
* trigger: Lanza una corrida (solo la líder). Si la anterior sigue en curso, el disparo se omite (Skipped).
**/
func (s *Job) trigger() {
	s.mu.Lock()
	if !s.crontab.isLeader() || s.Status != Scheduled && s.Status != Running {
		s.mu.Unlock()
		return
	}
	if s.inFlight() {
		s.Skipped++
		s.broadcast()
		s.mu.Unlock()
		return
	}

	s.Runs++
	run := &Run{ID: reg.ULID(), Status: RunDispatched, Attempt: 1, StartedAt: timezone.Now()}
	s.LastRun = run
	if s.isLastRun() {
		s.disarm()
	}
	s.setStatus(Running)
	s.broadcast()
	s.mu.Unlock()

	s.dispatch(run.ID)
}

/**
* waitRun: Arma el timeout del intento en curso de run. Con mu tomado.
* @param run *Run
**/
func (s *Job) waitRun(run *Run) {
	if s.runTimer != nil {
		s.runTimer.Stop()
	}
	id, attempt, timeout := run.ID, run.Attempt, s.timeout()
	s.runTimer = time.AfterFunc(timeout, func() {
		s.endAttempt(id, attempt, RunTimeout, "", fmt.Sprintf(MSG_ERROR_RUN_TIMEOUT, timeout))
	})
}

/**
* dispatch: Publica el intento en curso de la corrida runID en el canal del job y espera la respuesta del worker.
* Se publica sin mu: la respuesta (endAttempt) puede llegar antes de que Publish vuelva.
* @param runID string
**/
func (s *Job) dispatch(runID string) {
	s.mu.Lock()
	run := s.LastRun
	if run == nil || run.ID != runID || run.Status != RunDispatched {
		s.mu.Unlock()
		return
	}
	s.waitRun(run)
	attempt := run.Attempt
	data := et.Json{"id": s.Tag, "run_id": run.ID, "attempt": attempt, "params": s.Params}
	channel := s.crontab.jobChannel(s.Tag)
	s.mu.Unlock()

	if err := s.crontab.publish(channel, data); err != nil {
		s.endAttempt(runID, attempt, RunFailed, s.crontab.HostName, err.Error())
	}
}

/**
* retry: Lanza el siguiente intento de la corrida runID.
* @param runID string
**/
func (s *Job) retry(runID string) {
	s.mu.Lock()
	run := s.LastRun
	if !s.crontab.isLeader() || run == nil || run.ID != runID || run.Status != RunRetrying {
		s.mu.Unlock()
		return
	}
	s.retryTimer = nil
	run.Attempt++
	run.Status = RunDispatched
	s.broadcast()
	s.mu.Unlock()

	s.dispatch(runID)
}

/**
* endAttempt: Cierra el intento attempt de la corrida runID. Si falló y le quedan reintentos, espera RetryDelay; si
* no, cierra la corrida y mueve el job: al estado final si era la última, o de vuelta a scheduled (paused se respeta).
* Una respuesta tardía, repetida o de otro intento se ignora.
* @param runID string, attempt int, status RunStatus, hostName, errText string
**/
func (s *Job) endAttempt(runID string, attempt int, status RunStatus, hostName, errText string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	run := s.LastRun
	if !s.crontab.isLeader() || run == nil || run.ID != runID || run.Attempt != attempt || run.Status != RunDispatched {
		return
	}
	if s.runTimer != nil {
		s.runTimer.Stop()
		s.runTimer = nil
	}
	run.HostName, run.Error = hostName, errText

	if status != RunSucceeded && run.Attempt <= s.Retries {
		run.Status = RunRetrying
		s.retryTimer = time.AfterFunc(s.RetryDelay, func() { s.retry(runID) })
		s.broadcast()
		return
	}

	run.Status, run.EndedAt = status, timezone.Now()
	switch {
	case s.Status.IsFinal():
	case s.isLastRun() && status == RunSucceeded:
		s.setStatus(Finished)
	case s.isLastRun():
		s.setStatus(Failed)
	case s.Status == Paused:
	default:
		s.setStatus(Scheduled)
	}
	s.save()
}
