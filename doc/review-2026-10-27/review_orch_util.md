# Revisión orquestación (jwf, validator, crontab, resilience, queue, ia, jia, stores, dt, mem, ephemeral, service)
Consumidores: jwf→core-studio, josefina; validator→core-studio; crontab→core-studio; ia→core-studio, cmd/ia; resilience→solo jwf; dt→solo stores. SIN consumidor: jia, stores, mem, ephemeral, service, queue. staticcheck: 17 U1000, 1 S1011, 4 ST1001.

## Bugs con arreglo chico
1. resilience/instance.go:224-244 Run() programa AfterFunc y devuelve de inmediato Result vacío; jwf/instance.go:770-782 runResilence siempre MSG_RESILIENCE_NO_RESULT y 30 s después re-ejecuta el paso en goroutine (recursivo :238). core-studio expone total_attempts. → Run síncrono con bucle+Sleep; runAttempt (:181) vaciar s.Result antes de fn.Call. ~15; test temporal.
2. jwf/instance.go:674,677,679,683 fmt.Errorf con MSG sin %s (msg.go:18,26,27) y ALREADY_RUNNING recibe id en vez de status. jia/ia.go:497,532 igual (MSG_AGENT_NOT_FOUND). ~6.
3. validator.go:374 MustCompile por validación (panic con Pattern("(") desde Go); :398-401,414-417 min/max solo > 0 → campos re/patternErr/hasMin/hasMax; compilar en Pattern/ByJson; ToJson según hasMin/hasMax; actualizar CLAUDE.md. ~20.
4. ia/store.go:112-118 LoadIndex no asigna embedFn/answerFn; document.go:100, conversation.go:71,99 los llaman → panic. → guardia MSG_API_KEY_REQUIRED en ingest y Ask. ~8.
5. jwf/wrapper.go:108 bindings["Json"] = et.Json{} compartido entre corridas (Json.x=1 persiste; carrera) → Binder que cree et.Json{} por runtime. 3.
6. dt/handler.go:60-66 Get devuelve nil con error Redis (stores/instances.go:135, authorization.go:156, form_state.go:142 hacen .Ok → nil pointer); dt/object.go:114-120,222-228 Byte()/Json() fallan tras JSON ([]byte→string base64, Json→map). → newObject(key) en error; case string base64; case map[string]any. ~12.
7. jia Store Get/Delete invertidos: ia.go:90, conversation.go:89, message.go:192; plural/singular: agent.go:250/274, conversation.go:215/239, participant.go:192/242, ia.go:205/229 → constantes store* en orden (collection, id). ~8.
8. jia otros: skill.go:39-50 ValidStr sin `!` (4); conversation.go:167-169 bucle duplica mensajes (S1011); conversation.go:128 LastMessage nil panic; agent.go:372 Sprintf(contextStr, prompt) sin %s; agent.go:366 error ignorado; router.go:27,172,183 ruta sin {tag} y args corridos (ia.go:519); ia.go:109 DEBUG default true.
9. resilience/resilience.go:143-164 readInstance devuelve nil,false tras Unmarshal ok (:163) → LoadInstance:188 y Stop:213 rotos; clave Redis id pelado (instance.go:130, resilience.go:148) → instanceKey(id) "resilience:instance:". ~5.
10. stores/authorization.go:22 ErrorSetAuthor = errors.New(MSG_RECORD_NOT_FOUND) vs trigger :80 errors.New(MSG_RECORD_EXISTS) → errors.Is (:142) nunca → var ErrRecordExists en msg.go, usar en authorization.go:80, dbs.go:71. ~4.
11. ephemeral/ephemeral.go:38-42,69-77 un solo timer para todas las claves, asignado en goroutine sin lock → timers map[string]*time.Timer bajo lock. ~15.
12. crontab/event.go:11-27 EVENT_CRONTAB_* vars globales reescritas por instancia (dos New(tag) se pisan) → documentar en eventInit; crontab.go:149,170 "job not found" → MSG_JOB_NOT_FOUND; stopJob(tag) indexa por id → renombrar param; crontab.go:110, job.go:203 Sprintf("job:%s") → channelOf. ~10.
23. service/storage.go:72-73 escribe bucket/ relativo pero Delete:154 usa s.Path → path.Join(s.Path, …) (2); storages (:28) y senders (send.go:61) sin mutex (8); send.go:89-91 `if !valid { return nil, err }` devuelve nil,nil.
25. jwf/flow.go:226-247 ToJson omite app_id, save_instances, tenant; instance.go:286-319 ref() omite flow_tag, name, duration que LoadInstance lee (workflow.go:241) → añadir. 3.

## Duplicados
13. addAuditLog ×6: jwf/flow.go:184-207, workflow.go:102-120, crontab/job.go:134-151, jia/ia.go:145-162, agent.go:187-204, participant.go:132-149 → appendAudit(log, userId, action) privado por paquete; Flow.addAuditLog:201-206 y WorkFlow.push:89-96 → notify(fns, data). −60.
14. jwf "instance:<id>" a mano ×5: workflow.go:226,285,320,331, instance.go:559 → instanceKey/stopKey con comentario de TTL. ~8.
15. ia/index.go:86 scope() no usado en document.go:151-153, :165-168, :175-178, conversation.go:76-78, :147-150; DeleteDocument (document.go:163-181) sin transacción → deleteDocuments. ~10.
16. validator.go:494-523 validateArrayString/Ints/Floats/ArrayAny → validateArrayLength directo (−25); mem/mem.go:59-63,72-76,393-397,414-418 bloque timer ×4 → arm() (−15); stores/instances.go:105-115 re-registra triggers (−11); jwf/flow.go:484-487 code sin uso (−4); jwf/flow.go:16,53 MANUAL = TRIGGER_MANUAL; MANUAL/WEBHOOK/CRON/SCHEDULE sin uso.

## Erratas
17. Públicas (alias+Deprecated): jwf/step.go:27 ErrrStepNotFound, flow.go:23-24 ErrrFlowNotFound/ErrrTriggerNotFound, instance.go:68 ErrorInstanceNotFound → ErrStepNotFound/ErrFlowNotFound/ErrTriggerNotFound/ErrInstanceNotFound (8); service/send.go:31-32 TypeComercial/TypeAutentication → TypeCommercial/TypeAuthentication; jwf/msg.go:23 MSG_STEPER_NOT_FOUND Deprecated (tag JSON steper_id NO tocar); mem/sync.go:10-72 Peticiones/Ejecucion/GetPeticiones (solo documentar).
18. Privadas: jwf/instance.go:653,688,692,707,723,742,756,763,770 runing→running, runResilence→runResilience, resilence→attempt; jia/agent.go:91,132 loadAgend→loadAgent; jwf/instance.go:131,205 Instance.hostname nunca leído.

## Patrón / hermanos
19. jwf/step.go:123 Step.Up exportado vs Flow.up privados → comentar "needed after json.Unmarshal"; crontab.Store (crontab.go:24) ≡ jia.Store (ia.go:25); stores.Instance.Get devuelve error si no existe (instances.go:152) vs FormState.Get false,nil → alinear (2); mem/dt/ephemeral/cache: documentar cuándo usar cuál; jwf/step.go:28 KindList incompleta y sin uso; dt/handler.go:16 Up por valor vs Get puntero.

## Comentarios
20. Sin comentario: jwf/step.go:105 LoadStep, wrapper.go:24 GetArgs, validator.go:47 New, resilience.go:58 ToJson, mem/mem.go:23 Load, :454 Empty, jia/router.go:22 LoadRouter, service/send.go:35-41, jwf/instance.go:22 Status.Str. Copiados de otra función: jwf/workflow.go:38,:71,:297; instance.go:283,:493,:513,:531,:788; wrapper.go:53,63,201,249,:359,:381; step.go:120; flow.go:282,:298; validator.go:63,:79,:87,:151,:161; resilience.go:178, instance.go:247,:151; jia/participant.go:153,166, ia.go:80; stores/dbs.go:11; service/storage.go:149; mem/mem.go:118,:135; ia/loader.go:72.
21. msg.go: jwf/msg.go:47 es pierde %v; :42 igual en ambos; jia/msg.go:8 "agente actualizado" en en; es sin traducir :26-31; MSG_ATRIB_REQUIRED duplica msg.; service/msg.go en = es (español). Sin uso (agrupar Deprecated): jwf MSG_STEP_IS_FUNCTION, MSG_WORKFLOW_STORE_IS_NIL, MSG_WORKFLOW_NOT_FOUND, MSG_STEP_DEFINITION_IS_UNKNOWN/_IS_NIL, MSG_STEP_STATUS_INVALID, MSG_FLOW_CONNECTION_NOT_FOUND, MSG_INSTANCE_CONNECTION_NOT_FOUND, MSG_INSTANCE_INVALID_CONNECTION/_TRIGGER, MSG_INSTANCE_TRIGGER_NOT_FOUND, MSG_STEPER_NOT_FOUND, MSG_STEP_CODE_INDEX_NOT_FOUND, MSG_INSTANCE_STOP, MSG_INVALID_SOURCE, MSG_STEP_ON_PUBLISH_*, MSG_MODEL_NOT_FOUND; crontab MSG_JOB_STORE_IS_NIL; resilience MSG_RESILIENCE_NOT_INITIALIZED, MSG_INSTANCE_STOPPED, MSG_RESILIENCE_FINISHED_ERROR; stores MSG_RECORD_NOT_CREATED, MSG_RECORD_IS_NOT_ITEM, EVENT_ADD_AUTHORIZATION, EVENT_DEL_CONFIG, EVENT_SET_CONFIG; ia MSG_SOURCE_REQUIRED, MSG_DOCUMENT_CREATED; jia MSG_SKILL_HTTP_NOT_SUPPORTED, MSG_AGENT_ALREADY_EXISTS, EVENT_*_SET; validator MSG_VALIDATOR_BETWEEN.

## Muerto
22. U1000 jwf (seguro quitar): flow.go:95 Flow.isDebug, instance.go:121-122 isDebug/isChanged, script.go:15 Script.vm, step.go:62-63, workflow.go:30 isInitial; también Flow.isChanged (flow.go:96/200), WorkFlow.isDebug/isChanged (workflow.go:31-32/75/118). crontab.go:36 running; jia Ia.save (ia.go:194 — nada persiste el Ia), Ia.delete :224, loadConversation conversation.go:87, Conversation.save :204, Message.delete message.go:190, loadParticipant participant.go:69, eventInit event.go:16, Conversation.isChanged. ST1001 dot imports stores/{authorization,dbs,form_state,instances}.go → jsql. (~50 tokens). Públicos sin uso (Deprecated): jwf.GetArgs, KindList, StepStatusList, FlowStatusList, FlowDefinition, Node, ErrorInstanceNotFound; stores.Dbs.dbs; service/service.go (init rehace mapas → borrar); service.Storage no la implementa LocalStorage; jia.Sender nunca se asigna.

## Invariantes (doc.go por paquete)
24. Claves Redis: jwf instance:<id> (sin TTL; core-studio no pasa por WorkFlow), instance:<id>:stop (TimeAwait 10 min); resilience <id> sin prefijo y resilience:<id>:stop (Interval); dt object:<key> (CACHE_DURATION 5 min, solo PRODUCTION=true); stores vía dt instance:/form_state:/<tenant>:<profile>:<method>:<path>; crontab sin Redis; service OTP clave del llamador. Mutex: jwf.Instance.mu protege Status/Step/… pero next() (:572-614) e isStop() (:551) sin lock → documentar no concurrente; crontab Job.mu durante save() I/O (job.go:185-193), trigger Attempts++ fuera; jia addAuditLog bajo cualquier mutex; resilience.Instance sin lock. Persistencia: jwf nada (OnChange); crontab Store.Set("jobs") por setStatus; jia solo Agent/Participant/Message.

## No recomendado
Unificar ia.ingest con IngestText; jia persistencia completa; steper_id; resilience con contexto; mem decode[T]; crontab EVENT_* a campos; service Storage vs LocalStorage; validator orden de errores; partir jwf/instance.go (815); eliminar paquetes sin consumidor.

## Conservar
queue/queue.go; ia/index.go; validator/wrapper.go, jwf/script.go, console.go; crontab/handler.go (invariante escrita); mem RWMutex+timers.

# Revisión utilidades, mensajería, create, cmd/* (vet limpio; staticcheck 12)
## Bugs
1. utility/valid.go:26,37,65,76 Contains(val, notIn) pasa la lista como un solo elemento → ValidStr/ValidIn/ValidInt/ValidNum nunca excluyen; ettp/v1/server-proxy.go:134,249,237,241 dejan pasar "new"/"-1"/puertos. 48 usos consumidores + 65 et casi todos []string{""} (sin cambio para ellos) → slices.Contains(notIn, val). 4 líneas.
2. create/template/templates.go: :328 jsql.Def→jsql.Define; :467 `return $2.Query(query)` → `.All()`; :676 ModelhRpc redeclara PackageName (quitar); :946 DB_TENANT_ID. Verificado compilando scaffold. 4.
3. jwsp/handler.go:369-378 SendReplyVideoMessageByURL (to, url, caption) MessageID: url → (to, messageID, url, caption). 0 usos. 2.
4. jwsp/message.go:1100 "tumbnail_product_retailer_id" → thumbnail. 1.
5. envar/envar.go:36 `if store != nil { return }` invertido → == nil. 1.
6. utility/utility.go:110,117 More: count[tag]=0 → n; expiration*time.Second sobre Duration; mapas sin mutex. 0 usos. 6 o borrar.
7. utility/list.go:21 slices.Delete(*s, idx, 1) → (idx, idx+1). 0 usos.
8. cmds/step.go:98-128 RunSSH copia de RunOS; cmds.go:17 Load nil,nil → RunSSH→RunOS Deprecated; Load error; @parms→@param.
15. stdrout/stdrout.go:42-62,66-78 Windows: mapa colors construido antes del blanqueo en init → construir en init; :194,:236 Traces excluyen "ErrorM"/"ErrorF" pero son Errorm/Errorf; logs.go:66 Alertm if siempre cierto. ~6.
16. envar/get.go:29-34 cachea con ToUpper pero busca sin → mover ToUpper antes (1); arg.go:16 compara con ToLower(name).
17. utility/crypt.go:77,108 CFB SA1019; DecryptAES:102 `"", err` con err nil; 0 usos → AES-GCM (~15) o borrar; EncryptMD5 ≡ Md5 (hash.go:58), HashSHA256 ≡ Sha256 (hash.go:48, 4 usos) → Deprecated.
24. file/file.go:141,174 MakeFile devuelve path (carpeta) en vez de pathFile; :185 Remove registra "remove file:" cuando NO existe. 3.
25. create/progress.go:29 bar.Add(100/total*current) acumula → bar.Add(100/total); hCommand.go:12,49 CmdProject.Use dice "micro". 2.
23. cmd/install/main.go:11-25 instala deps que et ya no tiene (jdb, go-nanoid) → dejar installContext() (−30); cmd/create/main.go:14 ignora error de modTidy.
14c. cmd/daemon es `package daemon` no main (go run no arranca); apps nil map (main.go:50-54) → borrar o arreglar; CLAUDE.md lo lista como binario.

## Duplicados
9. {{key}} ×5: aws/sms.go:60-64, brevo/email.go:55-59, brevo/sms.go:67-72, infobip/email.go:37-41, infobip/sms.go:77-81 → strs.Parse(content, params) (strs.go:423). −20.
10. file.params (file.go:38) ≡ utility.Params (:339, 0 usos); file.joinSep (:54) ≡ strs.Append (:247) → usar strs.Append; xls.IndexOf (read.go:131) ≡ csv.IndexOf (:124) ≡ slices.Index (2 usos consumidores xls.IndexOf → wrapper); utility.ToSerialize/Serialize ≡ json.Marshal (0 usos); utility.UUID ≡ reg.UUID, GenId ≡ reg.GetUUID (12 usos et; reg importa utility → hacer que reg.UUID llame a utility.UUID, 1 línea); reg/id.go:95,108,121,161,174,187 literal map {"","*","new"} ×6 → isNew(id). ~8.
19. utility/utility.go:519,522,525 Normalize compila 3 regex por llamada (6 usos) → vars de paquete; ExtractMencion:221, valid.go:15 validate por ValidEmail. ~6.
18. strs/strs.go:216-232 Titlecase rama muerta (capitalize false) → quitar; comentario "uppercases the first rune"; 8 usos en create dependen del comportamiento.

## Erratas públicas (0 usos consumidores; alias+Deprecated)
11. utility.go:31 SELECt; :43 "bolean"; :219 ExtractMencion; :166 InInt(val string, in []string) ≡ InStr; file.go:70 GetExtencion (2 usos aws/s3.go:112) → GetExtension; strs.go:180,309,373 DaskSpace, ApendAny, RemoveAcents; reg/id.go:67 GenUUId → GenUUID; units.go:19 UnityMiligram; create/hMicroservice.go:45 MkMolue → MkModel (privado al uso); create/promps.go Promp* → prompts.go Prompt*; create/template/statefulset_tempalte.go + hDeployments.go:24 + deploy.go:10 tempalte; aws/sesion.go, sesion_test.go → session; cmd/whatcher → cmd/watcher; aws/msg.go:18 "clave de secret".

## Comentarios
12. race/race.go:181-236 bloques corridos una posición (IsNil, MapRange, ArrayRange, Range; Increase :237 sin); utility.go:74 NowTime, :531 Serialize; crypt.go:48 CryptoSHA512, :85 DecryptoAES; file.go:79 ExistPath @return bool (FileInfo); aws/s3.go:149,178 @return; xls/read.go:126 indexOf; logs.go:218 Pong "PoNG". ~12.
13. Sin comentario: jwsp/handler.go 33 de 40 Send*; jwsp/message.go ~15 tipos; create/*.go casi todo (MakeCmd, DeleteCmd, MakeDeployments, Command, MakeInternal*, Mk*, MakePkg, MakeModel, MakeRpc, MakeProject, MakeReadme, MakeEnv, MakeVersion, MakeDeploy, MakeScripts, MakeTest, MakeWWW, ProgressAdd/Next, Promp*; hMicroservice.go:76-78 bloque vacío); file/watcher.go:299 WatcherPath; utility/list.go; race.NewValue; cmd/daemon. ~100.
21. Español en archivos en inglés: file/file.go:204,217; watcher.go:237,275,305,328; utility.go:516-524, :442-453 (rutaArchivoGoMod, contenido, lineas, partes), :220-222 (patron, expresionRegular, mencions); pid.go:81; timezone.go:121-132 meses es; cmd/install/main.go:38,42,46; cmd/daemon/msg.go. ~25.

## Muerto (0 usos; borrar solo lo marcado * que además está roto)
14. utility: More*, UUIndex*, InInt, List*, Add, ToSerialize, Serialize, ToJson, ExtractMencion, GeneratePortNumber* (max 99999), TimeDifference, Quote, Unquote, ParamQuote, Params, Address, Num, DivNum, DivInt, GetPidByPort, BinarySearch, Dijkstra, MinDistance, QuickSort, Contains, InStr, GenKey, SetQuotedChar, FromBase64, EncryptAES/DecryptAES, EncryptMD5, HashSHA1, HashSHA512, 17 constantes utility.go:27-45. strs: Change, FormatSerie, Name, DaskSpace, RemoveAcents, ApendAny, AppendAny, StrToTime, StrToBool, HtmlToText, MaskToken, JoinQuoted, Same, Trim, NotSpace, GetSplitIndex, FormatUppCase, FormatLowCase, FormatDateTime. file.WatcherPath* (watcher.go:299-347; cmd/whatcher → NewWatcher(".").Debug().Load()). iterate* paquete (iterate.go:28 invertido, sin mutex). cmds*. timezone.FormatMDYYYY* (:115 layout Go en vez de clave), timezone.Add. reg: GenUUId, GenSnowflake, GenIndex, TagUUID/ULID/XID; init id.go:194-197 snowflake.Epoch sin uso → quitar dependencia. envar: Load*, SetNumber, GetConfig, Arg*, Str/Int/… cortos. logs: Ping, Pong*, Rpc, Debugger, Tracer, EnableCallerInfo; stdrout: Color, CW, ErrorTraces, SetStdout, IsTTY. create: MakeWWW, MakeProject, MakeVersion, MakeDeploy (+template.Version/Deploy 330 líneas bash), DeleteMicroservice/DeleteCmd (os.Remove sobre carpetas), Command (hExec.go:13 log.Fatal), MakeTest ignora name. jwsp/event.go 1 línea. cmd/daemon*.

## Hermanos
20. xls.Add(data, name, []Column) no variádica vs NewXls; xls/write.go:82 escribe Title sobre slice del llamador vs csv copia (:39) → copiar (2); envar SetNumber vs GetFloat → SetFloat + alias (3); logs.Warn(string) sin Warnf → añadir Warnf (4); brevo funciones vs infobip/aws struct+Params (anotar); aws/s3.go:83,87 ParseMultipartForm(2000) y "myFile" → consts (2); cmds/msg.go sin LANG.
22. staticcheck: cmd/daemon/main.go:51,53 U1000; cmd/resilience/main.go:42 ST1005; crypt.go:77,108 SA1019; utility.go:243,299,301,303,308,326,328,378 S1025 Sprintf("%s"); utility.go:24 importa x/exp/slices (list.go usa stdlib) → stdlib.

## No recomendado
Orden de params de 40 jwsp.Send*; units.Quantity.To tabla; partir utility.go; firmas ValidStr/xls.Add/logs.Warn/envar.Get; brevo a struct; stdrout io.Writer; Quote/Unquote; ModelhRpc+ModelRpc, deploy.sh rutas; file.Watcher.Load bloqueante.

## Conservar
xls/csv misma API; envar _mu con porqué, GetConfig copia, Set*ByArg; strs.cachedRegexp + Parse; file.Watcher nuevo; infobip/aws constructor con Params; scaffold de create a 4 líneas de compilar.
