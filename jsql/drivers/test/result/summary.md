# Resultado de TestCatalog

Generado por `go test -run TestCatalog` en `jsql/drivers/test`. El detalle de cada caso (salida y SQL) está en `<driver>.json` y `<driver>.sql`.

| Driver | Pasan | Fallan | Brechas | Omitidos |
|---|---|---|---|---|
| sqlite | 74 | 0 | 1 | 1 |
| postgres | 74 | 0 | 1 | 1 |
| oracle | 74 | 0 | 1 | 1 |

## 1. Conexión y base de datos

| Caso | sqlite | postgres | oracle |
|---|---|---|---|
| Register + ConnectTo | ✅ | ✅ | ✅ |
| Init (idempotente) | ✅ | ✅ | ✅ |
| NewDB | ✅ | ✅ | ✅ |
| LoadDb (desde ToJson) | ✅ | ✅ | ✅ |
| Load / LoadTo | ➖ omitido | ➖ omitido | ➖ omitido |
| Sql | ✅ | ✅ | ✅ |
| SqlTx + Tx.Commit | ✅ | ✅ | ✅ |
| Connection.GetParams / SetDatabase / GetDatabase | ✅ | ✅ | ✅ |
| SetDebug / Debug | ✅ | ✅ | ✅ |
| DB.Query / DB.QueryTx (SQL libre en JSON) | ⚠️ brecha | ⚠️ brecha | ⚠️ brecha |

## 2. Definición de modelos (DDL)

| Caso | sqlite | postgres | oracle |
|---|---|---|---|
| Define (declarativo) | ✅ | ✅ | ✅ |
| DefineModel + DefineColumn / DefineAttrib / DefineUnique / DefineHidden | ✅ | ✅ | ✅ |
| DefineTenantModel / DefineProjectModel | ✅ | ✅ | ✅ |
| DefineRequired (rechaza el insert sin el campo) | ✅ | ✅ | ✅ |
| DefineForeignKeys (rechaza un hijo sin padre) | ✅ | ✅ | ✅ |
| Stricted (ignora campos desconocidos) | ✅ | ✅ | ✅ |
| GetModel / RemoveModel / NewModel | ✅ | ✅ | ✅ |
| Insert (datos base) | ✅ | ✅ | ✅ |

## 3. Relaciones y campos calculados

| Caso | sqlite | postgres | oracle |
|---|---|---|---|
| Detail en el select (DefineDetail) | ✅ | ✅ | ✅ |
| Model.Detail + Query.Detail | ✅ | ✅ | ✅ |
| Master en el select (DefineMaster) | ✅ | ✅ | ✅ |
| Master 1 a 1 (rows = 1) | ✅ | ✅ | ✅ |
| Model.Master + Model.Bridge | ✅ | ✅ | ✅ |
| DefineRollup row (tp_doc → title) | ✅ | ✅ | ✅ |
| DefineRollup (count, sum, object) | ✅ | ✅ | ✅ |
| DefineCalcFunc + Model.Calc | ✅ | ✅ | ✅ |
| DefineCalc (script JS) | ✅ | ✅ | ✅ |

## 4. Consultas

| Caso | sqlite | postgres | oracle |
|---|---|---|---|
| Select + Where + OrderBy + All | ✅ | ✅ | ✅ |
| One / First / Count / Exists | ✅ | ✅ | ✅ |
| Limit (paginación) / Page | ✅ | ✅ | ✅ |
| Hidden (campo oculto en la consulta y en el modelo) | ✅ | ✅ | ✅ |
| Join (fluido) | ✅ | ✅ | ✅ |
| LeftJoin + GroupBy | ✅ | ✅ | ✅ |
| GroupBy + Having (fluido) | ✅ | ✅ | ✅ |
| Model.Query (JSON) | ✅ | ✅ | ✅ |
| NewQuery / GetField / GetColumn / GetFrom / ToJson | ✅ | ✅ | ✅ |
| Test (genera el SQL sin ejecutarlo) / Debug | ✅ | ✅ | ✅ |

## 5. Condiciones

| Caso | sqlite | postgres | oracle |
|---|---|---|---|
| Eq | ✅ | ✅ | ✅ |
| Neg | ✅ | ✅ | ✅ |
| Less | ✅ | ✅ | ✅ |
| LessEq | ✅ | ✅ | ✅ |
| More | ✅ | ✅ | ✅ |
| MoreEq | ✅ | ✅ | ✅ |
| Like | ✅ | ✅ | ✅ |
| In | ✅ | ✅ | ✅ |
| NotIn | ✅ | ✅ | ✅ |
| Is (NULL) | ✅ | ✅ | ✅ |
| IsNot (NULL) | ✅ | ✅ | ✅ |
| Null | ✅ | ✅ | ✅ |
| NotNull | ✅ | ✅ | ✅ |
| Between | ✅ | ✅ | ✅ |
| NotBetween | ✅ | ✅ | ✅ |
| Where (genérico) | ✅ | ✅ | ✅ |
| And / Or (conectores) | ✅ | ✅ | ✅ |

## 6. Comandos

| Caso | sqlite | postgres | oracle |
|---|---|---|---|
| Insert (RETURNING) | ✅ | ✅ | ✅ |
| Bulk | ✅ | ✅ | ✅ |
| Update + Where | ✅ | ✅ | ✅ |
| Update + Where + Or (varias filas) | ✅ | ✅ | ✅ |
| Upsert (inserta y luego actualiza) | ✅ | ✅ | ✅ |
| Return (campos del RETURNING) | ✅ | ✅ | ✅ |
| Delete (devuelve la fila borrada) | ✅ | ✅ | ✅ |
| Test (no ejecuta) + ToJson | ✅ | ✅ | ✅ |

## 7. Triggers

| Caso | sqlite | postgres | oracle |
|---|---|---|---|
| Before/After Insert, Update, Delete e InsertOrUpdate | ✅ | ✅ | ✅ |
| Trigger del comando + error que aborta | ✅ | ✅ | ✅ |
| Triggers JS (DefineBeforeInsert / DefineAfterUpdate…) | ✅ | ✅ | ✅ |

## 8. Transacciones

| Caso | sqlite | postgres | oracle |
|---|---|---|---|
| NewTx + ExecTx + Rollback | ✅ | ✅ | ✅ |
| NewTx + ExecTx + Commit | ✅ | ✅ | ✅ |

## 9. Series

| Caso | sqlite | postgres | oracle |
|---|---|---|---|
| DefineSeries | ✅ | ✅ | ✅ |
| SetSeries + GetSeries | ✅ | ✅ | ✅ |
| GenValue + GenSerie | ✅ | ✅ | ✅ |
| DeleteSeries | ✅ | ✅ | ✅ |

## 10. Auditoría

| Caso | sqlite | postgres | oracle |
|---|---|---|---|
| OnAuditLog + Model.AddAuditLog + AddAuditLog | ✅ | ✅ | ✅ |

## 11. Utilidades

| Caso | sqlite | postgres | oracle |
|---|---|---|---|
| Quoted / EscapeSQLString / SQLParse / JsonString | ✅ | ✅ | ✅ |
| ArgWhitAs / ArgWhitSchema / StatusList / TypeColumn.Str | ✅ | ✅ | ✅ |
| RowsToItems | ✅ | ✅ | ✅ |
| Model.Db / SetDb / GetModel / ToJson, Column.ToJson, Schema.ToJson | ✅ | ✅ | ✅ |
