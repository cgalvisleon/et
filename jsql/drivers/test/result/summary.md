# Resultado de TestCatalog

Generado por `go test -run TestCatalog` en `jsql/drivers/test`. El detalle de cada caso (salida y SQL) está en `<driver>.json` y `<driver>.sql`.

| Driver | Pasan | Fallan | Brechas | Omitidos |
|---|---|---|---|---|
| sqlite | 96 | 0 | 0 | 1 |
| postgres | 96 | 0 | 0 | 1 |
| oracle | 96 | 0 | 0 | 1 |
| mysql | 96 | 0 | 0 | 1 |
| mssql | 96 | 0 | 0 | 1 |

## 1. Conexión y base de datos

| Caso | sqlite | postgres | oracle | mysql | mssql |
|---|---|---|---|---|---|
| Register + ConnectTo | ✅ | ✅ | ✅ | ✅ | ✅ |
| Init (idempotente) | ✅ | ✅ | ✅ | ✅ | ✅ |
| NewDB | ✅ | ✅ | ✅ | ✅ | ✅ |
| LoadDb (desde ToJson) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Load / LoadTo | ➖ omitido | ➖ omitido | ➖ omitido | ➖ omitido | ➖ omitido |
| Sql | ✅ | ✅ | ✅ | ✅ | ✅ |
| SqlTx + Tx.Commit | ✅ | ✅ | ✅ | ✅ | ✅ |
| Connection.GetParams / SetDatabase / GetDatabase | ✅ | ✅ | ✅ | ✅ | ✅ |
| SetDebug / Debug | ✅ | ✅ | ✅ | ✅ | ✅ |
| DB.Query con descriptor inválido (devuelve error) | ✅ | ✅ | ✅ | ✅ | ✅ |

## 2. Definición de modelos (DDL)

| Caso | sqlite | postgres | oracle | mysql | mssql |
|---|---|---|---|---|---|
| Define (declarativo) | ✅ | ✅ | ✅ | ✅ | ✅ |
| DefineModel + DefineColumn / DefineAttrib / DefineUnique / DefineHidden | ✅ | ✅ | ✅ | ✅ | ✅ |
| DefineTenantModel / DefineProjectModel | ✅ | ✅ | ✅ | ✅ | ✅ |
| DefineRequired (rechaza el insert sin el campo) | ✅ | ✅ | ✅ | ✅ | ✅ |
| DefineForeignKeys (rechaza un hijo sin padre) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Stricted (ignora campos desconocidos) | ✅ | ✅ | ✅ | ✅ | ✅ |
| GetModel / RemoveModel / NewModel | ✅ | ✅ | ✅ | ✅ | ✅ |
| DB.Query define (Define en JSON) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Insert (datos base) | ✅ | ✅ | ✅ | ✅ | ✅ |
| DefineUnique (rechaza duplicados en insert, bulk y update) | ✅ | ✅ | ✅ | ✅ | ✅ |

## 3. Relaciones y campos calculados

| Caso | sqlite | postgres | oracle | mysql | mssql |
|---|---|---|---|---|---|
| Detail en el select (DefineDetail) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Model.Detail + Query.Detail | ✅ | ✅ | ✅ | ✅ | ✅ |
| Master en el select (DefineMaster) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Master 1 a 1 (rows = 1) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Model.Master + Model.Bridge | ✅ | ✅ | ✅ | ✅ | ✅ |
| DefineRollup row (tp_doc → title) | ✅ | ✅ | ✅ | ✅ | ✅ |
| DefineRollup (count, sum, object) | ✅ | ✅ | ✅ | ✅ | ✅ |
| DefineCalcFunc + Model.Calc | ✅ | ✅ | ✅ | ✅ | ✅ |
| Query.Calc con script JS (DefineCalc) | ✅ | ✅ | ✅ | ✅ | ✅ |
| DefineCalc (script JS) | ✅ | ✅ | ✅ | ✅ | ✅ |

## 4. Consultas

| Caso | sqlite | postgres | oracle | mysql | mssql |
|---|---|---|---|---|---|
| Select + Where + OrderBy + All | ✅ | ✅ | ✅ | ✅ | ✅ |
| One / First / Count / Exists | ✅ | ✅ | ✅ | ✅ | ✅ |
| Limit (paginación) / Page | ✅ | ✅ | ✅ | ✅ | ✅ |
| Hidden (campo oculto en la consulta y en el modelo) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Hidden pedido por nombre (se devuelve) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Join (fluido) | ✅ | ✅ | ✅ | ✅ | ✅ |
| LeftJoin + GroupBy | ✅ | ✅ | ✅ | ✅ | ✅ |
| RightJoin / FullJoin | ✅ | ✅ | ✅ | ✅ | ✅ |
| GroupBy + Having (fluido) | ✅ | ✅ | ✅ | ✅ | ✅ |
| OrderBy por alias de un agregado | ✅ | ✅ | ✅ | ✅ | ✅ |
| Model.Query (JSON) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Model.Query con from (otro modelo y alias) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Model.Query con from de varios orígenes | ✅ | ✅ | ✅ | ✅ | ✅ |
| Model.Query con from sin esquema | ✅ | ✅ | ✅ | ✅ | ✅ |
| Model.Query con from + join + groups | ✅ | ✅ | ✅ | ✅ | ✅ |
| Model.Query con from inválido (devuelve error) | ✅ | ✅ | ✅ | ✅ | ✅ |
| DB.Query consulta (select, from, join, group by, order by) | ✅ | ✅ | ✅ | ✅ | ✅ |
| DB.Query consulta (where + and/or de primer nivel, limit, offset) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Model.Query con claves select / group by / order by | ✅ | ✅ | ✅ | ✅ | ✅ |
| NewQuery / GetField / GetColumn / GetFrom / ToJson | ✅ | ✅ | ✅ | ✅ | ✅ |
| Test (genera el SQL sin ejecutarlo) / Debug | ✅ | ✅ | ✅ | ✅ | ✅ |

## 5. Condiciones

| Caso | sqlite | postgres | oracle | mysql | mssql |
|---|---|---|---|---|---|
| Eq | ✅ | ✅ | ✅ | ✅ | ✅ |
| Neg | ✅ | ✅ | ✅ | ✅ | ✅ |
| Less | ✅ | ✅ | ✅ | ✅ | ✅ |
| LessEq | ✅ | ✅ | ✅ | ✅ | ✅ |
| More | ✅ | ✅ | ✅ | ✅ | ✅ |
| MoreEq | ✅ | ✅ | ✅ | ✅ | ✅ |
| Like | ✅ | ✅ | ✅ | ✅ | ✅ |
| In | ✅ | ✅ | ✅ | ✅ | ✅ |
| NotIn | ✅ | ✅ | ✅ | ✅ | ✅ |
| Is (NULL) | ✅ | ✅ | ✅ | ✅ | ✅ |
| IsNot (NULL) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Null | ✅ | ✅ | ✅ | ✅ | ✅ |
| NotNull | ✅ | ✅ | ✅ | ✅ | ✅ |
| Between | ✅ | ✅ | ✅ | ✅ | ✅ |
| NotBetween | ✅ | ✅ | ✅ | ✅ | ✅ |
| Where (genérico) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Is / IsNot con valor (comparación NULL-safe) | ✅ | ✅ | ✅ | ✅ | ✅ |
| And / Or (conectores) | ✅ | ✅ | ✅ | ✅ | ✅ |

## 6. Comandos

| Caso | sqlite | postgres | oracle | mysql | mssql |
|---|---|---|---|---|---|
| Insert (RETURNING) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Bulk | ✅ | ✅ | ✅ | ✅ | ✅ |
| Update + Where | ✅ | ✅ | ✅ | ✅ | ✅ |
| Update + Where + Or (varias filas) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Upsert (inserta y luego actualiza) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Return (campos como en Select: columna, atributo, alias, ruta) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Delete (devuelve la fila borrada) | ✅ | ✅ | ✅ | ✅ | ✅ |
| DB.Query insert + bulk (con triggers JS) | ✅ | ✅ | ✅ | ✅ | ✅ |
| DB.Query update + delete | ✅ | ✅ | ✅ | ✅ | ✅ |
| DB.Query upsert (inserta, actualiza y exige where) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Update / Delete con limit (por defecto, n y 0 = todas) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Upsert fluido sin where (devuelve error) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Test (no ejecuta) + ToJson | ✅ | ✅ | ✅ | ✅ | ✅ |

## 7. Triggers

| Caso | sqlite | postgres | oracle | mysql | mssql |
|---|---|---|---|---|---|
| Before/After Insert, Update, Delete e InsertOrUpdate | ✅ | ✅ | ✅ | ✅ | ✅ |
| Trigger del comando + error que aborta | ✅ | ✅ | ✅ | ✅ | ✅ |
| Triggers JS (DefineBeforeInsert / DefineAfterUpdate…) | ✅ | ✅ | ✅ | ✅ | ✅ |

## 8. Transacciones

| Caso | sqlite | postgres | oracle | mysql | mssql |
|---|---|---|---|---|---|
| NewTx + ExecTx + Rollback | ✅ | ✅ | ✅ | ✅ | ✅ |
| NewTx + ExecTx + Commit | ✅ | ✅ | ✅ | ✅ | ✅ |

## 9. Series

| Caso | sqlite | postgres | oracle | mysql | mssql |
|---|---|---|---|---|---|
| DefineSeries | ✅ | ✅ | ✅ | ✅ | ✅ |
| SetSeries + GetSeries | ✅ | ✅ | ✅ | ✅ | ✅ |
| GenValue + GenSerie | ✅ | ✅ | ✅ | ✅ | ✅ |
| DeleteSeries | ✅ | ✅ | ✅ | ✅ | ✅ |

## 10. Auditoría

| Caso | sqlite | postgres | oracle | mysql | mssql |
|---|---|---|---|---|---|
| OnAuditLog + Model.AddAuditLog + AddAuditLog | ✅ | ✅ | ✅ | ✅ | ✅ |

## 11. Utilidades

| Caso | sqlite | postgres | oracle | mysql | mssql |
|---|---|---|---|---|---|
| Quoted / EscapeSQLString / SQLParse / JsonString | ✅ | ✅ | ✅ | ✅ | ✅ |
| SQLParse con 10 o más parámetros | ✅ | ✅ | ✅ | ✅ | ✅ |
| ArgWhitAs / ArgWhitSchema / StatusList / TypeColumn.Str | ✅ | ✅ | ✅ | ✅ | ✅ |
| RowsToItems | ✅ | ✅ | ✅ | ✅ | ✅ |
| Model.Db / SetDb / GetModel / ToJson, Column.ToJson, Schema.ToJson | ✅ | ✅ | ✅ | ✅ | ✅ |
