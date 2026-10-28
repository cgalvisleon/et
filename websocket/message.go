package websocket

import (
	"encoding/json"
	"time"
)

const (
	sendBuffer     = 256              // mensajes en espera de envío por cliente
	writeWait      = 10 * time.Second // plazo de cada escritura
	pongWait       = 60 * time.Second // sin recibir nada en este plazo, se desconecta
	pingPeriod     = 54 * time.Second // cada cuánto se envía ping (menor que pongWait)
	maxMessageSize = 1 << 20          // 1 MiB por mensaje del cliente
	maxCloseReason = 123              // bytes de texto que caben en un frame de cierre
)

// Tipos de mensaje del protocolo.
const (
	typeSubscribe   = "subscribe"
	typeQueue       = "queue"
	typeUnsubscribe = "unsubscribe"
	typePublish     = "publish"
	typeAck         = "ack"
	typeError       = "error"
	typeMessage     = "message"
	typeDirect      = "direct"
	typeConnected   = "connected" // conexión aceptada; id es el del cliente
)

/**
* SubscriptionType: Tipo de suscripción a un canal: topic (todos reciben) o queue
* (un miembro del grupo por mensaje, por turnos).
**/
type SubscriptionType string

const (
	TypeTopic SubscriptionType = "topic" // recibe todos los mensajes del canal
	TypeQueue SubscriptionType = "queue" // el grupo recibe cada mensaje una vez, por turnos
)

/**
* Subscription: Suscripción de un cliente a un canal.
**/
type Subscription struct {
	Channel string           `json:"channel"`
	Queue   string           `json:"queue,omitempty"`
	Type    SubscriptionType `json:"type"`
}

/**
* inbound: Mensaje del cliente al servidor.
**/
type inbound struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Channel string          `json:"channel,omitempty"`
	Queue   string          `json:"queue,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

/**
* reply: Respuesta (ack o error) a un mensaje del cliente.
**/
type reply struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	Message   string `json:"message,omitempty"`
	Delivered *int   `json:"delivered,omitempty"`
}

/**
* envelope: Mensaje publicado en un canal, tal como lo recibe cada suscriptor.
**/
type envelope struct {
	Type    string          `json:"type"`
	Channel string          `json:"channel"`
	Queue   string          `json:"queue,omitempty"`
	From    string          `json:"from"`
	Data    json.RawMessage `json:"data"`
}

/**
* direct: Mensaje enviado a un solo cliente con Conn.Send.
**/
type direct struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

/**
* toRaw: Codifica data en JSON; un json.RawMessage pasa tal cual.
* @param data any
* @return json.RawMessage, error
**/
func toRaw(data any) (json.RawMessage, error) {
	if raw, ok := data.(json.RawMessage); ok {
		return raw, nil
	}
	return json.Marshal(data)
}
