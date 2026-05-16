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

type TpAddress string

const (
	HOME TpAddress = "home"
	WORK TpAddress = "work"
)

type Address struct {
	Street      string    `json:"street"`
	City        string    `json:"city"`
	State       string    `json:"state"`
	Zip         string    `json:"zip"`
	Country     string    `json:"country"`
	CountryCode string    `json:"country_code"`
	Type        TpAddress `json:"type"`
}

type Email struct {
	Email string    `json:"email"`
	Type  TpAddress `json:"type"`
}

type Phone struct {
	Phone string    `json:"phone"`
	WaID  string    `json:"wa_id"`
	Type  TpAddress `json:"type"`
}

type Url struct {
	Url  string    `json:"url"`
	Type TpAddress `json:"type"`
}

type Contact struct {
	Address       []Address `json:"Address"`
	Birthday      string    `json:"birthday"`
	Emails        []Email   `json:"emails"`
	FormatedName  string    `json:"formated_name"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	MiddleName    string    `json:"middle_name"`
	Suffix        string    `json:"suffix"`
	Prefix        string    `json:"prefix"`
	OrgCompany    string    `json:"org_company"`
	OrgDepartment string    `json:"org_department"`
	OrgTitle      string    `json:"org_title"`
	Phones        []Phone   `json:"phones"`
	Urls          []Url     `json:"urls"`
}

type Action struct {
	Button   string        `json:"button"`
	Sections ActionSection `json:"sections"`
}

type ActionSection struct {
	Title                      string    `json:"title"`
	Rows                       ActionRow `json:"rows"`
	CatalogID                  string    `json:"catalog_id"`
	ProductRetailerID          string    `json:"product_retailer_id"`
	ThumbnailProductRetailerID string    `json:"thumbnail_product_retailer_id"`
}

type ActionRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Parameter struct {
	Header        string `json:"header"`
	Body          string `json:"body"`
	Text          string `json:"text"`
	DateTime      string `json:"date_time"`
	Currency      string `json:"currency"`
	FallbackValue string `json:"fallback_value"`
	Code          string `json:"code"`
	DayOfWeek     string `json:"day_of_week"`
	Year          string `json:"year"`
	Month         string `json:"month"`
	DayOfMonth    string `json:"day_of_month"`
	Hour          string `json:"hour"`
	Minute        string `json:"minute"`
	Calendar      string `json:"calendar"`
	Amount1000    string `json:"amount_1000"`
	Image         string `json:"image"`
	Link          string `json:"link"`
	String        string `json:"string"`
	ImageUrl      string `json:"image_Url"`
	Payload       string `json:"payload"`
	Button        string `json:"button"`
	Type          string `json:"type"`
	Index         string `json:"index"`
}

type Component struct {
	Type      string    `json:"type"`
	Parameter Parameter `json:"parameter"`
}

type Location struct {
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
	Name      string `json:"name"`
	Address   string `json:"address"`
}

type Template struct {
	Name       string      `json:"name"`
	Language   string      `json:"language"`
	Code       string      `json:"code"`
	Components []Component `json:"components"`
}

type Message struct {
	to                  string    `json:"-"`
	kind                string    `json:"-"`
	Text                string    `json:"text"`
	Buttons             []Button  `json:"buttons"`
	Header              Header    `json:"header"`
	Footer              Footer    `json:"footer"`
	Button              string    `json:"button"`
	Sections            []Section `json:"sections"`
	ImageObjectID       string    `json:"image_object_id"`
	MessageID           string    `json:"message_id"`
	Emoji               string    `json:"emoji"`
	AudioObjectID       string    `json:"audio_object_id"`
	DocumentObjectID    string    `json:"document_object_id"`
	DocumentCaptionText string    `json:"document_caption_text"`
	DocumentFilename    string    `json:"document_filename"`
	MediaObjectID       string    `json:"media_object_id"`
	VideoObjectID       string    `json:"video_object_id"`
	VideoCaptionText    string    `json:"video_caption_text"`
	Address             Address   `json:"address"`
	Contact             Contact   `json:"contact"`
	Email               Email     `json:"email"`
	Phone               Phone     `json:"phone"`
	Url                 Url       `json:"url"`
	Location            Location  `json:"location"`
	Template            Template  `json:"template"`
	Action              Action    `json:"action"`
	Component           Component `json:"component"`
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
			"to":                s.to,
			"type":              "interactive",
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
			"to":                s.to,
			"type":              "interactive",
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

	case "reply_list":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "interactive",
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

	case "reply_button":
		Button := s.Buttons[0]
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "interactive",
			"interactive": et.Json{
				"type": "button",
				"body": et.Json{
					"text": s.Text,
				},
				"action": et.Json{
					"buttons": et.Json{
						"type": "reply",
						"reply": et.Json{
							"id":    Button.ID,
							"title": Button.Text,
						},
					},
				},
			},
		}

	case "reply":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "text",
			"text": et.Json{
				"preview_url": false,
				"body":        s.Text,
			}}

	case "text_with_preview_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"to":                s.to,
			"text": et.Json{
				"preview_url": true,
				"body":        s.Url,
			}}

	case "reply_with_reaction":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "reaction",
			"reaction": et.Json{
				"message_id": s.MessageID,
				"emoji":      s.Emoji,
			},
		}
	case "send_image":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "image",
			"image": et.Json{
				"id": s.ImageObjectID,
			},
		}
	case "reply_to_image_by_ID":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "image",
			"image": et.Json{
				"id": s.ImageObjectID,
			},
		}
	case "send_image_by_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "image",
			"image": et.Json{
				"link": s.Url,
			},
		}
	case "reply_to_image_by_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "image",
			"image": et.Json{
				"link": s.Url,
			},
		}
	case "send_audio_by_ID":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "audio",
			"audio": et.Json{
				"id": s.AudioObjectID,
			},
		}
	case "reply_to_audio_by_ID":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "audio",
			"audio": et.Json{
				"id": s.AudioObjectID,
			},
		}
	case "send_audio_by_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "audio",
			"audio": et.Json{
				"link": s.Url,
			},
		}
	case "reply_to_audio_by_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "audio",
			"audio": et.Json{
				"link": s.Url,
			},
		}

	case "send_document_by_ID":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "document",
			"document": et.Json{
				"id":       s.DocumentObjectID,
				"caption":  s.DocumentCaptionText,
				"filename": s.DocumentFilename,
			},
		}
	case "reply_to_document_by_ID":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "document",
			"document": et.Json{
				"id":       s.DocumentObjectID,
				"caption":  s.DocumentCaptionText,
				"filename": s.DocumentFilename,
			},
		}
	case "send_document_by_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "document",
			"document": et.Json{
				"link":    s.Url,
				"caption": s.DocumentCaptionText,
			},
		}
	case "reply_to_document_by_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "document",
			"document": et.Json{
				"link":    s.Url,
				"caption": s.DocumentCaptionText,
			},
		}
	case "send_sticker_message_by_ID":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "sticker",
			"sticker": et.Json{
				"id": s.MediaObjectID,
			},
		}
	case "reply_to_sticker_message_by_ID":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "sticker",
			"sticker": et.Json{
				"id": s.MediaObjectID,
			},
		}
	case "send_sticker_message_by_URL":

		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "sticker",
			"sticker": et.Json{
				"link": s.Url,
			},
		}
	case "reply_to_sticker_message_by_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "sticker",
			"sticker": et.Json{
				"link": s.Url,
			},
		}
	case "send_video_by_ID":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "video",
			"video": et.Json{
				"caption": s.VideoCaptionText,
				"id":      s.VideoObjectID,
			},
		}
	case "reply_to_video_by_ID":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "video",
			"video": et.Json{
				"caption": s.VideoCaptionText,
				"id":      s.VideoObjectID,
			},
		}
	case "send_video_by_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "video",
			"video": et.Json{
				"link":    s.Url,
				"caption": s.VideoCaptionText,
			},
		}
	case "reply_to_video_by_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "video",
			"video": et.Json{
				"link":    s.Url,
				"caption": s.VideoCaptionText,
			},
		}
	case "send_contact":
		address := s.Address
		contact := s.Contact
		email := s.Email
		phone := s.Phone
		url := s.Url
		return et.Json{
			"messaging_product": "whatsapp",
			"to":                s.to,
			"type":              "contacts",
			"contacts": []et.Json{
				{
					"addresses": []et.Json{
						{
							"street":       address.Street,
							"city":         address.City,
							"state":        address.State,
							"zip":          address.Zip,
							"country":      address.Country,
							"country_code": address.CountryCode,
							"type":         address.Type,
						},
					},
					"birthday": contact.Birthday,
					"emails": []et.Json{
						{
							"email": email.Email,
							"type":  email.Type,
						},
					},
					"name": et.Json{
						"formatted_name": contact.FormatedName,
						"first_name":     contact.FirstName,
						"last_name":      contact.LastName,
						"middle_name":    contact.MiddleName,
						"suffix":         contact.Suffix,
						"prefix":         contact.Prefix,
					},
					"org": et.Json{
						"company":    contact.OrgCompany,
						"department": contact.OrgDepartment,
						"title":      contact.OrgTitle,
					},
					"phones": []et.Json{
						{
							"phone": phone.Phone,
							"wa_id": phone.WaID,
							"type":  phone.Type,
						},
					},
					"urls": []et.Json{
						{
							"url":  url.Url,
							"type": url.Type,
						},
					},
				},
			},
		}

	case "reply_to_contact":
		address := s.Address
		contact := s.Contact
		email := s.Email
		phone := s.Phone
		url := s.Url
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "contacts",
			"contacts": []et.Json{
				{
					"addresses": []et.Json{
						{
							"street":       address.Street,
							"city":         address.City,
							"state":        address.State,
							"zip":          address.Zip,
							"country":      address.Country,
							"country_code": address.CountryCode,
							"type":         address.Type,
						},
					},
					"birthday": contact.Birthday,
					"emails": []et.Json{
						{
							"email": email.Email,
							"type":  email.Type,
						},
					},
					"name": et.Json{
						"formatted_name": contact.FormatedName,
						"first_name":     contact.FirstName,
						"last_name":      contact.LastName,
						"middle_name":    contact.MiddleName,
						"suffix":         contact.Suffix,
						"prefix":         contact.Prefix,
					},
					"org": et.Json{
						"company":    contact.OrgCompany,
						"department": contact.OrgDepartment,
						"title":      contact.OrgTitle,
					},
					"phones": []et.Json{
						{
							"phone": phone.Phone,
							"wa_id": phone.WaID,
							"type":  phone.Type,
						},
					},
					"urls": []et.Json{
						{
							"url":  url.Url,
							"type": url.Type,
						},
					},
				},
			},
		}

	case "send_location":
		location := s.Location
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "location",
			"location": et.Json{
				"latitude":  location.Latitude,
				"longitude": location.Longitude,
				"name":      location.Name,
				"address":   location.Address,
			},
		}

	case "reply_to_location":
		location := s.Location
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type": "location",
			"location": et.Json{
				"latitude":  location.Latitude,
				"longitude": location.Longitude,
				"name":      location.Name,
				"address":   location.Address,
			},
		}
	case "send_template_text":
		template := s.Template
		component := s.Component
		parameter := component.Parameter
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "template",
			"template": et.Json{
				"name": template.Name,
				"language": et.Json{
					"code": template.Language,
				},
				"components": []et.Json{
					{
						"type": "body",
						"parameters": []et.Json{
							{
								"type": "text",
								"text": parameter.Text,
							},
							{
								"type": "currency",
								"currency": et.Json{
									"fallback_value": parameter.FallbackValue,
									"code":           parameter.Code,
									"amount_1000":    parameter.Amount1000,
								},
							},
							{
								"type": "date_time",
								"date_time": et.Json{
									"day_of_week":  parameter.DayOfWeek,
									"year":         parameter.Year,
									"month":        parameter.Month,
									"day_of_month": parameter.DayOfMonth,
									"hour":         parameter.Hour,
									"minute":       parameter.Minute,
									"calendar":     parameter.Calendar,
								},
							},
						},
					},
				},
			},
		}

	case "send_template_media":
		template := s.Template
		component := s.Component
		parameter := component.Parameter
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "template",
			"template": et.Json{
				"name": template.Name,
				"language": et.Json{
					"code": template.Language,
				},
				"components": []et.Json{
					{
						"type": "header",
						"parameters": []et.Json{
							{
								"type": "image",
								"image": et.Json{
									"link": parameter.ImageUrl,
								},
							},
							{
								"type": "body",
								"parameters": []et.Json{
									{
										"type": "text",
										"text": parameter.Text,
									},
									{
										"type": "currency",
										"currency": et.Json{
											"fallback_value": parameter.FallbackValue,
											"code":           parameter.Code,
											"amount_1000":    parameter.Amount1000,
										},
									},
									{
										"type": "date_time",
										"date_time": et.Json{
											"fallback_value": parameter.FallbackValue,
											"day_of_week":    parameter.DayOfWeek,
											"year":           parameter.Year,
											"month":          parameter.Month,
											"day_of_month":   parameter.DayOfMonth,
											"hour":           parameter.Hour,
											"minute":         parameter.Minute,
											"calendar":       parameter.Calendar,
										},
									},
								},
							},
						},
					},
				},
			},
		}

	case "send_template_interactive":
		template := s.Template
		component := s.Component
		parameter := component.Parameter
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "template",
			"template": et.Json{
				"name": template.Name,
				"language": et.Json{
					"code": template.Language,
				},
				"components": []et.Json{
					{
						"type": "header",
						"parameters": []et.Json{
							{
								"type": "image",
								"image": et.Json{
									"link": parameter.ImageUrl,
								},
							},
							{
								"type": "body",
								"parameters": []et.Json{
									{
										"type": "text",
										"text": parameter.Text,
									},
									{
										"type": "currency",
										"currency": et.Json{
											"fallback_value": parameter.FallbackValue,
											"code":           parameter.Code,
											"amount_1000":    parameter.Amount1000,
										},
									},
									{
										"type": "date_time",
										"date_time": et.Json{
											"fallback_value": parameter.FallbackValue,
											"day_of_week":    parameter.DayOfWeek,
											"year":           parameter.Year,
											"month":          parameter.Month,
											"day_of_month":   parameter.DayOfMonth,
											"hour":           parameter.Hour,
											"minute":         parameter.Minute,
											"calendar":       parameter.Calendar,
										},
									},
									{
										"type":     "button",
										"sub_type": "quick_reply",
										"index":    parameter.Index,
										"parameters": []et.Json{
											{
												"type":    "payload",
												"payload": parameter.Payload,
											},
										},
									},
									{
										"type":     "button",
										"sub_type": "quick_reply",
										"index":    parameter.Index,
										"parameters": []et.Json{
											{
												"type":    "payload",
												"payload": parameter.Payload,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

	case "single_product":
		sections := s.Action.Sections
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "interactive",
			"interactive": et.Json{
				"type": "product",
				"body": et.Json{
					"text": s.Text,
				},
				"footer": et.Json{
					"text": s.Footer,
				},
				"action": et.Json{
					"catalog_id":          sections.CatalogID,
					"product_retailer_id": sections.ProductRetailerID,
				},
			},
		}
	case "multi_product":
		sections := s.Action.Sections
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "interactive",
			"interactive": et.Json{
				"type": "product_list",
				"header": et.Json{
					"type": "text",
					"text": s.Header,
				},
				"body": et.Json{
					"text": s.Text,
				},
				"footer": et.Json{
					"text": s.Footer,
				},
				"action": et.Json{
					"catalog_id": sections.CatalogID,
					"sections": []et.Json{
						{
							"title": sections.Title,
							"product_items": []et.Json{
								{
									"product_retailer_id": sections.ProductRetailerID,
								},
								{
									"product_retailer_id": sections.ProductRetailerID,
								},
							},
						},
						{
							"title": sections.Title,
							"product_items": []et.Json{
								{
									"product_retailer_id": sections.ProductRetailerID,
								},
								{
									"product_retailer_id": sections.ProductRetailerID,
								},
							},
						},
					},
				},
			},
		}
	case "catalog":
		sections := s.Action.Sections
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "interactive",
			"interactive": et.Json{
				"type": "catalog_message",
				"body": et.Json{
					"text": s.Text,
				},
				"action": et.Json{
					"type": "catalog_message",
					"parameters": et.Json{
						"thumbnail_product_retailer_id": sections.ThumbnailProductRetailerID,
					},
				},
				"footer": et.Json{
					"text": s.Footer,
				},
			},
		}
	case "catalog_template":
		sections := s.Action.Sections
		template := s.Template
		parameter := s.Component.Parameter
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "template",
			"template": et.Json{
				"name": template.Name,
				"language": et.Json{
					"code": template.Language,
				},
				"components": []et.Json{
					{
						"type": "body",
						"parameters": []et.Json{
							{
								"type": "text",
								"text": s.Text,
							},
							{
								"type": "text",
								"text": s.Text,
							},
							{
								"type": "text",
								"text": s.Text,
							},
						},
					},
					{
						"type":     "button",
						"sub_type": "CATALOG",
						"index":    parameter.Index,
						"parameters": []et.Json{
							{
								"type": "action",
								"action": et.Json{
									"tumbnail_product_retailer_id": sections.ThumbnailProductRetailerID,
								},
							},
						},
					},
				},
			},
		}
	default:
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "text",
			"text": et.Json{
				"preview_url": false,
				"body":        s.Text,
			},
		}
	}
}
