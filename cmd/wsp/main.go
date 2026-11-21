package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/cgalvisleon/et/jwsp"
	"github.com/cgalvisleon/et/logs"
)

/**
* main: el ejemplo de jwsp. Sin -token envía contra una Graph API falsa local (sin red); con -token, contra Meta. Con -listen
* atiende el webhook (GET de suscripción y POST firmado) e imprime los eventos
**/
func main() {
	token := flag.String("token", "", "el token de acceso (vacío: Graph API falsa local)")
	phone := flag.String("phone", "phone_number_id", "el phone_number_id que envía")
	to := flag.String("to", "573001234567", "el destinatario")
	text := flag.String("text", "Hola desde jwsp", "el texto")
	listen := flag.String("listen", "", "dirección del webhook, p. ej. :8080 (vacío: no atiende)")
	verify := flag.String("verify_token", "", "el verify token del webhook")
	secret := flag.String("app_secret", "", "el app secret con que Meta firma el webhook")
	flag.Parse()

	client := jwsp.New(*token, *phone)
	if *token == "" {
		fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			logs.Debug(r.Method, " ", r.URL.Path, " ", string(body))
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"messaging_product":"whatsapp","contacts":[{"input":"`+*to+`","wa_id":"`+*to+`"}],"messages":[{"id":"wamid.FAKE"}]}`)
		}))
		defer fake.Close()
		client = jwsp.New("fake-token", *phone)
		client.BaseURL = fake.URL
	}

	result, err := client.Send(context.Background(), jwsp.TextMessage(*to, *text))
	if err != nil {
		if failure, ok := jwsp.AsError(err); ok && failure.IsReengagement() {
			logs.Alertm("fuera de la ventana de 24 h: envía una plantilla")
		}
		logs.Error(err)
		return
	}
	logs.Debug("message_id: ", result.MessageId)

	if *listen == "" {
		return
	}

	handler := jwsp.Handler(*verify, *secret, func(events []jwsp.Event) {
		for _, event := range events {
			switch event.Kind {
			case jwsp.EventMessage:
				logs.Debug("mensaje de ", event.Message.From, ": ", event.Message.Text)
			case jwsp.EventStatus:
				logs.Debug("estado ", event.Status.MessageId, ": ", event.Status.Status)
			case jwsp.EventError:
				logs.Debug("error ", event.Error.Code, ": ", event.Error.Message)
			}
		}
	})
	logs.Debug("webhook en ", *listen)
	if err := http.ListenAndServe(*listen, handler); err != nil {
		logs.Error(err)
	}
}
