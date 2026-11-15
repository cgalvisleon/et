package crontab

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

/**
* subscribe: Los mensajes del grupo de réplicas: jobs (definición y estado), órdenes (remove, stop, start),
* respuestas de los workers (done) y anuncios (hello, bye).
* @return error
**/
func (s *Crontab) subscribe() error {
	handlers := []struct {
		channel string
		fn      func(et.Json)
	}{
		{s.evJob, s.eventJob},
		{s.evRemove, s.eventRemove},
		{s.evStop, s.eventStop},
		{s.evStart, s.eventStart},
		{s.evDone, s.eventDone},
		{s.evHello, s.eventHello},
		{s.evBye, s.eventBye},
	}
	for _, h := range handlers {
		if err := s.bus.Subscribe(h.channel, h.fn); err != nil {
			return err
		}
		s.mu.Lock()
		s.channels = append(s.channels, h.channel)
		s.mu.Unlock()
	}
	return nil
}

/**
* unsubscribe: Cancela las suscripciones de la instancia, incluidos los canales de sus jobs.
* @return error
**/
func (s *Crontab) unsubscribe() error {
	s.mu.Lock()
	channels := s.channels
	for channel := range s.listening {
		channels = append(channels, channel)
	}
	s.channels = nil
	s.listening = make(map[string]bool)
	s.mu.Unlock()

	var result error
	for _, channel := range channels {
		if err := s.bus.Unsubscribe(channel); err != nil && result == nil {
			result = err
		}
	}
	return result
}

/**
* listen: Atiende los disparos del job tag con fn en la cola QUEUE_JOBS (una sola réplica por disparo) y responde
* el resultado en el canal done. Reemplaza la suscripción anterior del mismo canal.
* @param tag string, fn func(params et.Json) error
* @return error
**/
func (s *Crontab) listen(tag string, fn func(params et.Json) error) error {
	channel := s.jobChannel(tag)
	// una suscripción por canal: la anterior se cancela antes, o quedaría viva y repetiría cada disparo
	if err := s.bus.Unsubscribe(channel); err != nil {
		return err
	}

	err := s.bus.Queue(channel, QUEUE_JOBS, func(data et.Json) {
		err := runSafe(fn, data.Json("params"))
		done := et.Json{"id": tag, "run_id": data.Str("run_id"), "attempt": data.Int("attempt"), "host_name": s.HostName}
		if err != nil {
			done["error"] = err.Error()
			logs.Log(packageName, fmt.Sprintf(MSG_ERROR_EXECUTING_JOB, s.Tag, tag, err))
		}
		if err := s.publish(s.evDone, done); err != nil {
			logs.Error(err)
		}
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.listening[channel] = true
	s.mu.Unlock()
	return nil
}

/**
* unlisten: Deja de atender los disparos del job tag en esta réplica.
* @param tag string
* @return error
**/
func (s *Crontab) unlisten(tag string) error {
	channel := s.jobChannel(tag)
	s.mu.Lock()
	delete(s.listening, channel)
	s.mu.Unlock()
	return s.bus.Unsubscribe(channel)
}

/**
* runSafe: Ejecuta fn y convierte un panic en error, para que la corrida termine como failed y no por timeout.
* @param fn func(params et.Json) error, params et.Json
* @return error
**/
func runSafe(fn func(params et.Json) error, params et.Json) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf(MSG_ERROR_JOB_PANIC, r)
		}
	}()
	return fn(params)
}

/**
* eventJob: Un job difundido por otra réplica. Una revisión más reciente reemplaza a la copia (y la líder la
* programa); la misma revisión trae el estado que difundió la líder. Lo propio y lo de una revisión vieja se ignora.
* @param data et.Json
**/
func (s *Crontab) eventJob(data et.Json) {
	if data.Str("from") == s.id {
		return
	}
	incoming, err := jobOf(s, data)
	if err != nil {
		logs.Error(err)
		return
	}

	current := s.job(incoming.ID)
	if current != nil && current.Rev == incoming.Rev {
		if s.isLeader() {
			return
		}
		current.mu.Lock()
		current.applyState(incoming)
		current.mu.Unlock()
		return
	}
	if s.put(incoming) {
		incoming.lead()
	}
}

/**
* eventRemove
* @param data et.Json
**/
func (s *Crontab) eventRemove(data et.Json) {
	s.removeJob(data.Str("id"))
}

/**
* eventStop
* @param data et.Json
**/
func (s *Crontab) eventStop(data et.Json) {
	if job := s.job(data.Str("id")); job != nil {
		job.stop()
	}
}

/**
* eventStart
* @param data et.Json
**/
func (s *Crontab) eventStart(data et.Json) {
	job := s.job(data.Str("id"))
	if job == nil {
		return
	}
	if err := job.start(); err != nil && s.isLeader() {
		logs.Errorf(MSG_ERROR_STARTING_JOB, err.Error())
	}
}

/**
* eventDone: La respuesta de un worker: cierra el intento en la líder.
* @param data et.Json
**/
func (s *Crontab) eventDone(data et.Json) {
	job := s.job(data.Str("id"))
	if job == nil {
		return
	}
	status := RunSucceeded
	errText := data.Str("error")
	if errText != "" {
		status = RunFailed
	}
	job.endAttempt(data.Str("run_id"), data.Int("attempt"), status, data.Str("host_name"), errText)
}
