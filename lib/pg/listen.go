package lib

import (
	"log"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/linq"
	"github.com/cgalvisleon/et/logs"
	"github.com/lib/pq"
)

/**
* SetListen set the channels to listen
* @param channels []string
* @param listen linq.HandlerListend
**/
func (p *Postgres) SetListen(channels []string, listen linq.HandlerListend) {
	go p.defineListend(channels, listen)
}

var listened bool

/**
* defineListend define the channels to listen
* @param channels []string
* @param lited linq.HandlerListend
**/
func (p *Postgres) defineListend(channels []string, lited linq.HandlerListend) {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	listener := pq.NewListener(p.chain, 10*time.Second, time.Minute, reportProblem)
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
					logs.Alertm("defineListend: Not conver to Json")
				}

				result.Set("channel", notification.Channel)
				lited(result)
			}
		case <-time.After(90 * time.Second):
			go listener.Ping()
		}
	}
}
