# Revisión jsql/drivers (vet/staticcheck/gofmt limpios; nada exportado de drivers se usa fuera de et; q=query.go c=command.go)

## Comportamiento (bugs reales, arreglo chico)
4. pg q:745-768 y sqlite q:675-696 emiten ORDER BY/LIMIT en COUNT(*) (sin `if query.IsCount { return }` que sí tienen mysql q:702, mssql q:770, oracle q:667): `Where().OrderBy("name").Count()` en postgres → error GROUP BY. + guardia `!query.IsExists` en GROUP BY/HAVING (mysql q:683,695). ~8 líneas c/u + caso en TestCatalog §4.
5. oracle q:379-388 IS/IS_NOT con valor usa =/!= (no null-safe); los otros 4 sí → `DECODE(a, b, 1, 0) = 1` / `= 0`. 2 líneas + comentario.
6. postgres connect.go:20-37 defaultChain y :44-61 chain son iguales salvo default de user ("" vs "postgres"); DSN sin escapar (contraseña con @ / # rompe); josefina/driver.go:37-53 ya escapa → una `chain(params, database)` con escape. −18.
2. Agregados pg: pgAggExpr q:413-440 (COALESCE(SUM),0, casts) vs pgQuoteAggregate quoted.go:172-221 (sin COALESCE); igual sqlite q:142-163 vs quoted.go:186-235; mysql/mssql/oracle tienen un único xAggregate(fn, expr) quoted.go:303 → pgAggregate/sqliteAggregate únicos; SUM de un et.AGGREGATE gana COALESCE en pg/sqlite (como los otros 3). −25 c/u; TestCatalog §4/5.
9. pg q:290-301, sqlite q:263-274 BETWEEN con jsql.Quoted del núcleo y 2 case (bool→true no 1; time no pasa por pgQuoteTime) → un case BETWEEN,NOT_BETWEEN con op y Quoted(et.NewValue()) del driver como mysql q:336-345; igual pgInValues q:146/sqliteInValues q:125. ~10 c/u.

## Consistencia entre drivers (privados, paralelismo)
1. Tabla: mysql/mssql `xColumnRefValue` → `xJoinColumnRef` (cuerpo idéntico a pg/sqlite/oracle); param `refs` → `isJoin` en my/msCondExpr y Conds; sqlite: mover sqliteCondsSQL/sqliteJoinCondsSQL debajo de sqliteConds (orden de pg); mysql/mssql: añadir masterRows (oracle q:711, 6 líneas) y usarlo en SelectExpr (inline q:472, q:577); pg pgAggExpr switch sobre jsql.RollupOperation → switch sobre et.COUNT como los otros 4 (q:427-437); sqlite FieldValueRef devuelve 2 valores (q:194) vs 3 en los otros (dejar, anotar).
11. oracle q:671-694 ORDER BY inline → `oraOrderExpr` copiado de myOrderExpr (mysql q:621-634). 0 netas.
12. pg q:509-581 / sqlite q:391-463 relaciones en SelectExpr con 5 bloques `if From == nil/Model == nil` (~72 líneas; `fld.From == nil` inalcanzable tras q:453/351) vs mysql q:461-489 (~28) → forma compacta. −44 c/u. Opcional.
10. findField muerto: pg q:130-139, sqlite q:109-118, oracle q:156-161 (query nunca nil) → query.GetField directo (llamadas pg q:449, sqlite q:347,571, oracle q:483). −26.
15. postgres/condition.go:98 pgValueCast (otro archivo/nombre/tipo que my/ms/oraValueType en query.go) → mover a query.go junto a pgFieldValueRef + doc; nota en sqliteCondExpr de que no tiene equivalente. 0 netas.
16. Quoted default pg quoted.go:59 / sqlite :65 usan jsql.Quoted; my/ms/ora xQuoteAny ([]byte → X'…', log) → pgQuoteAny (bytea '\x%x') / sqliteQuoteAny (X'%x'). ~22 c/u. Opcional.
8. Command sin modelo: mysql c:257 y mssql c:257 error "command without model"; pg c:282,330, sqlite c:235,283, oracle c:210,250 lo soportan → documentar en spec §9.2/9.3 (2 líneas).

## Helpers idénticos byte a byte (md5) → una vez en jsql (catalog.go sección Utilidades)
3. xNestSource 5/5 (22×5); arrayElemType 5/5 (18×5); connectWithRetry 4/4 pg/my/ms/ora (24×4) → jsql.ConnectWithRetry(ctx, open func() (*sql.DB, error), maxRetries); isJsonType 3/3 my/ms/ora (pg/sqlite lista inline en types.go); timeLayouts 3/3; columnType(model, name) my/ms/ora load.go:30,:30,:56 → Model.ColumnType(name) (1 línea con GetColumn); masterRows 3/3; pgColsVals=sqliteColsVals (~40), pgPKWhere=sqlitePKWhere (10), pgJoinColumnRef=sqliteJoinColumnRef. Neto ≈ −150.

## Mensajes
7. "database is required" ×7 (pg connect.go:158,201; mysql :133,165; mssql :132,164; sqlite :26), "service_name is required" ×2 (oracle :99,136) → jsql.MSG_ATRIB_REQUIRED (ya usado 3 líneas abajo). 9 líneas. Tres textos para "refusing to UPDATE without WHERE" (pg c:360, sqlite c:313 dice New pero mira Old y New, my/ms/ora otro); "query has no FROM source" ×5; "no columns to insert into %s" ×5; "unsupported command type: %s" ×5 → 5 MSG_* en jsql/msg.go (MSG_QUERY_NO_FROM, MSG_NO_COLUMNS_INSERT, MSG_NO_COLUMNS_UPDATE, MSG_REFUSE_WITHOUT_WHERE, MSG_UNSUPPORTED_COMMAND). ~27.

## Comentarios / idioma
13. Español en archivos en inglés: pg q:594,627; sqlite q:489,562-563; mysql q:557,579-580; mssql q:664; oracle q:545,574; sqlite connect.go:23, types.go:25; josefina/driver.go (8 bloques, todo en español); test/driver_test.go:116-117, createdb_test.go:50, catalog_test.go:603. ~35.
14. pg load.go:174 doc ExistModel con params viejos; :180 `ddlTable(model)` resultado descartado (sentencia muerta); pg q:588-589 pgSelects @return string (es []string); q:649 pgFrom @return strings.Builder (es []string); q:676-677 cierre `*/` en vez de `**/`; sqlite q:213 @param sin isJoin; pg load.go:36-43 ddlColumns narra "_idx BIGSERIAL" que no emite; pg quoted.go:15-20 y sqlite quoted.go:16-21 Quoted con 6 líneas narrativas → 1 como mysql.
17. sqlite/connect.go:7 import de et/et en el grupo stdlib → mover a :13.
25. sqlite/test.go:28-36 reimplementa connectTo (para no MkdirAll) → 1 línea de comentario.
18. Públicos sin uso externo (ExistDatabase/CreateDatabase/DropDatabase pg; QuotedAs; Quoted ×5): no tocar.

## Batería
19. test/wrapper_test.go:88-91 salta 4 motores con `continue` silencioso → t.Run + t.Skip con motivo. 3 líneas.
20. oracleDrops a mano en driver_test.go:152 (6) y catalog_test.go:246-247 (18) → oracleDropSchema con PL/SQL sobre USER_TABLES LIKE 'schema\_%' (como mysqlDropSchema/mssqlDropSchema). ~12. Mientras: 1 línea de doc.
21. createdb_test.go:21-28 findTarget devuelve target{} vacío si no existe → findTarget(t, name) con t.Fatalf. 4.
22. catalog_test.go:1853 writeSummary lista drivers a mano → derivar de catalogTargets(). 3.
23. test/localhost/ directorio vacío no versionado → borrar.
24. josefina/driver.go:37-67 duplica chain/connectTo de postgres (privados) → anotar; exportar sería API nueva.

## No recomendado ahora
Unificar FieldExpr/ScalarExpr y Froms ([]string vs string); capa común de condiciones; Command sin modelo en mysql/mssql; QuotedAs en pg/sqlite; exportar postgres.chain/connectTo; partir query.go.

## Conservar
Mismo juego de archivos + init(); Quoted con mismo orden de case ×5 y xConds idéntico; comentarios del "por qué" del motor (msFragment, oraBlock, myJsonSet, sqlite ExistModel); TimeoutContext+defer cancel y sanitize* por motor; batería (targets con skip, recorder, result/ con -update, mysqlDropSchema/mssqlDropSchema).
