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
	Street  	string    	`json:"street"`
	City    	string    	`json:"city"`
	State   	string    	`json:"state"`
	Zip     	string    	`json:"zip"`
	Country 	string    	`json:"country"`
	CountryCode string 		`json:"country_code"`
	Type		TpAddress 	`json:"type"`
}

type emails struct {
	Email string 	`json:"email"`
	Type  TpAddress `json:"type"`
}

type phones struct {
	Phone 	string 		`json:"phone"`
	WaID  	string 		`json:"wa_id"`
	Type  	TpAddress 	`json:"type"`
}

type Urls struct {
	Url 	string 		`json:"url"`
	Type 	TpAddress 	`json:"type"`
}

type Contact struct {
	Address 		[]Address 	`json:"Address"`
	Birthday		string 		`json:"birthday"`
	Email    		[]emails 	`json:"email"`
	FormatedName 	string 		`json:"formated_name"`
	FirstName 		string 		`json:"first_name"`
	LastName 		string 		`json:"last_name"`
	MiddleName 		string 		`json:"middle_name"`
	Suffix 			string 		`json:"suffix"`
	Prefix 			string 		`json:"prefix"`
	OrgCompany 		string 		`json:"org_company"`
	OrgDepartment 	string 		`json:"org_department"`
	OrgTitle 		string 		`json:"org_title"`
	Phones 			[]Phones 	`json:"Phones"`
	Url 			[]Urls 		`json:"urls"`
}
type components struct {
	Type 			string 		`json:"type"`
	Parameters 		[]parameters `json:"parameters"`
}

type parameters struct {
	Header 			string 		`json:"header"`
	Body 			string 		`json:"body"`
	Text 			string 		`json:"text"`
	DateTime		string 		`json:"date_time"`
	Currency		string 		`json:"currency"`
	FallbackValue	string 		`json:"fallback_value"`
	Code 			string 		`json:"code"`	
	DayOfWeek 		string 		`json:"day_of_week"`
	Year 			string 		`json:"year"`
	Month 			string 		`json:"month"`
	DayOfMonth 		string 		`json:"day_of_month"`
	Hour 			string 		`json:"hour"`
	Minute 			string 		`json:"minute"`
	Calendar 		string 		`json:"calendar"`
	Amount1000 		string 		`json:"amount_1000"`	
	Image         	string 		`json:"image"`
	Link 			string 		`json:"link"`
	String          string 		`json:"string"`
	ImageUrl		string 		`json:"image_Url"`
	Payload 		string 		`json:"payload"`
	Button          string 		`json:"button"`
}

type location struct {
	Latitude 	string `json:"latitude"`
	Longitude	string `json:"longitude"`
	Name 		string `json:"name"`
	Address 	string `json:"address"`
}



type template struct {
	Name 		string 		`json:"name"`
	Language 	string 		`json:"language"`
	Code 		string 		`json:"code"`
	Components 	[]components `json:"components"`
}

type Message struct {
	to            		string    `json:"-"`
	kind          		string    `json:"-"`
	Text         		string    `json:"text"`
	Buttons       		[]Button  `json:"buttons"`
	Header        		Header    `json:"header"`
	Footer       		Footer    `json:"footer"`
	Button       		string    `json:"button"`
	Sections      		[]Section `json:"sections"`
	ImageObjectID 		string    `json:"image_object_id"`
	MessageID	 		string    `json:"message_id"`
	Emoji 	      		string    `json:"emoji"`
	URL 	      		string    `json:"url"`
	AudioObjectID 		string    `json:"audio_object_id"`
	DocumentObjectID 	string    `json:"document_object_id"`
	DocumentCaptionText string    `json:"document_caption_text"`
	DocumentFilename 	string    `json:"document_filename"`
	MediaObjectID       string    `json:"media_object_id"`
	VideoObjectID       string    `json:"video_object_id"`
	VideoCaptionText	string    `json:"video_caption_text"`

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
	case "reply":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": {
				"message_id": s.MessageID,
			},
			"type":              "text",
			"text": {
				"preview_url": false,
				"body":        s.Text,
			},}

	case "text_with_preview_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"to":                s.to,
			"text": {
				"preview_url": true,
				"body":        s.URL,
			},}
		
	case "reply_with_reaction":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "reaction",
			"reaction": et.Json{
				"message_id": s.MessageID,
				"emoji": s.Emoji,
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
			"type":              "image",
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
				"link": s.URL,
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
			"type":              "image",
			"image": et.Json{
				"link": s.URL,
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
			"type":              "audio",
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
				"link": s.URL,
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
			"type":              "audio",
			"audio": et.Json{
				"link": s.URL,
			},
		}
	
	case "send_document_by_ID":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "document",
			"document": et.Json{
				"id": s.DocumentObjectID,
				"caption": s.DocumentCaptionText,
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
			"type":              "document",
			"document": et.Json{
				"id": s.DocumentObjectID,
				"caption": s.DocumentCaptionText,
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
				"link": s.URL,
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
			"type":              "document",
			"document": et.Json{
				"link": s.URL,
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
			"type":              "sticker",
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
				"link": s.URL,
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
			"type":              "sticker",
			"sticker": et.Json{
				"link": s.URL,
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
				"id": s.VideoObjectID,
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
			"type":              "video",
			"video": et.Json{
				"caption": s.VideoCaptionText,
				"id": s.VideoObjectID,
			},
		}
	case "send_video_by_URL":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "video",
			"video": et.Json{
				"link": s.URL,
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
			"type":              "video",
			"video": et.Json{
				"link": s.URL,
				"caption": s.VideoCaptionText,
			},
		}	
	case "send_contact":
		return et.Json{
			"messaging_product": "whatsapp",
			"to":                s.to,
			"type":              "contacts",
			"contacts": []et.Json{
				{
					"addresses": []et.Json{
						{
							"street": Address.street,
							"city": Address.city,
							"state": Address.state,
							"zip": Address.zip,
							"country": Address.country,
							"country_code": Address.CountryCode,
							"type": Address.Type,
						},
					},
					"birthday": contact.Birthday,
					"emails": []et.Json{
						{
							"email": emails.Email,
							"type": emails.Type,
						},
					},
					"name": et.Json{
						"formatted_name": contact.FormatedName,
						"first_name": contact.FirstName,
						"last_name": contact.LastName,
						"middle_name": contact.MiddleName,
						"suffix": contact.Suffix,
						"prefix": contact.Prefix,
					},
					"org": et.Json{
						"company": contact.OrgCompany,
						"department": contact.OrgDepartment,
						"title": contact.OrgTitle,
					},
					"phones": []et.Json{
						{
							"phone": phones.Phone,
							"wa_id": phones.WaID,
							"type": phones.Type,
						},
					},
					"urls": []et.Json{
						{
							"url": Urls.Url,
							"type": Urls.Type,
						},
					},
				},
			},
		}
	
	case "reply_to_contact":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type":              "contacts",
			"contacts": []et.Json{
				{
					"addresses": []et.Json{
						{
							"street": Address.street,
							"city": Address.city,
							"state": Address.state,
							"zip": Address.zip,
							"country": Address.country,
							"country_code": Address.CountryCode,
							"type": Address.Type,
						},
					},
					"birthday": contact.Birthday,
					"emails": []et.Json{
						{
							"email": emails.Email,
							"type": emails.Type,
						},
					},
					"name": et.Json{
						"formatted_name": contact.FormatedName,
						"first_name": contact.FirstName,	
						"last_name": contact.LastName,
						"middle_name": contact.MiddleName,
						"suffix": contact.Suffix,
						"prefix": contact.Prefix,
					},
					"org": et.Json{
						"company": contact.OrgCompany,
						"department": contact.OrgDepartment,
						"title": contact.OrgTitle,
					},
					"phones": []et.Json{
						{
							"phone": phones.Phone,
							"wa_id": phones.WaID,
							"type": phones.Type,
						},
					},
					"urls": []et.Json{
						{
							"url": Urls.Url,
							"type": Urls.Type,
						},
					},
				},
			},
		}
	
	case "send_location":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "location",
			"location": et.Json{
				"latitude": location.Latitude,
				"longitude": location.Longitude,
				"name": location.Name,
				"address": location.Address,
			},
		}
	
	case "reply_to_location":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"context": et.Json{
				"message_id": s.MessageID,
			},
			"type":              "location",
			"location": et.Json{
				"latitude": location.Latitude,
				"longitude": location.Longitude,
				"name": location.Name,
				"address": location.Address,
			},
		}
	case "send_template_text":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "template",
			"template": et.Json{
				"name": Template.Name,
				"language": et.Json{
					"code": Template.Language,
				},
				"components": []et.Json{
					{
						"type": components.Body,
						"parameters": []et.Json{
							{
								"type": parameters.Text,
								"text": parameters.Text,
							},
							{
								"type": parameters.Currency,
								"currency": et.Json{
									"fallback_value": parameters.FallbackValue,
									"code": parameters.Code,
									"amount_1000": parameters.Amount1000,
								},
							},
							{
								"type": parameters.DateTime,
								"date_time": {
									"day_of_week": parameters.DayOfWeek,
									"year": parameters.Year,
									"month": parameters.Month,
									"day_of_month": parameters.DayOfMonth,
									"hour": parameters.Hour,
									"minute": parameters.Minute,
									"calendar": parameters.Calendar,
								},
							},
						},
					},
				},
			},
		}
	
	case "send_template_media":
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "template",
			"template": et.Json{
				"name": Template.Name,
				"language": et.Json{
					"code": Template.Language,
				},
				"components": []et.Json{
					{
						"type": components.Header,
						"parameters": []et.Json{
							{
								"type": parameters.Image,
								"image": et.Json{
									"link": parameters.ImageUrl,
								},
							},
						},
					},
				},
			},
		},
		{
			"type": parameter.Body,
			"parameters": []et.Json{
				{
					"type": parameters.Text,
					"text": parameters.String,
				},
				{
					"type": parameters.Currency,
					"currency": {
						"fallback_value": parameters.FallbackValue,
						"code": parameters.Code,
						"amount_1000": parameters.Amount1000,
					},
				},
				{
					"type": parameters.DateTime,
					"date_time": {
						"fallback_value": parameters.FallbackValue,
						"day_of_week": parameters.DayOfWeek,
						"year": parameters.Year,
						"month": parameters.Month,
						"day_of_month": parameters.DayOfMonth,
						"hour": parameters.Hour,
						"minute": parameters.Minute,
						"calendar": parameters.Calendar,
					},
				}
			]
		}	
					
	default:
		return et.Json{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                s.to,
			"type":              "text",
			"text": {
				"preview_url": false,
				"body":        s.Text,
			},
		}
	}
}
