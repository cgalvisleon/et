package tcp

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

/**
* Tipos de Message (spec §14.2).
**/
const (
	TypeBeat  = iota // 0: latido; sin valor
	TypeInt          // 1: int64
	TypeFloat        // 2: float64
	TypeText         // 3: string
	TypeJson         // 4: et.Json
)

/**
* Message: Mensaje tipado del Hub. From lo pone el Hub que envía (lo que traiga se
* reemplaza); Header son metadatos libres (nil viaja como {}).
**/
type Message struct {
	From    et.Json `json:"from"`
	Header  et.Json `json:"header"`
	Type    int     `json:"type"`
	Message any     `json:"message"`
}

/**
* wireMessage: Un Message en la conexión: el Data de una trama msgSend (o de un
* latido, sin seq).
**/
type wireMessage struct {
	Seq     uint64  `json:"seq,omitempty"`
	From    et.Json `json:"from"`
	Header  et.Json `json:"header"`
	Type    int     `json:"type"`
	Message any     `json:"message"`
}

/**
* errInvalidMessage: MSG_HUB_INVALID_MESSAGE
* @return error
**/
func errInvalidMessage() error {
	return errors.New(msg.MSG_HUB_INVALID_MESSAGE)
}

/**
* normalizeValue: Valida value contra tp y lo lleva al tipo que viaja (tabla §14.2).
* @param tp int, value any
* @return any, error
**/
func normalizeValue(tp int, value any) (any, error) {
	switch tp {
	case TypeBeat:
		return nil, nil
	case TypeInt:
		switch v := value.(type) {
		case int:
			return int64(v), nil
		case int8:
			return int64(v), nil
		case int16:
			return int64(v), nil
		case int32:
			return int64(v), nil
		case int64:
			return v, nil
		case uint8:
			return int64(v), nil
		case uint16:
			return int64(v), nil
		case uint32:
			return int64(v), nil
		}
	case TypeFloat:
		switch v := value.(type) {
		case float32:
			return float64(v), nil
		case float64:
			return v, nil
		}
	case TypeText:
		if v, ok := value.(string); ok {
			return v, nil
		}
	case TypeJson:
		switch v := value.(type) {
		case et.Json:
			return v, nil
		case map[string]any:
			return et.Json(v), nil
		}
	}
	return nil, errInvalidMessage()
}

/**
* encodeMessage: Valida m (sin latidos) y lo codifica con seq y from, sin modificar m.
* @param m *Message, seq uint64, from et.Json
* @return []byte, error
**/
func encodeMessage(m *Message, seq uint64, from et.Json) ([]byte, error) {
	if m == nil || m.Type <= TypeBeat || m.Type > TypeJson {
		return nil, errInvalidMessage()
	}

	value, err := normalizeValue(m.Type, m.Message)
	if err != nil {
		return nil, err
	}

	header := m.Header
	if header == nil {
		header = et.Json{}
	}

	return json.Marshal(wireMessage{Seq: seq, From: from, Header: header, Type: m.Type, Message: value})
}

/**
* encodeBeat: Codifica un latido (sin seq).
* @param from et.Json
* @return []byte
**/
func encodeBeat(from et.Json) []byte {
	data, _ := json.Marshal(wireMessage{From: from, Header: et.Json{}, Type: TypeBeat})
	return data
}

/**
* decodeWire: Lee Data con UseNumber. ok=false si no es JSON legible (sin seq).
* @param data []byte
* @return wireMessage, bool
**/
func decodeWire(data []byte) (wireMessage, bool) {
	var result wireMessage
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&result); err != nil {
		return wireMessage{}, false
	}
	return result, true
}

/**
* isBeat: Si w es un latido: type 0, sin seq y con el From del Hub que lo envía. Un
* JSON cualquiera también decodifica con type 0, así que from.id es lo que lo distingue.
* @param w wireMessage
* @return bool
**/
func isBeat(w wireMessage) bool {
	return w.Type == TypeBeat && w.Seq == 0 && w.From.Str("id") != ""
}

/**
* toMessage: Convierte un wireMessage recibido en Message según su tipo (§14.2).
* @param w wireMessage
* @return *Message, error
**/
func toMessage(w wireMessage) (*Message, error) {
	result := &Message{From: w.From, Header: w.Header, Type: w.Type}
	if result.From == nil {
		result.From = et.Json{}
	}
	if result.Header == nil {
		result.Header = et.Json{}
	}

	switch w.Type {
	case TypeBeat:
		return result, nil
	case TypeInt:
		if n, ok := w.Message.(json.Number); ok {
			if v, err := n.Int64(); err == nil {
				result.Message = v
				return result, nil
			}
		}
	case TypeFloat:
		if n, ok := w.Message.(json.Number); ok {
			if v, err := n.Float64(); err == nil {
				result.Message = v
				return result, nil
			}
		}
	case TypeText:
		if v, ok := w.Message.(string); ok {
			result.Message = v
			return result, nil
		}
	case TypeJson:
		if v, ok := w.Message.(map[string]any); ok {
			result.Message = et.Json(v)
			return result, nil
		}
	}
	return nil, errInvalidMessage()
}
