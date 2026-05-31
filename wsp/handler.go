package wsp

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
)

/**
* webhooks
* @param w http.ResponseWriter, r *http.Request
**/
func (s *Whatsapp) Webhooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mode := request.Query(r, "hub.mode").Str()
		token := request.Query(r, "hub.verify_token").Str()
		challenge := request.Query(r, "hub.challenge").Str()
		if mode == "subscribe" && token == s.VerifyToken {
			if s.EventHandler != nil {
				s.EventHandler(et.Json{
					"kind": mode,
					"data": et.Json{
						"hub.mode":         mode,
						"hub.challenge":    challenge,
						"hub.verify_token": token,
					},
				})
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}
		response.HTTPError(w, r, http.StatusForbidden, http.StatusText(http.StatusForbidden))
	case http.MethodPost:
		body, err := request.GetBody(r)
		if err != nil {
			if s.EventHandlerError != nil {
				s.EventHandlerError(err)
			}
			return
		}
		if s.EventHandler != nil {
			s.EventHandler(et.Json{
				"kind": "conversation",
				"data": body,
			})
		}
		w.WriteHeader(http.StatusOK)
	default:
		response.HTTPError(w, r, http.StatusMethodNotAllowed, msg.MSG_METHOD_NOT_ALLOWED)
		return
	}
}

/**
* getHeader
* @return et.Json
**/
func (s *Whatsapp) getHeader() et.Json {
	header := et.Json{
		"Authorization": "Bearer " + s.Token,
		"Content-Type":  "application/json",
	}
	return header
}

/**
* sendRequest
* @param message *Message
* @return et.Json, error
**/
func (s *Whatsapp) sendRequest(message *Message) (et.Json, error) {
	url := fmt.Sprintf("%s/%s/messages", s.Path, s.PhoneNumberId)
	body := message.body()
	if s.isDebug {
		logs.Debug("body:", body.ToString())
	}

	if s.isTest {
		return body, nil
	}

	res, status := request.Post(url, s.getHeader(), body)
	if status.Code != 200 {
		return et.Json{}, errors.New(status.Message)
	}

	result, err := res.ToJson()
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* SendTextMessage
* @param to, text string
* @return et.Json, error
**/
func (s *Whatsapp) SendTextMessage(to, text string) (et.Json, error) {
	message := &Message{
		To:   to,
		Text: text,
	}
	message.kind = MessageTypeText

	return s.sendRequest(message)
}

/**
* SendReplyTextMessage
* @param to, message_id, text string
* @return et.Json, error
**/
func (s *Whatsapp) SendReplyTextMessage(to, message_id, text string) (et.Json, error) {
	message := &Message{
		To:        to,
		Text:      text,
		MessageID: message_id,
	}
	message.kind = MessageTypeReplyText

	return s.sendRequest(message)
}

/**
* SendTextMessageWithPreviewURL
* @param to, url, text string
* @return et.Json, error
**/
func (s *Whatsapp) SendTextMessageWithPreviewURL(to, url string) (et.Json, error) {
	message := &Message{
		To:  to,
		Url: Url{Url: url},
	}
	message.kind = MessageTypeTextWithPreviewURL

	return s.sendRequest(message)
}

/**
* SendReplyWithReactionMessage
* @param to, message_id, emoji string
* @return et.Json, error
**/
func (s *Whatsapp) SendReplyWithReactionMessage(to, messageId, emoji string) (et.Json, error) {
	message := &Message{
		To:        to,
		MessageID: messageId,
		Emoji:     emoji,
	}
	message.kind = MessageTypeReplyWithReaction

	return s.sendRequest(message)
}

func (s *Whatsapp) SendImageMessageById(to, imageObjectID string) (et.Json, error) {
	message := &Message{
		To:            to,
		ImageObjectID: imageObjectID,
	}
	message.kind = MessageTypeImageById

	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyImageMessageById(to, imageObjectID, messageID string) (et.Json, error) {
	message := &Message{
		To:            to,
		MessageID:     messageID,
		ImageObjectID: imageObjectID,
	}
	message.kind = MessageTypeReplyImageById

	return s.sendRequest(message)
}

func (s *Whatsapp) SendImageMessageByURL(to, url string) (et.Json, error) {
	message := &Message{
		To:  to,
		Url: Url{Url: url},
	}
	message.kind = MessageTypeImageByURL

	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyImageMessageByURL(to, url, messageID string) (et.Json, error) {
	message := &Message{
		To:        to,
		MessageID: messageID,
		Url:       Url{Url: url},
	}
	message.kind = MessageTypeReplyImageByURL

	return s.sendRequest(message)
}

func (s *Whatsapp) SendAudioMessageById(to, audioObjectID string) (et.Json, error) {
	message := &Message{
		To:            to,
		AudioObjectID: audioObjectID,
	}
	message.kind = MessageTypeAudioById

	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyAudioMessageById(to, audioObjectID, messageID string) (et.Json, error) {
	message := &Message{
		To:            to,
		MessageID:     messageID,
		AudioObjectID: audioObjectID,
	}
	message.kind = MessageTypeReplyAudioById

	return s.sendRequest(message)
}

func (s *Whatsapp) SendAudioMessageByURL(to, url string) (et.Json, error) {
	message := &Message{
		To:  to,
		Url: Url{Url: url},
	}
	message.kind = MessageTypeAudioByURL

	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyAudioMessageByURL(to, url, messageID string) (et.Json, error) {
	message := &Message{
		To:        to,
		MessageID: messageID,
		Url:       Url{Url: url},
	}
	message.kind = MessageTypeReplyAudioByURL

	return s.sendRequest(message)
}

func (s *Whatsapp) SendDocumentById(to, documentObjectID, documentCaptionText, documentFilename string) (et.Json, error) {
	message := &Message{
		To:                  to,
		DocumentObjectID:    documentObjectID,
		DocumentCaptionText: documentCaptionText,
		DocumentFilename:    documentFilename,
	}
	message.kind = MessageTypeDocumentById

	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyDocumentById(to, messageID, documentObjectID, documentCaptionText, documentFilename string) (et.Json, error) {
	message := &Message{
		To:                  to,
		MessageID:           messageID,
		DocumentObjectID:    documentObjectID,
		DocumentCaptionText: documentCaptionText,
		DocumentFilename:    documentFilename,
	}
	message.kind = MessageTypeReplyDocumentById

	return s.sendRequest(message)
}

func (s *Whatsapp) SendDocumentByURL(to, url, documentCaptionText string) (et.Json, error) {
	message := &Message{
		To:                  to,
		Url:                 Url{Url: url},
		DocumentCaptionText: documentCaptionText,
	}
	message.kind = MessageTypeDocumentByURL

	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyDocumentByURL(to, messageID, url, documentCaptionText string) (et.Json, error) {
	message := &Message{
		To:                  to,
		MessageID:           messageID,
		Url:                 Url{Url: url},
		DocumentCaptionText: documentCaptionText,
	}
	message.kind = MessageTypeReplyDocumentByURL

	return s.sendRequest(message)
}

func (s *Whatsapp) SendStickerMessageById(to, mediaObjectID string) (et.Json, error) {
	message := &Message{
		To:            to,
		MediaObjectID: mediaObjectID,
	}
	message.kind = MessageTypeStickerById
	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyStickerMessageById(to, mediaObjectID, messageID string) (et.Json, error) {
	message := &Message{
		To:            to,
		MediaObjectID: mediaObjectID,
		MessageID:     messageID,
	}
	message.kind = MessageTypeReplyStickerById

	return s.sendRequest(message)
}

func (s *Whatsapp) SendStickerMessageByURL(to, url string) (et.Json, error) {
	message := &Message{
		To:  to,
		Url: Url{Url: url},
	}
	message.kind = MessageTypeStickerByURL

	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyStickerMessageByURL(to, messageID, url string) (et.Json, error) {
	message := &Message{
		To:        to,
		MessageID: messageID,
		Url:       Url{Url: url},
	}
	message.kind = MessageTypeReplyStickerByURL

	return s.sendRequest(message)
}

func (s *Whatsapp) SendVideoMessageById(to, videoCaptionText, videoObjectID string) (et.Json, error) {
	message := &Message{
		To:               to,
		VideoCaptionText: videoCaptionText,
		VideoObjectID:    videoObjectID,
	}
	message.kind = MessageTypeVideoById
	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyVideoMessageById(to, messageID, videoCaptionText, videoObjectID string) (et.Json, error) {
	message := &Message{
		To:               to,
		MessageID:        messageID,
		VideoCaptionText: videoCaptionText,
		VideoObjectID:    videoObjectID,
	}
	message.kind = MessageTypeReplyVideoById
	return s.sendRequest(message)
}

func (s *Whatsapp) SendVideoMessageByURL(to, url, videoCaptionText string) (et.Json, error) {
	message := &Message{
		To:               to,
		Url:              Url{Url: url},
		VideoCaptionText: videoCaptionText,
	}
	message.kind = MessageTypeVideoByURL
	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyVideoMessageByURL(to, url, videoCaptionText string) (et.Json, error) {
	message := &Message{
		To:               to,
		MessageID:        url,
		Url:              Url{Url: url},
		VideoCaptionText: videoCaptionText,
	}
	message.kind = MessageTypeReplyVideoByURL
	return s.sendRequest(message)
}

func (s *Whatsapp) SendContact(to, address, contact, email, phone, tp, url string) (et.Json, error) {

	message := &Message{
		To: to,
		Address: Address{
			Street:      address,
			City:        address,
			Zip:         address,
			Country:     address,
			CountryCode: address,
			Type:        TpAddress(tp),
		},
		Contact: Contact{
			Birthday:      contact,
			FormatedName:  contact,
			FirstName:     contact,
			LastName:      contact,
			Suffix:        contact,
			Prefix:        contact,
			OrgCompany:    contact,
			OrgDepartment: contact,
			OrgTitle:      contact,
		},
		Email: Email{
			Email: email,
			Type:  TpAddress(tp),
		},
		Phone: Phone{
			Phone: phone,
			WaID:  phone,
			Type:  TpAddress(tp),
		},
		Url: Url{
			Url:  url,
			Type: TpAddress(tp),
		},
	}
	message.kind = MessageTypeSendContact
	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyContact(to, messageID, address, contact, email, phone, tp, url string) (et.Json, error) {
	message := &Message{
		To:        to,
		MessageID: messageID,
		Address: Address{
			Street:      address,
			City:        address,
			Zip:         address,
			Country:     address,
			CountryCode: address,
			Type:        TpAddress(tp),
		},
		Contact: Contact{
			Birthday:      contact,
			FormatedName:  contact,
			FirstName:     contact,
			LastName:      contact,
			Suffix:        contact,
			Prefix:        contact,
			OrgCompany:    contact,
			OrgDepartment: contact,
			OrgTitle:      contact,
		},
		Email: Email{
			Email: email,
			Type:  TpAddress(tp),
		},
		Phone: Phone{
			Phone: phone,
			WaID:  phone,
			Type:  TpAddress(tp),
		},
		Url: Url{
			Url:  url,
			Type: TpAddress(tp),
		},
	}
	message.kind = MessageTypeSendReplyContact
	return s.sendRequest(message)
}

func (s *Whatsapp) SendLocation(to, location string) (et.Json, error) {
	message := &Message{
		To: to,
		Location: Location{
			Latitude:  location,
			Longitude: location,
			Name:      location,
			Address:   location,
		},
	}
	message.kind = MessageTypeSendLocation
	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyLocation(to, messageID, location string) (et.Json, error) {
	message := &Message{
		To:        to,
		MessageID: messageID,
		Location: Location{
			Latitude:  location,
			Longitude: location,
			Name:      location,
			Address:   location,
		},
	}
	message.kind = MessageTypeSendReplyLocation
	return s.sendRequest(message)
}

func (s *Whatsapp) SendTemplate(to, parameter, template string) (et.Json, error) {
	message := &Message{
		To: to,
		Template: Template{
			Name:     template,
			Language: template,
		},
		Parameter: Parameter{
			Text:          parameter,
			FallbackValue: parameter,
			Code:          parameter,
			Amount1000:    parameter,
			DayOfWeek:     parameter,
			Year:          parameter,
			Month:         parameter,
			DayOfMonth:    parameter,
			Hour:          parameter,
			Minute:        parameter,
			Calendar:      parameter,
		},
	}
	message.kind = MessageTypeSendTemplate
	return s.sendRequest(message)
}

func (s *Whatsapp) SendTemplateMedia(to, parameter, template string) (et.Json, error) {
	message := &Message{
		To: to,
		Template: Template{
			Name:     template,
			Language: template,
		},

		Parameter: Parameter{
			ImageUrl:      parameter,
			Text:          parameter,
			FallbackValue: parameter,
			Code:          parameter,
			Amount1000:    parameter,
			DayOfWeek:     parameter,
			Year:          parameter,
			Month:         parameter,
			DayOfMonth:    parameter,
			Hour:          parameter,
			Minute:        parameter,
			Calendar:      parameter,
		},
	}
	message.kind = MessageTypeSendTemplateMedia
	return s.sendRequest(message)
}

func (s *Whatsapp) SendTemplateInteractive(to, parameter, template string) (et.Json, error) {
	message := &Message{
		To: to,
		Template: Template{
			Name:     template,
			Language: template,
		},

		Parameter: Parameter{
			ImageUrl:      parameter,
			Text:          parameter,
			FallbackValue: parameter,
			Code:          parameter,
			Amount1000:    parameter,
			DayOfWeek:     parameter,
			Year:          parameter,
			Month:         parameter,
			DayOfMonth:    parameter,
			Hour:          parameter,
			Minute:        parameter,
			Calendar:      parameter,
			Index:         parameter,
			Payload:       parameter,
		},
	}
	message.kind = MessageTypeSendTemplateInteractive
	return s.sendRequest(message)
}

func (s *Whatsapp) SendSingleProduct(to, Text, footer, section string) (et.Json, error) {
	message := &Message{
		To:   to,
		Text: Text,
		Footer: Footer{
			Text: footer,
		},
		ActionSection: ActionSection{
			CatalogID:         section,
			ProductRetailerID: section,
		},
	}
	message.kind = MessageTypeSingleProduct
	return s.sendRequest(message)
}

func (s *Whatsapp) SendmultiProduct(to, Text, header, footer, section string) (et.Json, error) {
	message := &Message{
		To:   to,
		Text: Text,
		Header: Header{
			Content: header,
		},
		Footer: Footer{
			Text: footer,
		},
		ActionSection: ActionSection{
			CatalogID:         section,
			ProductRetailerID: section,
		},
	}
	message.kind = MessageTypeMultiProduct
	return s.sendRequest(message)
}

func (s *Whatsapp) SendCatalog(to, Text, footer, section string) (et.Json, error) {
	message := &Message{
		To:   to,
		Text: Text,
		ActionSection: ActionSection{
			ThumbnailProductRetailerID: section,
		},
		Footer: Footer{
			Text: footer,
		},
	}
	message.kind = MessageTypeCatalog
	return s.sendRequest(message)
}

func (s *Whatsapp) SendCatalogTemplate(to, template, parameter, section string) (et.Json, error) {
	message := &Message{
		To: to,
		Template: Template{
			Name:     template,
			Language: template,
		},
		Parameter: Parameter{
			Text:  parameter,
			Index: parameter,
		},
		ActionSection: ActionSection{
			ThumbnailProductRetailerID: section,
		},
	}

	message.kind = MessageTypeCatalogTemplate
	return s.sendRequest(message)
}

func (s *Whatsapp) SendList(to, text, header, footer, button, section string) (et.Json, error) {
	message := &Message{
		To: to,
		Header: Header{
			Content: header,
		},
		Text: text,
		Footer: Footer{
			Text: footer,
		},
		Button:   button,
		Sections: []Section{},
	}

	message.kind = MessageTypeList
	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyList(to, messageId, text, header, footer, button, section string) (et.Json, error) {
	message := &Message{
		To:        to,
		MessageID: messageId,
		Header: Header{
			Content: header,
		},
		Text: text,
		Footer: Footer{
			Text: footer,
		},
		Button:   button,
		Sections: []Section{},
	}

	message.kind = MessageTypeReplyList
	return s.sendRequest(message)
}

func (s *Whatsapp) SendReplyButton(to, text string, buttons []Button) (et.Json, error) {
	message := &Message{
		To:      to,
		Text:    text,
		Buttons: buttons,
	}

	message.kind = MessageTypeReplyButton
	return s.sendRequest(message)
}
