package crontab

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
)

/**
* bus: Los mensajes entre réplicas. En producción es NATS (natsBus); las pruebas usan uno en memoria.
* Unsubscribe cancela la suscripción de esta instancia al canal.
**/
type bus interface {
	Publish(channel string, data et.Json) error
	Subscribe(channel string, fn func(data et.Json)) error
	Queue(channel, queue string, fn func(data et.Json)) error
	Unsubscribe(channel string) error
}

// natsBus: el bus sobre el paquete event (una suscripción por canal y proceso).
type natsBus struct{}

/**
* Publish
* @param channel string, data et.Json
* @return error
**/
func (natsBus) Publish(channel string, data et.Json) error {
	return event.Publish(channel, data)
}

/**
* Subscribe
* @param channel string, fn func(data et.Json)
* @return error
**/
func (natsBus) Subscribe(channel string, fn func(data et.Json)) error {
	return event.Subscribe(channel, func(msg event.Message) { fn(msg.Data) })
}

/**
* Queue
* @param channel, queue string, fn func(data et.Json)
* @return error
**/
func (natsBus) Queue(channel, queue string, fn func(data et.Json)) error {
	return event.Queue(channel, queue, func(msg event.Message) { fn(msg.Data) })
}

/**
* Unsubscribe
* @param channel string
* @return error
**/
func (natsBus) Unsubscribe(channel string) error {
	return event.Unsubscribe(channel)
}
