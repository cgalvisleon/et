package wsp

import (
	"github.com/cgalvisleon/et/et"
)

type Button struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Header struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func (s *Header) body() et.Json {
	return et.Json{
		"type": s.Type,
		"text": s.Content,
	}
}

type Footer struct {
	Text string `json:"text"`
}

type Row struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Section struct {
	Title string `json:"title"`
	Rows  []Row  `json:"rows"`
}

type Message struct {
	to       string    `json:"-"`
	kind     string    `json:"-"`
	Text     string    `json:"text"`
	Buttons  []Button  `json:"buttons"`
	Header   Header    `json:"header"`
	Footer   Footer    `json:"footer"`
	Button   string    `json:"button"`
	Sections []Section `json:"sections"`
}

/**
* setTo
* @param to string
* @return
**/
func (m *Message) setTo(to string) {
	m.to = to
}

/**
* btns
* @return et.Json
**/
func (m *Message) btns() []et.Json {
	result := []et.Json{}
	for _, btn := range m.Buttons {
		result = append(result, et.Json{
			"type": "reply",
			"reply": et.Json{
				"id":    btn.ID,
				"title": btn.Text,
			},
		})
	}
	return result
}

func (m *Message) sections() []et.Json {
	result := []et.Json{}
	for _, section := range m.Sections {
		result = append(result, et.Json{
			"title": section.Title,
			"rows":  section.Rows,
		})
	}
	return result
}

/**
* body
* @return et.Json
**/
func (s *Message) body() et.Json {
	switch s.kind {
	case "action":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"type":              "interactive",
			"to":                s.to,
			"interactive": et.Json{
				"type": "button",
				"body": et.Json{
					"text": s.Text,
				},
				"action": et.Json{
					"buttons": s.btns(),
				},
			},
		}
	case "list":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"type":              "interactive",
			"to":                s.to,
			"interactive": et.Json{
				"type":   "list",
				"header": s.Header.body(),
				"body": et.Json{
					"text": s.Text,
				},
				"footer": et.Json{
					"text": s.Footer.Text,
				},
				"action": et.Json{
					"button":   s.Button,
					"sections": s.sections(),
				},
			},
		}
	default:
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"type":              "text",
			"to":                s.to,
			"text": et.Json{
				"preview_url": false,
				"body":        s.Text,
			},
		}
	}
}
