package lib

import (
	"fmt"
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
					logs.Alertm("defineLinten: Not conver to Json")
				}

				result.Set("channel", notification.Channel)
				handleListen(result)
			}
		case <-time.After(90 * time.Second):
			go func() {
				err := listener.Ping()
				if err != nil {
					log.Println(err)
				}
			}()
			fmt.Println("No notifications received for 90 seconds, checking connection...")
		}
	}
}

func handleListen(res et.Json) {
	logs.Debug("lintened: ", res.ToString())
}
