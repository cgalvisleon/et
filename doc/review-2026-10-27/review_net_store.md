# Revisión tcp/, websocket/, store/ (vet/gofmt/staticcheck limpios en los 3; 5 avisos en test/*)
Contexto: store solo lo usa test/storebench; websocket solo test/websocket; josefina usa tcp básico (sin streams ni Hub.Metrics/SendTo/DownServer…). Comentarios de los 3 en español.

## Spec desfasada (solo doc)
1. store/spec.md:113 cita MSG_ID_IS_REQUIRED/MSG_INVALID_ID_LENGTH, :137 MSG_CORRUPTED_RECORD → son MSG_STORE_* (store.go:600, :70, segment.go:379). §7 paso 5 (:89) dice config tras reparar; se lee en newFileStore (store.go:977) antes. 3 líneas + versión.
3. tcp/spec.md:242, :419 y websocket/spec.md:150 dicen `internal/msg` → `et/msg`. 3 líneas.
4. tcp/spec.md:324-333 §14.2 describe campos de Hub inexistentes (mu/clients/servers); reales hub.go:62-80 clientMap/infoMap/connecting/serverMap/conns; mu solo mapas, beatMu, fnMu, timesMu. Orden de locks real: mu → Client.mu → clientInfo.mu / connState.timesMu; recvMu/sendMu/beatMu/fnMu no se anidan. Escribirlo en hub.go:57-61 y spec. ~10.
5. tcp/brecha.md:46 lista "recargar certificados" como fuera, pero se hizo (brecha.md:52); tcp/spec.md:281 §13 dice casos 22-24, son 22-26. 2 líneas.
6. websocket/spec.md:13 "Decisión: et/jws…" (jws eliminado); brecha.md:37-38 pasos de josefina; brecha.md:7 última corrida 2026-09-27 (historia dice 2026-10-21). 4 líneas.
21. Comentarios vs firma: websocket/message.go:79 Client.Send→Conn.Send; client.go:279 @return falta *session; server.go:190 @param channel→channelName; tcp/catalog.go:22 Client.ToJson omite tls/auth; store/store.go:127,:541 mencionan Prune (quitado). 6 líneas.

## Muerto / inconsistente
2. store/segment.go:430-443 `segment.read` inalcanzable (readPinned id=="" store.go:759-761; único llamador :673 siempre con id); spec v8 dice que se quitó. −20; bump versión.
7. msg/msg.go:34-51 y :181-198: 18 MSG_TCP_*/MSG_HUB_NOT_STARTED del jtcp eliminado, 0 usos → Deprecated (quitar en siguiente versión).
8. store/store.go:107 FileStore.ID nunca se asigna (solo toJson :154) → `ID: name` en newFileStore (:969) o Deprecated. 1 línea.
17. store/catalog.go:227 / store.go:181 `IsDebug()` es setter → añadir `Debug()`, IsDebug Deprecated; actualizar CLAUDE.md fila y README store. 8 líneas.
19. store/store.go:585 empty(): `defer os.RemoveAll; return nil` pierde el error → `return os.RemoveAll(s.Path)`.
18. store/store.go:125 `compacting int32` + atomic.CAS (:340,344) → atomic.Bool (regla counter[T]/atomic). 3 líneas.
15. tcp/security.go:334 errors.New("tls: no certificate…") único texto fuera de msg → MSG_TCP_TLS_NO_CERT en/es + fila spec §11 (tcp/spec.md:244-263). 4 líneas.
16. tcp/hub.go:343 compara err.Error() con MSG_TCP_NOT_CONNECTED/MSG_DATA_TOO_LARGE → sentinelas errNotConnected/errDataTooLarge en client.go:232, link.go:308,496 + errors.Is. 6 líneas.

## Duplicados pequeños
9. panic→error ×3 en tcp (link.go:254-262 callRequest; stream.go:455-462, 488-495) y websocket/handlers.go:27-34 `call`; tcp/link.go:122 safeCall → `call(fn func() error) error` en tcp; safeCall sobre él. −12.
10. websocket/conn.go:290-296 closeReason; tcp/link.go:472-478 inline en finish → extraer closeReason en tcp. ±0.
11. tcp/client.go:194-204 close = closeWith(nil) (:212-222). −8.
12. sort.Slice por Channel ×3: websocket/conn.go:245, :314, client.go:409 → sortSubs en message.go. −3.
13. store/store.go:999 y recover.go:49 "store not found" con dos textos → errNotFound(path). 5.
14. tcp/hub.go:647,:669 encodeMessage(m,1,nil) solo para validar → validMessage(m) en message.go, llamado por encodeMessage. +8/−2.

## Idioma / estilo
20. tcp/security.go:208-209, 386-388 en inglés; store/store.go:580-581 inglés; "(true if inserted…)" en store/catalog.go:46,55,64,89,98,107,127,137 y store.go:378,389,403,412,609,628,655,697,768; segment.go:358 "(at least 18 bytes)"; tcp/hub.go:18 usa // en archivo de bloques. ~15.
22. Tipos sin comentario: tcp Client (client.go:16), Server (server.go:16), StreamWriter/StreamReader (stream.go:41,45); websocket SubscriptionType (message.go:30); store FileStore (store.go:106), Mode/ReadOnly/ReadWrite (:99-104), WalEntry (wal.go:7), Op (notify.go:9), Active/Deleted (type.go:8-11), recordHeader/recordRef (segment.go:15,40). ~25.
23. store/catalog.go:74-76 tipo Deferred en catalog.go → store.go/commit.go; tcp/server.go:73 toJson llama peer.ToJson() → toJson(); websocket/client.go:158,:314 time.After en select → NewTimer+Stop. ~8.
24. staticcheck verificadores: test/tcp/test.go:1191 ST1005, :1559 ST1008; test/websocket/test.go:1069,:1230 ST1005, :1298 SA1019 parser.ParseDir deprecado (anotar o migrar a x/tools/go/packages).

## No recomendado ahora
NewClient/NewServer error siempre nil (firma); paquete común tcp/websocket; renombrar `recover` privada; partir store.go (1056); Hub.close no espera OnDisconnect; config websocket/tcp; hub.go:277-303 receive.

## Conservar
Catálogo verificado (caso 12 websocket, publicAPI tcp); spec+brecha con versión e historial; invariantes de store en un lugar y cumplidas; handlers+safeCall+duration compartido en tcp; kill/finish y eventos sin locks en websocket.
