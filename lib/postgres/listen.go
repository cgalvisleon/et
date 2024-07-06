package lib

import (
	"log"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/lib/pq"
)

var listened bool

func defineListen(connStr string, channels []string) {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	listener := pq.NewListener(connStr, 10*time.Second, time.Minute, reportProblem)
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
				handleListen(result)
			}
		case <-time.After(90 * time.Second):
			go listener.Ping()
		}
	}
}

func handleListen(res et.Json) {
	logs.Debug("lintened: ", res.ToString())
}
