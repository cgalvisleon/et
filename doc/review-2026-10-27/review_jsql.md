# Revisión jsql/ núcleo (vet/staticcheck limpios; wrapper.go y jsql.d.ts sincronizados hoy)

## Duplicados pequeños (interno)
1. define.go:471-614 seis defineBefore/After* idénticos (24 líneas c/u) → helper `setScript(list, name, code, version) []*jwf.Script`. 145→45.
2. define.go:203-287 defineIndex/definePrimaryKey/defineUnique/defineRequired idénticos; además append en sitio sin muDef → `addIndex(list, name) ([]*Index, *Index)` con slices.Clone (patrón withEntry). 85→35.
3. command.go insert (497-509, 529-541), update (573-585, 606-618), delete (649-661, 686-692): 12 bucles de triggers → `runTriggerFuncs(tx, fns, old, new) error` y `runTriggerScripts(model, tx, scripts, old, new) (old, new, err)`. 90→30.
4. query.go:700-731 Query.and/or → `addToSection(cond, connector)`. 32→18.
5. query.go:1313-1396 existsTx/countTx casi idénticos y repiten allTx (1183-1203) → `renderSQL() (sql, isTest, err)` + `scalarTx(tx, key)`. Comentario de existsTx/ExistsTx (query.go:1310, catalog.go:989) es de otra función. 85→45.
6. model.go:484 idxColumn = indexColumn (único uso getColumn :495) → borrar; define.go:143 defineColumn reescribe literal de model.go:469 newColumn → usar newColumn. ~15.
7. db.go:485-662 define: bucle DefIndex→[]*Index 4 veces (525-551) + 2 (584-596); bloque rollup 2 veces (602-622, 637-657) → `toIndexes(defs)` y `defineRollupDef(model, def)`. 178→120 sin partir.
19. relation.go whereToJson (160-168) = filtro de rollupWhere (175-183); Rollups.MarshalJSON (relation.go:224) y QueryRollups.MarshalJSON (query.go:148) mismo literal → `rollupJsonOf`. ~10.
20. series.go:100,120,174 "%08d" → const defaultSerieFormat.
16. helpers.go:158,175 argWhitAs/argWhitSchema compilan regex por llamada → vars de paquete (como `placeholder` :42, query.go:861 fromIdent).
17. define.go:650,663 defineTenantModel/defineProjectModel llaman DefineSource tras DefineModel que ya lo hizo (625); defineModel (635-639) error siempre nil → quitar 2 líneas + comentario.

## Muerto
8. relation.go:328-346 TypeJoin, JOIN/LEFT/RIGHT/FULL, Joins: público 0 usos; duplica JoinType de query.go:75-92 → Deprecated (o borrar).
9. Model.isChanged (model.go:67,212,252), DB.isChanged (db.go:30,157) solo se asignan; EVENT_MODEL_SET db.go:988 sin uso; Calc.Module query.go:286 siempre "" (drivers) y leído vacío en setCalc :1154; ConnectParams.ID/IsDebug (connection.go:749,756) llenados por loadTo (799,805) y toConnectParams (wrapper.go:257) pero newDB (db.go:45-77) no los lee: id de connection.Str("id"), debug de envar DEBUG (db.go:74) → **DB_IS_DEBUG (connection.go:797, CLAUDE.md, spec §10) no tiene efecto; el que funciona es DEBUG**. Decidir: newDB usa params.IsDebug || DEBUG y params.ID (2 líneas, cambio mínimo de comportamiento) o borrar campos + doc. 9 MSG_* sin uso (msg.go: CATALOG_NOT_FOUND, DB_NOT_FOUND, DB_STORE_IS_NIL, DEAD_DB_NOT_CONFIGURED, HISTORY_DB_NOT_CONFIGURED, INSTANCE_REQUIRED_ID, PARAMS_IS_EMPTY, RECORD_NOT_FOUND, STORE_IS_NIL); MSG_BRIDGE_MODEL_REQUIRED (:28) sin traducción es en init().

## Comentarios desfasados (solo comentarios)
10. connection.go:646 DriverJosefina "pending B5" (cerrada); getConnection (:19-39) sin case josefina → GetConnection("josefina")/DB_DRIVER=josefina dan MSG_UNSUPPORTED_DRIVER → documentar que usa PgConection o añadir case (3 líneas). spec.md:41,:289 dicen jrex (es jwf). CLAUDE.md "_idx is reg.UUID()" vs model.go:229 getIdx = UnixMilli texto. Comentarios con params inexistentes: newDB/NewDB (db.go:42, catalog.go:32); existModel db.go:274; connectTo/ConnectTo (connection.go:761, catalog.go:143); loadTo/LoadTo "Returns existing DB" (connection.go:779, catalog.go:151); init/Init "UseCore" (db.go:192, catalog.go:66); define/Define "delegates to DefineModel" (db.go:481, catalog.go:217); calc/Calc "DETAIL clause" (query.go:573, catalog.go:790); Command.setDebug "@return *Query" (command.go:141); Command.whereByJson/WhereByJson con comentario de where (command.go:187, catalog.go:1225); getDb/Db "*sql.DB" (model.go:430, catalog.go:1819); loadModel/LoadModel "from catalog" (db.go:817, catalog.go:1870,1879); DefineCalc "bytecode" (define.go:454, catalog.go:444); QueryTx "sql ry.Json" catalog.go:134; "ConnectionFromJso" connection.go:569; nombres en minúscula en exportados de connection.go (178/272/365, 254/346, 280/373).
14. Solo nombre sin descripción: catalog.go:40 LoadDb, 453 DetailKeys, 460 RollupKeys, 1592-1663 series (DefineSeries, NewSeries, SetSeries, GetSeries, DeleteSeries, GenSerie, GenValue), 1685 OnAuditLog, 1692 AddAuditLog, 1720 StatusList; series.go:16,70,85,114,134,149 (@param tipo/nombre invertidos; ownerId inexistente 134-135, 149-150); helpers.go:187 addAuditLog.
13. query.go:1496-1563 STAR, HasStar, hiddenIn, StarFields, SourceHiddens en español; exportados fuera del catálogo.

## Concurrencia
11. Invariantes en 7 sitios (model.go:79, define.go:135, :162, db.go:355, :161, relation.go:293, model.go:400). Imprecisiones: model.go:79 dice que todo es copy-on-write pero Indexes/PrimaryKeys/Unique/Required (h.2), Hiddens (define.go:293), OmitUpdates (120), ForeignKeys (241), Before*/After* (471-614) hacen append en sitio sin muDef; muIdx (model.go:78) protege getIdx (no toca estado) y Replace → renombrar muReplace y quitar de getIdx. Propuesta: bloque `Concurrency` sobre `type Model` (model.go:35) con lista explícita; corregir CLAUDE.md (_idx).

## Catálogo
12. Índice catalog.go:14-28 dice 200, hay 206 (§2 21/24, §3 19/21, §4 47/48, §6 17/18, §11 17/20); falta banner §1 (primer banner §2 en :186). Wrappers con lógica: Query.AddCondition (:830-833) y DB.LoadModel (:1883-1886) → privados. Exportados fuera del catálogo: Model.Replace (model.go:174; core-studio catalog_store.go:116), ToArrayAny (define.go:744; 8 usos), Bind (helpers.go:218; 4), TimeoutContext/Tester/TestConnection/Probe/ConnectionFromJson (connection.go), Wrapper/BindWrapper/ToJsMapWrapper, HasStar/StarFields/SourceHiddens. → mover wrappers o decir en encabezado que connection.go y wrapper.go exportan directo.
18. ~40 privados que llaman públicos (defineIdxField→DefineIndex define.go:188; defineModel 621-626; define→GetModel db.go:515,554,603; DB.sql→SqlTx db.go:477; Tx.query→Rollback db.go:978; isValid→IsAggregate relation.go:125; Detail.init→Init relation.go:60; allTx→SqlTx query.go:1205; all→AllTx 1235; Model.up→GetModel model.go:132). Mecánico, al tocar cada archivo.

## Erratas
15. ArgWhitAs/ArgWhitSchema (helpers.go:157,174; catalog.go:1789,1798; wrapper.go:449-450; jsql.d.ts) público 0 usos → ArgWithAs/ArgWithSchema + alias Deprecated. XConection: público con usos (core-studio connection.go:334-371 ×5, postick dbutil/open.go:47) → no ahora. `deFault` param (define.go:134,203,222,256,275,302,311; catalog.go; wrapper.go:565) → defaultValue. jquery.go:234 func libre toJson → jsonOf. query.go:782 page alias de setPage (772) → uno.
22. SqliteConection (connection.go:201-292) no embebe Pool (lee 4 campos a mano en ConnectionFromJson 603-606) y GetParams (257-269) sin "id" → newDB (db.go:64) da UUID nuevo por sqlite en vez de db:sqlite:<file>. Añadir "id" (aditivo).

## Ubicación
21. command.go:427-473 withoutOmitUpdates/isOmitUpdate/restoreOmitUpdates (métodos de Model) → model.go. query.go:119-282 QueryDetail/QueryRollups + getQuery/resolveWhere/rollupAggSelect → relation.go. model.go:30 Index → define.go. Tabla archivo→contenido solo en CLAUDE.md → comentario de paquete en catalog.go + spec §2.

## Escalabilidad (inventario)
23. (a) TypeColumn nuevo ~18 sitios (define.go:697-705, Model model.go:35-80, newModel db.go:770-808, up model.go:108-160, Replace 174-223, toJson 270-306, Define define.go:14-81, DB.define, Define.UnmarshalJSON, Query query.go:292-320/:327/:577-652/:1107-1120, relation.go:565-585, wrapModel wrapper.go:530-707, jsql.d.ts:48, 5 drivers query.go, spec §2.1, TestCatalog). (b) columna automática ~10 sitios (define.go:669-682, defineIdxField 186-196, Define.IdxField :68 + db.go:514,598, Model.IdxField model.go:43 + Replace 188 + toJson 284 + defineColumn 152-156, catalog.go:237, wrapper.go:564, d.ts, spec §2.2, drivers). (c) driver nuevo: connection.go const 640-647, getConnection 19-39, XConection ~90 líneas, ConnectionFromJson 587-635, drivers/<x>/8 archivos, driver_test.go:119, d.ts:199, wrapper.go:263, spec §9, core-studio connection.go:334-371. `baseConnection{Id}` embebido quitaría 10 métodos (~40 líneas).
24. Test `TestWrapperDeclared` en jsql_test.go: claves "nombre": de wrapper.go presentes en jsql.d.ts. ~40 líneas, sin base.

## No recomendado ahora
25. command.go insert/update/delete pipeline unificado; DB.define partir; loadQuery partir; setRollup/setDetail/setMaster; wrapModel/wrapPackage partir; ConnectionFromJson → fromJson por tipo; Model.Replace *s=*other; Query vs Command sqlBase embebido (cambia json tags y result/*.json); PgConection→PgConnection (usos).

## Conservar
catalog.go + TestCatalog por sección; withEntry+muDef y schemaList/modelList; helpers de lotes en relation.go probados sin base; parser único (normalizeQuery/queryAliases, parseFromRef/modelOf, rollupJson); Query.err/Command.err + getTx/execTx con recover.
