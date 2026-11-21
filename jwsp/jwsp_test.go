package jwsp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
)

// call: una llamada que recibió la Graph API falsa
type call struct {
	method string
	path   string
	auth   string
	ctype  string
	raw    []byte
	body   et.Json
}

/**
* fakeGraph: una Graph API falsa que guarda cada llamada y responde lo que answer diga
* @param t *testing.T, calls *[]call, answer func(item call) (int, string)
* @return *httptest.Server
**/
func fakeGraph(t *testing.T, calls *[]call, answer func(item call) (int, string)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		item := call{method: r.Method, path: r.URL.RequestURI(), auth: r.Header.Get("Authorization"), ctype: r.Header.Get("Content-Type"), raw: raw, body: et.Json{}}
		if strings.HasPrefix(item.ctype, "application/json") && len(raw) > 0 {
			if err := json.Unmarshal(raw, &item.body); err != nil {
				t.Errorf("cuerpo %s: %v", raw, err)
			}
		}
		*calls = append(*calls, item)
		status, output := answer(item)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, output)
	}))
	t.Cleanup(server.Close)
	return server
}

/**
* clientOf: un cliente contra la Graph API falsa, sin esperas largas
* @param url string
* @return *Client
**/
func clientOf(url string) *Client {
	result := New("tok", "1055")
	result.BaseURL = url
	result.WabaId = "waba1"
	result.RetryDelay = time.Millisecond
	return result
}

// okSend: la respuesta de Meta a un envío
const okSend = `{"messaging_product":"whatsapp","contacts":[{"input":"+573001234567","wa_id":"573001234567"}],"messages":[{"id":"wamid.X","message_status":"accepted"}]}`

/**
* TestSendMessages: el cuerpo de cada tipo de mensaje y la respuesta tipada
* @param t *testing.T
**/
func TestSendMessages(t *testing.T) {
	calls := []call{}
	server := fakeGraph(t, &calls, func(call) (int, string) { return http.StatusOK, okSend })
	client := clientOf(server.URL)
	ctx := context.Background()

	text := TextMessage("+573001234567", "Hola")
	text.ReplyTo = "wamid.PREV"
	text.CallbackData = `{"order":"A-1"}`
	template := TemplateMessage("573001234567", "pedido", "es",
		HeaderComponent("Aviso"), BodyComponent("Ana", "A-1"), UrlButtonComponent(0, "track/1"), QuickReplyComponent(1, "NO"), CopyCodeComponent(2, "DESC10"))
	buttons := ButtonsMessage("573001234567", "¿Confirmas?", Button{Id: "si", Title: "Sí"}, Button{Id: "no", Title: "No"})
	list := ListMessage("573001234567", "Elige", "Ver", Section{Title: "Días", Rows: []Row{{Id: "lun", Title: "Lunes"}}})
	messages := []Message{
		text,
		template,
		buttons,
		list,
		UrlButtonMessage("573001234567", "Tu factura", "Abrir", "https://x/f"),
		{To: "573001234567", Document: &Media{Link: "https://x/f.pdf", Caption: "Factura", Filename: "f.pdf"}},
		{To: "573001234567", Image: &Media{Id: "media-1"}},
		{To: "573001234567", Location: &Location{Latitude: 4.6, Longitude: -74.08, Name: "Oficina", Address: "Calle 1"}},
		{To: "573001234567", Contacts: []Contact{{Name: ContactName{FormattedName: "Ana"}, Phones: []ContactPhone{{Phone: "+573001", Type: "CELL"}}}}},
		{To: "573001234567", Reaction: &Reaction{MessageId: "wamid.PREV", Emoji: "👍"}},
	}
	for _, message := range messages {
		result, err := client.Send(ctx, message)
		if err != nil {
			t.Fatal(err)
		}
		if result.MessageId != "wamid.X" || result.WaId != "573001234567" || result.MessageStatus != "accepted" || len(result.Raw) == 0 {
			t.Errorf("resultado %+v", result)
		}
	}

	first := calls[0]
	if first.method != http.MethodPost || first.path != "/v25.0/1055/messages" || first.auth != "Bearer tok" {
		t.Errorf("llamada %+v", first)
	}
	body := first.body
	if body.Str("to") != "573001234567" || body.Str("type") != "text" || body.Json("text").Str("body") != "Hola" || body.Str("messaging_product") != "whatsapp" {
		t.Errorf("texto %v", body)
	}
	if body.Json("context").Str("message_id") != "wamid.PREV" || body.Str("biz_opaque_callback_data") != `{"order":"A-1"}` {
		t.Errorf("contexto o callback %v", body)
	}

	sent := calls[1].body.Json("template")
	components := sent.ArrayJson("components")
	if sent.Str("name") != "pedido" || sent.Json("language").Str("code") != "es" || len(components) != 5 {
		t.Fatalf("plantilla %v", sent)
	}
	if components[1].ArrayJson("parameters")[1].Str("text") != "A-1" {
		t.Errorf("cuerpo %v", components[1])
	}
	url, quick, code := components[2], components[3], components[4]
	if url.Str("sub_type") != "url" || url.Str("index") != "0" || url.ArrayJson("parameters")[0].Str("text") != "track/1" {
		t.Errorf("botón url %v", url)
	}
	if quick.Str("sub_type") != "quick_reply" || quick.ArrayJson("parameters")[0].Str("payload") != "NO" {
		t.Errorf("respuesta rápida %v", quick)
	}
	if code.Str("sub_type") != "copy_code" || code.ArrayJson("parameters")[0].Str("coupon_code") != "DESC10" {
		t.Errorf("copiar código %v", code)
	}

	reply := calls[2].body.Json("interactive")
	button := reply.Json("action").ArrayJson("buttons")[0]
	if reply.Str("type") != "button" || button.Str("type") != "reply" || button.Json("reply").Str("id") != "si" {
		t.Errorf("botones %v", reply)
	}
	rows := calls[3].body.Json("interactive").Json("action").ArrayJson("sections")[0].ArrayJson("rows")
	if calls[3].body.Json("interactive").Json("action").Str("button") != "Ver" || rows[0].Str("id") != "lun" {
		t.Errorf("lista %v", calls[3].body)
	}
	if calls[4].body.Json("interactive").Json("action").Json("parameters").Str("url") != "https://x/f" {
		t.Errorf("cta_url %v", calls[4].body)
	}
	document := calls[5].body.Json("document")
	if document.Str("link") != "https://x/f.pdf" || document.Str("filename") != "f.pdf" || document.Str("caption") != "Factura" {
		t.Errorf("documento %v", document)
	}
	if calls[6].body.Json("image").Str("id") != "media-1" || calls[6].body.Json("image")["link"] != nil {
		t.Errorf("imagen por id %v", calls[6].body)
	}
	location := calls[7].body.Json("location")
	if location.Num("latitude") != 4.6 || location.Num("longitude") != -74.08 || location.Str("name") != "Oficina" || location.Str("address") != "Calle 1" {
		t.Errorf("ubicación %v", location)
	}
	if calls[8].body.ArrayJson("contacts")[0].Json("name").Str("formatted_name") != "Ana" {
		t.Errorf("contactos %v", calls[8].body)
	}
	if calls[9].body.Json("reaction").Str("emoji") != "👍" {
		t.Errorf("reacción %v", calls[9].body)
	}
}

/**
* TestSendInvalid: sin destinatario, sin contenido, con dos contenidos o con callback de más de 512 caracteres falla sin llamar
* @param t *testing.T
**/
func TestSendInvalid(t *testing.T) {
	calls := []call{}
	server := fakeGraph(t, &calls, func(call) (int, string) { return http.StatusOK, okSend })
	client := clientOf(server.URL)
	ctx := context.Background()

	long := TextMessage("1", "x")
	long.CallbackData = strings.Repeat("ñ", CallbackMax+1)
	exact := TextMessage("1", "x")
	exact.CallbackData = strings.Repeat("ñ", CallbackMax)
	invalid := []Message{
		{Text: &Text{Body: "x"}},
		{To: "1"},
		{To: "1", Text: &Text{Body: "x"}, Image: &Media{Id: "m"}},
		long,
	}
	for _, message := range invalid {
		if _, err := client.Send(ctx, message); err == nil {
			t.Errorf("sin error %+v", message)
		}
	}
	if len(calls) != 0 {
		t.Errorf("llamó %d veces", len(calls))
	}
	if _, err := client.Send(ctx, exact); err != nil {
		t.Errorf("512 caracteres: %v", err)
	}

	empty := clientOf(server.URL)
	empty.Token = ""
	if _, err := empty.Send(ctx, TextMessage("1", "x")); err == nil {
		t.Error("sin token")
	}
	empty = clientOf(server.URL)
	empty.PhoneNumberId = ""
	if _, err := empty.Send(ctx, TextMessage("1", "x")); err == nil {
		t.Error("sin número")
	}
}

/**
* TestOtpComponents: el código va en el cuerpo y en el botón url de índice 0
* @param t *testing.T
**/
func TestOtpComponents(t *testing.T) {
	components := OtpComponents("482913")
	raw, _ := json.Marshal(components)
	items := []et.Json{}
	json.Unmarshal(raw, &items)
	if len(items) != 2 || items[0].Str("type") != "body" || items[0].ArrayJson("parameters")[0].Str("text") != "482913" {
		t.Fatalf("cuerpo %v", items)
	}
	if items[1].Str("type") != "button" || items[1].Str("sub_type") != "url" || items[1].Str("index") != "0" {
		t.Errorf("botón %v", items[1])
	}
}

/**
* TestMarkRead: marcar como leído con y sin indicador de escritura
* @param t *testing.T
**/
func TestMarkRead(t *testing.T) {
	calls := []call{}
	server := fakeGraph(t, &calls, func(call) (int, string) { return http.StatusOK, `{"success":true}` })
	client := clientOf(server.URL)

	if err := client.MarkRead(context.Background(), "wamid.IN", true); err != nil {
		t.Fatal(err)
	}
	if err := client.MarkRead(context.Background(), "wamid.IN", false); err != nil {
		t.Fatal(err)
	}
	body := calls[0].body
	if calls[0].path != "/v25.0/1055/messages" || body.Str("status") != "read" || body.Str("message_id") != "wamid.IN" || body.Json("typing_indicator").Str("type") != "text" {
		t.Errorf("leído %v", body)
	}
	if _, has := calls[1].body["typing_indicator"]; has {
		t.Errorf("sin typing %v", calls[1].body)
	}
}

/**
* TestErrors: el error de Meta tipado, 131047, límites y una respuesta que no es JSON
* @param t *testing.T
**/
func TestErrors(t *testing.T) {
	answer := `{"error":{"message":"(#131047) Re-engagement message","type":"OAuthException","code":131047,"error_subcode":2494010,"error_user_title":"Fuera de ventana","error_user_msg":"Han pasado más de 24 horas","error_data":{"details":"more than 24 hours"},"fbtrace_id":"Abc"}}`
	calls := []call{}
	server := fakeGraph(t, &calls, func(call) (int, string) { return http.StatusBadRequest, answer })
	client := clientOf(server.URL)

	_, err := client.Send(context.Background(), TextMessage("1", "x"))
	failure, ok := AsError(err)
	if !ok {
		t.Fatalf("no es *Error: %v", err)
	}
	if failure.Status != 400 || failure.Code != 131047 || failure.Subcode != 2494010 || failure.FbTraceId != "Abc" || failure.Details != "more than 24 hours" {
		t.Errorf("error %+v", failure)
	}
	if !failure.IsReengagement() || failure.IsRateLimit() || failure.IsTemporary() || failure.Text() != "Han pasado más de 24 horas" {
		t.Errorf("clasificación %+v", failure)
	}
	if !strings.Contains(err.Error(), "131047") || !strings.Contains(err.Error(), "24 horas") {
		t.Errorf("mensaje %q", err.Error())
	}
	if len(calls) != 1 {
		t.Errorf("un 400 no se reintenta: %d llamadas", len(calls))
	}

	if !(&Error{Status: 429}).IsRateLimit() || !(&Error{Status: 400, Code: CodeRateLimit}).IsRateLimit() || !(&Error{Status: 503}).IsTemporary() {
		t.Error("límites")
	}

	plain := fakeGraph(t, &[]call{}, func(call) (int, string) { return http.StatusBadGateway, `<html>bad gateway</html>` })
	other := clientOf(plain.URL)
	other.Retries = 0
	_, err = other.Send(context.Background(), TextMessage("1", "x"))
	if failure, ok := AsError(err); !ok || failure.Status != 502 || !strings.Contains(failure.Body, "bad gateway") {
		t.Errorf("sin JSON %v", err)
	}

	// Un 200 con {error} también es un error
	odd := fakeGraph(t, &[]call{}, func(call) (int, string) { return http.StatusOK, `{"error":{"message":"x","code":100}}` })
	if _, err := clientOf(odd.URL).Send(context.Background(), TextMessage("1", "x")); err == nil {
		t.Error("200 con error")
	}
}

/**
* TestRetries: reintenta ante 429 y 5xx hasta Retries y respeta el éxito; sin respuesta, un POST no se reintenta
* @param t *testing.T
**/
func TestRetries(t *testing.T) {
	calls := []call{}
	server := fakeGraph(t, &calls, func(call) (int, string) {
		if len(calls) < 3 {
			return http.StatusTooManyRequests, `{"error":{"message":"rate","code":130429}}`
		}
		return http.StatusOK, okSend
	})
	client := clientOf(server.URL)

	if _, err := client.Send(context.Background(), TextMessage("1", "x")); err != nil {
		t.Fatalf("tras dos reintentos debía pasar: %v", err)
	}
	if len(calls) != 3 {
		t.Errorf("llamadas %d", len(calls))
	}

	failing := []call{}
	down := fakeGraph(t, &failing, func(call) (int, string) {
		return http.StatusServiceUnavailable, `{"error":{"message":"down","code":131016}}`
	})
	client = clientOf(down.URL)
	client.Retries = 1
	_, err := client.Send(context.Background(), TextMessage("1", "x"))
	if failure, ok := AsError(err); !ok || failure.Code != 131016 || len(failing) != 2 {
		t.Errorf("5xx %v, %d llamadas", err, len(failing))
	}

	// Sin respuesta (el servidor corta la conexión): un POST no se reintenta, un GET sí
	var hits atomic.Int32
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	}))
	defer broken.Close()
	client = clientOf(broken.URL)
	if _, err := client.Send(context.Background(), TextMessage("1", "x")); err == nil || hits.Load() != 1 {
		t.Errorf("POST sin respuesta: %v, %d", err, hits.Load())
	}
	hits.Store(0)
	if _, err := client.GetMedia(context.Background(), "m1"); err == nil || hits.Load() != 3 {
		t.Errorf("GET sin respuesta: %v, %d", err, hits.Load())
	}
}

/**
* TestTimeout: el timeout corta una llamada lenta
* @param t *testing.T
**/
func TestTimeout(t *testing.T) {
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer slow.Close()
	defer close(release)
	client := clientOf(slow.URL)
	client.Timeout = 50 * time.Millisecond
	client.Retries = 0

	start := time.Now()
	if _, err := client.Send(context.Background(), TextMessage("1", "x")); err == nil {
		t.Error("sin timeout")
	}
	if time.Since(start) > time.Second {
		t.Errorf("tardó %v", time.Since(start))
	}
}

/**
* TestTemplates: crear (normal y AUTHENTICATION), listar, buscar por nombre e idioma, editar y borrar
* @param t *testing.T
**/
func TestTemplates(t *testing.T) {
	calls := []call{}
	server := fakeGraph(t, &calls, func(item call) (int, string) {
		switch {
		case item.method == http.MethodPost && strings.HasSuffix(item.path, "/message_templates"):
			return http.StatusOK, `{"id":"9001","status":"PENDING","category":"UTILITY"}`
		case item.method == http.MethodGet && strings.Contains(item.path, "after=c1"):
			return http.StatusOK, `{"data":[{"id":"9001","name":"pedido","language":"es","status":"APPROVED","category":"UTILITY","quality_score":{"score":"GREEN"},"components":[{"type":"BODY","text":"Hola {{1}}"}]}]}`
		case item.method == http.MethodGet:
			return http.StatusOK, `{"data":[{"id":"8000","name":"pedido_v2","language":"es","status":"APPROVED"}],"paging":{"cursors":{"after":"c1"},"next":"https://x"}}`
		}
		return http.StatusOK, `{"success":true}`
	})
	client := clientOf(server.URL)
	ctx := context.Background()

	spec := TemplateSpec{Name: "pedido", Language: "es", Category: CategoryUtility, Components: []TemplateComponent{
		{Type: "HEADER", Format: "TEXT", Text: "Aviso {{1}}", Example: &TemplateExample{HeaderText: []string{"Cuenta"}}},
		{Type: "BODY", Text: "Hola {{1}}", Example: &TemplateExample{BodyText: [][]string{{"Ana"}}}},
		{Type: "BUTTONS", Buttons: []TemplateButton{{Type: "QUICK_REPLY", Text: "No fui yo"}}},
	}}
	created, err := client.CreateTemplate(ctx, spec)
	if err != nil || created.Id != "9001" || created.Status != "PENDING" {
		t.Fatalf("crear %+v %v", created, err)
	}
	body := calls[0].body
	components := body.ArrayJson("components")
	if calls[0].path != "/v25.0/waba1/message_templates" || body.Str("category") != "UTILITY" || len(components) != 3 {
		t.Fatalf("cuerpo %v", body)
	}
	if len(components[1].Json("example").Array("body_text")) != 1 || components[2].ArrayJson("buttons")[0].Str("type") != "QUICK_REPLY" {
		t.Errorf("componentes %v", components)
	}

	auth := TemplateSpec{Name: "codigo", Language: "es", Category: CategoryAuthentication, Components: AuthComponents(true, 10, "", "Copiar")}
	if _, err := client.CreateTemplate(ctx, auth); err != nil {
		t.Fatal(err)
	}
	authComponents := calls[1].body.ArrayJson("components")
	otp := authComponents[2].ArrayJson("buttons")[0]
	if !authComponents[0].Bool("add_security_recommendation") || authComponents[0]["text"] != nil || authComponents[1].Int("code_expiration_minutes") != 10 ||
		otp.Str("type") != "OTP" || otp.Str("otp_type") != "COPY_CODE" || otp.Str("text") != "Copiar" {
		t.Errorf("AUTHENTICATION %v", authComponents)
	}
	plain := AuthComponents(false, 0, "", "")
	raw, _ := json.Marshal(plain)
	if len(plain) != 2 || !strings.Contains(string(raw), `"add_security_recommendation":false`) {
		t.Errorf("sin pie %s", raw)
	}

	page, err := client.Templates(ctx, TemplateQuery{Name: "pedido", Status: "APPROVED", Limit: 5})
	if err != nil || len(page.Data) != 1 || page.After != "c1" || page.Data[0].Quality != "UNKNOWN" {
		t.Fatalf("listar %+v %v", page, err)
	}
	if !strings.Contains(calls[2].path, "name=pedido") || !strings.Contains(calls[2].path, "status=APPROVED") || !strings.Contains(calls[2].path, "limit=5") {
		t.Errorf("filtro %s", calls[2].path)
	}

	found, ok, err := client.Template(ctx, "pedido", "es")
	if err != nil || !ok || found.Id != "9001" || found.Quality != "GREEN" || len(found.Components) != 1 || found.Components[0].Text != "Hola {{1}}" {
		t.Fatalf("buscar %+v %v %v", found, ok, err)
	}
	if _, ok, _ := client.Template(ctx, "pedido", "en"); ok {
		t.Error("otro idioma no está")
	}

	if err := client.UpdateTemplate(ctx, "9001", TemplateUpdate{Components: spec.Components}); err != nil {
		t.Fatal(err)
	}
	last := calls[len(calls)-1]
	if last.path != "/v25.0/9001" || len(last.body.ArrayJson("components")) != 3 {
		t.Errorf("editar %+v", last)
	}

	if err := client.DeleteTemplate(ctx, "pedido", ""); err != nil {
		t.Fatal(err)
	}
	if last := calls[len(calls)-1]; last.method != http.MethodDelete || last.path != "/v25.0/waba1/message_templates?name=pedido" {
		t.Errorf("borrar %+v", last)
	}
	if err := client.DeleteTemplate(ctx, "pedido", "9001"); err != nil {
		t.Fatal(err)
	}
	if last := calls[len(calls)-1]; last.path != "/v25.0/waba1/message_templates?hsm_id=9001&name=pedido" {
		t.Errorf("borrar uno %+v", last)
	}

	noWaba := clientOf(server.URL)
	noWaba.WabaId = ""
	if _, err := noWaba.CreateTemplate(ctx, spec); err == nil {
		t.Error("sin waba_id")
	}
	if _, err := noWaba.SubscribeApp(ctx); err == nil {
		t.Error("suscribir sin waba_id")
	}
	if _, err := client.SubscribeApp(ctx); err != nil || calls[len(calls)-1].path != "/v25.0/waba1/subscribed_apps" {
		t.Errorf("suscribir %v", err)
	}
}

/**
* TestMedia: subir (multipart), consultar, descargar y borrar
* @param t *testing.T
**/
func TestMedia(t *testing.T) {
	calls := []call{}
	var server *httptest.Server
	server = fakeGraph(t, &calls, func(item call) (int, string) {
		switch {
		case item.method == http.MethodPost:
			return http.StatusOK, `{"id":"media-1"}`
		case strings.HasPrefix(item.path, "/file"):
			return http.StatusOK, `%PDF`
		case item.method == http.MethodGet:
			return http.StatusOK, `{"id":"media-1","url":"` + server.URL + `/file","mime_type":"application/pdf","sha256":"abc","file_size":4}`
		}
		return http.StatusOK, `{"success":true}`
	})
	client := clientOf(server.URL)
	ctx := context.Background()

	id, err := client.UploadMedia(ctx, `f"1.pdf`, "application/pdf", strings.NewReader("%PDF"))
	if err != nil || id != "media-1" {
		t.Fatalf("subir %v %v", id, err)
	}
	upload := calls[0]
	_, params, _ := mime.ParseMediaType(upload.ctype)
	reader := multipart.NewReader(strings.NewReader(string(upload.raw)), params["boundary"])
	form, err := reader.ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	if upload.path != "/v25.0/1055/media" || form.Value["messaging_product"][0] != "whatsapp" || form.Value["type"][0] != "application/pdf" || len(form.File["file"]) != 1 {
		t.Errorf("multipart %v", form.Value)
	}
	if form.File["file"][0].Filename != `f"1.pdf` {
		t.Errorf("nombre %q", form.File["file"][0].Filename)
	}

	info, err := client.GetMedia(ctx, "media-1")
	if err != nil || info.MimeType != "application/pdf" || info.FileSize != 4 || calls[1].path != "/v25.0/media-1?phone_number_id=1055" {
		t.Fatalf("consultar %+v %v %s", info, err, calls[1].path)
	}
	data, err := client.DownloadMedia(ctx, info.Url)
	if err != nil || string(data) != "%PDF" || calls[2].auth != "Bearer tok" {
		t.Errorf("descargar %q %v", data, err)
	}
	if err := client.DeleteMedia(ctx, "media-1"); err != nil || calls[3].method != http.MethodDelete {
		t.Errorf("borrar %v", err)
	}
	if _, err := client.GetMedia(ctx, ""); err == nil {
		t.Error("sin id")
	}
}

// webhookSample: un POST de Meta con un texto, un botón de plantilla, una respuesta de lista, una ubicación, un estado failed con
// callback y un error de la cuenta
const webhookSample = `{
  "object": "whatsapp_business_account",
  "entry": [{
    "id": "102290129340398",
    "changes": [{
      "field": "messages",
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {"display_phone_number": "15550783881", "phone_number_id": "106540352242922"},
        "contacts": [{"profile": {"name": "Sheena Nelson"}, "wa_id": "16505551234", "user_id": "US.1349"}],
        "messages": [
          {"from": "16505551234", "from_user_id": "US.1349", "id": "wamid.T", "timestamp": "1749416383", "type": "text", "text": {"body": "Hola"}},
          {"context": {"from": "15550783881", "id": "wamid.CTX"}, "from": "16505551234", "id": "wamid.B", "timestamp": "1749416384", "type": "button", "button": {"payload": "SI", "text": "Sí"}},
          {"from": "16505551234", "id": "wamid.I", "timestamp": "1749416385", "type": "interactive", "interactive": {"type": "list_reply", "list_reply": {"id": "row-2", "title": "Martes", "description": "Tarde"}}},
          {"from": "16505551234", "id": "wamid.L", "timestamp": "1749416386", "type": "location", "location": {"latitude": 4.6, "longitude": -74.08}}
        ],
        "statuses": [{
          "id": "wamid.S", "status": "failed", "timestamp": "1750263773", "recipient_id": "16505551234",
          "biz_opaque_callback_data": "{\"order\":\"A-1\"}",
          "pricing": {"billable": true, "category": "utility"},
          "errors": [{"code": 131047, "title": "Re-engagement message", "message": "Re-engagement message", "error_data": {"details": "more than 24 hours"}, "href": "/docs"}]
        }],
        "errors": [{"code": 131000, "title": "Something went wrong", "message": "Something went wrong", "error_data": {"details": "x"}}]
      }
    }, {"field": "account_update", "value": {}}]
  }]
}`

/**
* TestSignature: la firma válida pasa y las demás no
* @param t *testing.T
**/
func TestSignature(t *testing.T) {
	raw := []byte(webhookSample)
	signature := Signature(raw, "secreto")
	if !VerifySignature(raw, signature, "secreto") {
		t.Error("la firma válida no pasa")
	}
	for _, item := range []string{"", "sha256=", "sha256=zz", "sha1=" + signature[7:], Signature(raw, "otro")} {
		if VerifySignature(raw, item, "secreto") {
			t.Errorf("firma %q pasa", item)
		}
	}
	if VerifySignature(append(raw, ' '), signature, "secreto") || VerifySignature(raw, signature, "") {
		t.Error("cuerpo cambiado o sin secreto pasa")
	}
	if !VerifyTokenOk("tok", "tok") || VerifyTokenOk("tok", "") || VerifyTokenOk("", "") || VerifyTokenOk("a", "b") {
		t.Error("verify token")
	}
}

/**
* TestParseEvents: los eventos tipados de un POST de ejemplo
* @param t *testing.T
**/
func TestParseEvents(t *testing.T) {
	events, err := ParseEvents([]byte(webhookSample))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 6 {
		t.Fatalf("eventos %d: %+v", len(events), events)
	}
	for _, item := range events {
		if item.PhoneNumberId != "106540352242922" || item.WabaId != "102290129340398" || item.DisplayPhone != "15550783881" {
			t.Errorf("número %+v", item)
		}
	}
	text := events[0].Message
	if events[0].Kind != EventMessage || text.Text != "Hola" || text.From != "16505551234" || text.FromUserId != "US.1349" || text.Name != "Sheena Nelson" || text.Timestamp != 1749416383 {
		t.Errorf("texto %+v", text)
	}
	button := events[1].Message
	if button.Text != "Sí" || button.Reply == nil || button.Reply.Id != "SI" || button.Context.Str("id") != "wamid.CTX" {
		t.Errorf("botón %+v", button)
	}
	list := events[2].Message
	if list.Text != "Martes" || list.Reply.Type != "list_reply" || list.Reply.Id != "row-2" || list.Reply.Description != "Tarde" {
		t.Errorf("lista %+v", list)
	}
	if location := events[3].Message; location.Type != "location" || location.Content.Num("latitude") != 4.6 {
		t.Errorf("ubicación %+v", location)
	}
	status := events[4].Status
	if events[4].Kind != EventStatus || status.Status != "failed" || status.CallbackData != `{"order":"A-1"}` || len(status.Errors) != 1 ||
		status.Errors[0].Code != CodeReengagement || status.Errors[0].Details != "more than 24 hours" || !status.Pricing.Bool("billable") {
		t.Errorf("estado %+v", status)
	}
	if events[5].Kind != EventError || events[5].Error.Code != 131000 {
		t.Errorf("error %+v", events[5])
	}
	if _, err := ParseEvents([]byte(`{"object":"page","entry":[]}`)); err == nil {
		t.Error("otro objeto")
	}
	if _, err := ParseEvents([]byte(`{`)); err == nil {
		t.Error("JSON inválido")
	}
}

/**
* TestHandler: VerifyHandler y el POST firmado, sin firma, inválido y demasiado grande
* @param t *testing.T
**/
func TestHandler(t *testing.T) {
	received := []Event{}
	handler := Handler("verify", "secreto", func(events []Event) { received = append(received, events...) })

	get := func(query string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest(http.MethodGet, "/hook?"+query, nil))
		return w
	}
	if w := get("hub.mode=subscribe&hub.verify_token=verify&hub.challenge=123"); w.Code != 200 || w.Body.String() != "123" {
		t.Errorf("challenge %d %q", w.Code, w.Body.String())
	}
	if w := get("hub.mode=subscribe&hub.verify_token=otro&hub.challenge=123"); w.Code != 403 {
		t.Errorf("token malo %d", w.Code)
	}
	if w := get("hub.mode=unsubscribe&hub.verify_token=verify&hub.challenge=123"); w.Code != 403 {
		t.Errorf("modo malo %d", w.Code)
	}

	post := func(raw []byte, signature string) int {
		r := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(string(raw)))
		if signature != "" {
			r.Header.Set(SignatureHeader, signature)
		}
		w := httptest.NewRecorder()
		handler(w, r)
		return w.Code
	}
	raw := []byte(webhookSample)
	if code := post(raw, Signature(raw, "secreto")); code != 200 || len(received) != 6 {
		t.Errorf("firmado %d, %d eventos", code, len(received))
	}
	if code := post(raw, ""); code != 401 {
		t.Errorf("sin firma %d", code)
	}
	if code := post(raw, Signature(raw, "otro")); code != 401 {
		t.Errorf("firma mala %d", code)
	}
	bad := []byte(`{"object":"page"}`)
	if code := post(bad, Signature(bad, "secreto")); code != 400 {
		t.Errorf("objeto inválido %d", code)
	}
	big := []byte(strings.Repeat(" ", MaxWebhookBody+1))
	if code := post(big, Signature(big, "secreto")); code != 413 {
		t.Errorf("grande %d", code)
	}
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodPut, "/hook", nil))
	if w.Code != 405 {
		t.Errorf("PUT %d", w.Code)
	}

	r := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(webhookSample))
	if _, err := ParseWebhook(r, "secreto"); !errors.Is(err, ErrSignature) {
		t.Errorf("ParseWebhook sin firma %v", err)
	}
}
