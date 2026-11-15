package crontab

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

/**
* Líder entre réplicas: las instancias con el mismo tag se anuncian cada heartbeat (hello) y la más antigua viva
* (el menor id, que empieza por su hora de arranque) programa todos los jobs. Una réplica que deja de anunciarse por
* 3 heartbeats, o que se despide (bye) al cerrarse, sale de la lista. Una réplica nueva espera 1,5 heartbeats antes de
* poder liderar, para conocer a las que ya estaban. Si la red se parte, cada lado elige su líder y un job puede
* dispararse en los dos hasta que se una de nuevo.
**/

/**
* heartbeatLoop: Se anuncia y reevalúa la líder cada heartbeat, hasta Close.
**/
func (s *Crontab) heartbeatLoop() {
	ticker := time.NewTicker(s.heartbeat)
	defer ticker.Stop()
	s.hello()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.hello()
			s.elect()
		}
	}
}

/**
* hello: Anuncia esta réplica.
**/
func (s *Crontab) hello() {
	if err := s.publish(s.evHello, et.Json{"id": s.id}); err != nil {
		logs.Error(err)
	}
}

/**
* eventHello: Otra réplica sigue viva. Si es nueva y esta es la líder, le envía todos los jobs.
* @param data et.Json
**/
func (s *Crontab) eventHello(data et.Json) {
	id := data.Str("id")
	if id == "" || id == s.id {
		return
	}
	s.mu.Lock()
	_, known := s.members[id]
	s.members[id] = time.Now()
	leader := s.leader
	s.mu.Unlock()

	if !known {
		s.elect()
		if leader {
			s.syncJobs()
		}
	}
}

/**
* eventBye: Otra réplica se cerró: sale de la lista ya, sin esperar a que venza.
* @param data et.Json
**/
func (s *Crontab) eventBye(data et.Json) {
	s.mu.Lock()
	delete(s.members, data.Str("id"))
	s.mu.Unlock()
	s.elect()
}

/**
* elect: Quita las réplicas vencidas y decide si esta es la líder; si cambia, programa o suelta todos los jobs.
* electMu serializa los cambios para que lead/follow no se crucen.
**/
func (s *Crontab) elect() {
	s.electMu.Lock()
	defer s.electMu.Unlock()

	now := time.Now()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	leader := now.Sub(s.startedAt) >= s.heartbeat*3/2
	for id, seen := range s.members {
		if now.Sub(seen) > 3*s.heartbeat {
			delete(s.members, id)
			continue
		}
		if id < s.id {
			leader = false
		}
	}
	changed := leader != s.leader
	s.leader = leader
	jobs := s.jobList()
	s.mu.Unlock()

	if !changed {
		return
	}
	logs.Logf(packageName, MSG_LEADER, s.Tag, s.id, leader)
	for _, job := range jobs {
		if leader {
			job.lead()
		} else {
			job.follow()
		}
	}
}

/**
* isLeader
* @return bool
**/
func (s *Crontab) isLeader() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.leader
}

/**
* IsLeader: Esta réplica es la que programa los jobs de su tag.
* @return bool
**/
func (s *Crontab) IsLeader() bool {
	return s.isLeader()
}

/**
* syncJobs: La líder difunde todos sus jobs (a una réplica que acaba de llegar).
**/
func (s *Crontab) syncJobs() {
	s.mu.Lock()
	jobs := s.jobList()
	s.mu.Unlock()
	for _, job := range jobs {
		job.mu.Lock()
		job.broadcast()
		job.mu.Unlock()
	}
}
