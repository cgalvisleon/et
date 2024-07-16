package lib

import (
	"log"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/lib/pq"
)

var listened bool

func (p *Postgres) defineListend(channels []string, lited linq.HandleListen) {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	listener := pq.NewListener(p.connStr, 10*time.Second, time.Minute, reportProblem)
	for _, channel := range channels {
		err := listener.Listen(channel)
		if err != nil {
			log.Fatal(err)
		}
	}

	for {
		select {
		case notification := <-listener.Notify:
			if notification != nil {
				result, err := et.ToJson(notification.Extra)
				if err != nil {
					logs.Alertm("defineListen: Not conver to Json")
				}

				result.Set("channel", notification.Channel)
				lited(result)
			}
		case <-time.After(90 * time.Second):
			go listener.Ping()
		}
	}
}

func (p *Postgres) SetListen(listen linq.HandleListen) {
	go p.defineListend([]string{"sync", "recycling"}, listen)
}
