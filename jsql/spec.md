# Especificación del paquete `jsql`

> Fuente original: [`../feature.md`](../feature.md). Este documento la formaliza y la contrasta con el código actual de `jsql/` y `jsql/drivers/postgres/`. La sección [Brechas](#12-brechas-entre-la-especificación-y-el-código) lista lo que la especificación pide y el código todavía no cumple.

## 1. Objetivo

El objetivo de `jsql` es ser un **SQL builder que recibe estructuras JSON (`et.Json`) como parámetros de entrada** y las traduce a SQL del motor configurado. Debe interpretar las cláusulas:

| Consulta | Uniones | Filtros | Agrupación / orden | Paginación | Comandos |
|---|---|---|---|---|---|
| `select`, `from` | `join`, `left join`, `right join`, `full join` | `where`, `and`, `or` | `group by`, `having`, `order by` | `limit`, `offset` | `insert`, `update`, `delete` (+ `upsert`, `bulk`) |

Además, cubre la **definición y creación (DDL)** de base de datos, esquema y tablas a partir de modelos.

El SQL concreto lo genera un `Driver` por motor (ver §9); el núcleo de `jsql` solo arma estructuras (`Model`, `Query`, `Command`) independientes del motor.

## 2. Conceptos

| Concepto | Tipo Go | Descripción |
|---|---|---|
| Base de datos | `*jsql.DB` (`db.go`) | Conexión (`*sql.DB`), driver, esquemas y límite de registros (`RecordLimit`). |
| Esquema | `*jsql.Schema` (`schema.go`) | Agrupa modelos por nombre. |
| Modelo | `*jsql.Model` (`model.go`) | Definición de una tabla: columnas, índices, llaves, `SourceField`, relaciones y triggers. |
| Columna | `*jsql.Column` (`column.go`) | `Name`, `TypeColumn`, `TypeData` (`et.TypeData`), `Default`. |
| Origen | `*jsql.From` (`query.go`) | Ubicación de un modelo (`Database`, `Schema`, `Name`, `Table`) y su **alias** `As`. |
| Campo | `et.Field` (`et/condition.go`) / `*jsql.Field` (`query.go`) | `et.Field` es la referencia sintáctica (`Source`, `Name`, `As`, `Agg`, `Page`), sin modelo. `jsql.Field` la embebe y le agrega el `From` resuelto, `TypeColumn` y `TypeData`. |
| Consulta | `*jsql.Query` (`query.go`) | SQL de tipo consulta (`SELECT`). |
| Comando | `*jsql.Command` (`command.go`) | SQL de tipo comando: `INSERT`, `UPDATE`, `DELETE`, `UPSERT`, `BULK`. |
| Transacción | `*jsql.Tx` (`tx.go`) | Envoltura de `*sql.Tx`; `nil` = transacción propia con auto-commit. |

### 2.1 Tipos de columna (`TypeColumn`)

| Constante | Valor | Persistencia |
|---|---|---|
| `COLUMN` | `column` | Columna real de la tabla. |
| `ATTRIB` | `atrib` | Llave dentro del campo JSONB `SourceField`. |
| `DETAIL` | `detail` | Relación 1‑N virtual (sub-consulta por fila). |
| `MASTER` | `master` | Relación N‑1/N‑N virtual (vía modelo puente). |
| `ROLLUP` | `rollup` | Agregado (`count`, `sum`, `avg`, `min`, `max`, `row`, `object`) sobre otro modelo. |
| `CALCFUNC` | `calc_func` | Calculado en Go (`CalcFunction`) tras la consulta. |
| `CALC` | `calc` | Calculado con script JS (`jrex`) tras la consulta. |
| `AGG` | `agg` | Expresión de agregación. |

### 2.2 Columnas estándar (`column.go`)

`id`, `_idx`, `_source`, `status`, `version`, `tenant_id`, `project_id`, `created_at`, `updated_at` y `result` (alias de la columna JSON devuelta por los `SELECT` con `SourceField`).

## 3. `SourceField`: atributos en JSONB

Regla central de la especificación.

- Todo `Model` tiene el atributo `SourceField`. Si es `""`, el modelo es relacional puro. Si no, contiene el nombre de una columna **JSONB** (por convención `_source`, creada con `DefineSource()`).
- **Cualquier valor sin columna definida en el modelo se almacena como atributo dentro de `SourceField`.** Implementación: `Model.GetColumn(name)` devuelve una columna sintética `ATTRIB` (tipo `ANY`) para nombres desconocidos cuando el modelo tiene `SourceField` y no es estricto.
- `Model.Stricted()` desactiva esa regla: los nombres desconocidos se ignoran o son error.

Por ejemplo, imagina un modelo `Users` con las columnas `id`, `name` y `_source` (`SourceField = "_source"`) en el que se hace un insert con este JSON:

```json
{ "id": 1, "name": "Cesar", "last_name": "Galvis" }
```

`last_name` no tiene una columna definida, así que se guarda dentro de `_source`:

| Clave | Destino |
|---|---|
| `id` | columna `id` |
| `name` | columna `name` |
| `last_name` | `_source->'last_name'` |

La misma regla aplica en:

| Operación | Comportamiento esperado |
|---|---|
| `INSERT` | Las claves sin columna se guardan en `SourceField`. |
| `UPDATE` | Las claves sin columna se fusionan en `SourceField` (Postgres: `jsonb_set` encadenado por llave, sin borrar las otras). |
| `WHERE` / `HAVING` / `ORDER BY` / `GROUP BY` | Un campo sin columna se resuelve como `SourceField->>'campo'` (con *cast* si el tipo es conocido). |
| `SELECT` | Ver §5.3. |

En esta condición, `last_name` no tiene columna, así que se busca como atributo: `_source->>'last_name'`.

```json
{
  "where": [
    { "id": { "eq": 1 } },
    { "and": { "name": { "eq": "Cesar" } } },
    { "or": { "last_name": { "eq": "Galvis" } } }
  ]
}
```

Rutas anidadas: `"a->b->c"` se traduce a `->` en los niveles intermedios y `->>` en la hoja; si `a` es `ATTRIB`, la ruta parte de `SourceField` (`_source->'a'->'b'->>'c'`).

## 4. Definición de modelos (DDL)

### 4.1 Formas de definir

1. **Declarativa** con `jsql.Define` (preferida):

```go
model, err := db.Define(jsql.Define{
    Schema: "public", Name: "users", Version: 1,
    SourceField: jsql.SOURCE, IdxField: jsql.IDX,
    Columns: []jsql.Column{
        {Name: "email", TypeColumn: jsql.COLUMN, TypeData: et.TEXT, Default: ""},
        {Name: "name",  TypeColumn: jsql.ATTRIB, TypeData: et.TEXT, Default: ""},
    },
    PrimaryKeys: []jsql.DefIndex{{Name: "id", Sorted: true}},
    Unique:      []jsql.DefIndex{{Name: "email"}},
})
```

   `Define` admite además `ForeignKeys`, `Indexes`, `Required`, `Hiddens`, `Details` (`DefDetail`), `Masters` (`DefMaster`), `Rollups` (`DefRollup`) y `UserId` (auditoría).

2. **Programática**: `db.NewModel(schema, name, version, userId)` + `DefineColumn`, `DefineAttrib`, `DefineIndex`, `DefinePrimaryKey`, `DefineUnique`, `DefineRequired`, `DefineHidden`, `DefineForeignKeys`, `DefineDetail`, `DefineMaster`, `DefineRollup`, `DefineCalcFunc`, `DefineCalc`, `DefineSource`, `DefineIdxField`.

3. **Plantillas**: `db.DefineModel(...)` (añade `created_at`, `updated_at`, `status`, `id` PK, `_source`, `_idx`), `db.DefineTenantModel(...)` (+ `tenant_id`), `db.DefineProjectModel(...)` (+ `project_id`).

### 4.2 Creación

- `Model.Init()`: si `Driver.ExistModel` indica que la tabla no existe, ejecuta el DDL de `Driver.Load(model)`. Después inicializa sus `Details` y `Masters`. Las llamadas siguientes no hacen nada.
- `Driver.Load` genera en orden: `CREATE SCHEMA IF NOT EXISTS`, `CREATE TABLE IF NOT EXISTS` (solo columnas `COLUMN`; `SourceField` como JSONB), `PRIMARY KEY`, índices únicos, índices (`Sorted` → BTREE, si no → HASH) y llaves foráneas (`ON DELETE/UPDATE CASCADE` opcionales).
- `Driver.Connect` puede crear la base de datos si no existe (Postgres lo hace; Oracle no: el servicio debe existir).

## 5. Consultas (`Query`)

### 5.1 API fluida

```go
items, err := model.
    Where(jsql.Eq("status", jsql.ACTIVE)).
    And(jsql.More("age", 18)).
    Select("id", "name", "last_name").
    OrderBy("name", true).
    Limit(1, 20) // page, rows
```

Constructores: `Model.Select/Where/Join/As/Calc`, `jsql.NewQuery(model, as...)`. Cláusulas: `Join`, `LeftJoin`, `RightJoin`, `FullJoin`, `Where`, `And`, `Or`, `GroupBy`, `Having`, `OrderBy`, `Page`, `Hidden`, `Detail`, `Master`, `Calc`. Ejecución: `All`, `One`, `First(n)`, `Limit(page, rows)`, `Count`, `Exists` y sus variantes `…Tx(tx)`. `Debug()` registra el SQL; `Test()` lo genera sin ejecutarlo.

`And`/`Or` se aplican a la sección activa: `where`, el `on` del último `join` o `having`.

### 5.2 Sintaxis de campo (`et.ToField`)

| Forma | Ejemplo | Significado |
|---|---|---|
| `campo` | `name` | Campo del primer `From`. |
| `campo:as` | `name:nombre` | Con alias. |
| `from.campo` | `A.name` | Campo de un origen por nombre o alias. |
| `from.campo:as` | `A.name:nombre` | Origen + alias. |
| `agg(campo)` | `count(id)` | Agregación; alias = nombre de la función. |
| `agg(campo):as` | `sum(total):total` | Agregación con alias. |
| `campo->a->b` | `_source->addr->city` | Ruta JSON anidada. |
| `campo\|page:n` | `items\|page:2` | Página de un detalle (por defecto, 1). |

`et.ToField` interpreta el texto sin conocer los modelos; `Query.GetField` resuelve después el origen (`Source` → `From`, por nombre o alias) y la columna. Un nombre sin columna en un modelo con `SourceField` se resuelve como `ATTRIB` (§3). Las funciones de agregación válidas son las de `et.ToFunction`; un nombre desconocido no es un campo válido.

### 5.3 Semántica del `SELECT` con `SourceField`

`select` es una lista (`[]interface{}` / `[]string`) de campos.

| `select` | Modelo **con** `SourceField` | Modelo **sin** `SourceField` |
|---|---|---|
| vacío | `A._source \|\| jsonb_build_object('id', A.id, 'name', A.name, …) AS result`: todos los atributos más todas las columnas `COLUMN` no ocultas. | Todas las columnas `COLUMN` no ocultas. |
| con campos | `jsonb_build_object('id', A.id, 'last_name', A._source->>'last_name', …) AS result`: solo los `Field` pedidos, agrupados en un objeto. | Lista de columnas pedidas. |

Siempre se excluyen `Model.Hiddens` y `Query.Hiddens`. La fila resultante con `SourceField` es un único JSON en la columna `result`, que `jsql` devuelve como `et.Json`.

### 5.4 Consulta descrita en JSON

`Model.Query(json)` / `Model.QueryTx(tx, json)` construyen un `*Query` a partir de un `et.Json` (`Query.loadQuery`). El `from` es el propio modelo (alias `A`).

```json
{
  "selects": ["id", "name", "last_name"],
  "hiddens": ["password"],
  "join": [
    { "to": "public.roles:R", "on": [{ "A.role_id": { "eq": "R.id" } }] }
  ],
  "left_join": [{ "to": "schema.tabla:alias", "on": [] }],
  "right_join": [],
  "full_join": [],
  "where": [
    { "id": { "eq": 1 } },
    { "and": { "name": { "eq": "Cesar" } } },
    { "or": { "last_name": { "eq": "Galvis" } } }
  ],
  "groups": ["status"],
  "havings": [{ "count(id)": { "more": 1 } }],
  "orders": [{ "name": true }, { "created_at": false }],
  "limit": 20,
  "page": 1
}
```

- `to` en los joins es obligatorio con la forma `schema.tabla:alias`; el modelo destino debe estar definido en la `DB`.
- `limit` por defecto: `DB.RecordLimit` (`DB_RECORD_LIMIT`, 1000). `page` calcula el `OFFSET`.
- `orders`: `true` = ASC, `false` = DESC.

## 6. Condiciones

Cada condición es un objeto `{campo: {operador: valor}}`, opcionalmente envuelto en un conector `and` / `or`. La primera condición no lleva conector (`et.ToConditions`).

| Operador JSON | Constructor Go | SQL |
|---|---|---|
| `eq` | `jsql.Eq` | `=` |
| `neg` | `jsql.Neg` | `!=` |
| `less` / `less_eq` | `jsql.Less` / `jsql.LessEq` | `<` / `<=` |
| `more` / `more_eq` | `jsql.More` / `jsql.MoreEq` | `>` / `>=` |
| `like` | `jsql.Like` | `ILIKE` / `LIKE` |
| `in` / `not_in` | `jsql.In` / `jsql.NotIn` | `IN (…)` / `NOT IN (…)` |
| `is` / `is_not` | `jsql.Is` / `jsql.IsNot` | `IS` / `IS NOT` |
| `null` / `not_null` | `jsql.Null` / `jsql.NotNull` | `IS NULL` / `IS NOT NULL` |
| `between` / `not_between` | `jsql.Between` / `jsql.NotBetween` | `BETWEEN … AND …` |

Un campo sin columna se busca en `SourceField` (`_source->>'last_name'`), según §3. El ejemplo de `where` de §3 usa este formato.

## 7. Comandos (`Command`)

| Tipo | Constructor | Comportamiento |
|---|---|---|
| `INSERT` | `model.Insert(data)` | Valida `Required`, corre triggers *before*, genera y ejecuta el SQL, corre triggers *after*. |
| `BULK` | `model.Bulk([]data)` | `INSERT` por cada elemento. |
| `UPDATE` | `model.Update(data).Where(...)` | Lee las filas que cumplen el `where`; por cada una, `new = old ⊕ data`, triggers, SQL, triggers. |
| `DELETE` | `model.Delete().Where(...)` | Lee las filas que cumplen el `where`; triggers, SQL, triggers. |
| `UPSERT` | `model.Upsert(data).Where(...)` | **A partir del resultado del `where`**: si existe → `UPDATE`, si no → `INSERT`. |

Reglas:

- **`SourceField`**: `INSERT` y `UPDATE` aplican §3 a las claves de `data` sin columna.
- **`RETURNING`**: los tres comandos incluyen `RETURNING`. `INSERT` y `UPDATE` devuelven los **datos actualizados**; `DELETE` devuelve los **datos eliminados**. Con `SourceField`, los atributos se reconstruyen en el resultado. `Command.Return(fields...)` restringe los campos devueltos.
- **Filtros**: `Where`/`And`/`Or` reciben `*et.Condition` y admiten el mismo `[]Json` de §6.
- **Ejecución**: `Exec()`, `ExecTx(tx)` → `et.Items`; `One()`, `OneTx(tx)` → `et.Item`. Con `tx == nil`, el comando abre su propia transacción y hace commit.
- **Triggers**: `TriggerFunction func(tx *Tx, old, new et.Json) error` para `Before/After × Insert/Update/Delete` (en `Model` o por `Command`), y scripts JS (`jrex`) registrados con `DefineBeforeInsert`, etc. Un error en un trigger aborta el comando.

## 8. Consulta SQL descrita en JSON sobre la `DB` (`jquery.go`)

`DB.Query(json)` / `DB.QueryTx(tx, json)`: SQL libre descrito por palabras clave, independiente de un modelo. Claves reconocidas (sin distinguir mayúsculas; con espacio o guion bajo): `select`, `from`, `join`, `left join`, `right join`, `full join`, `where`, `and`, `or`, `group by`, `having`, `order by`, `limit`, `offset`, `insert`, `update`, `delete`, `upsert`, `define`.

## 9. Drivers

El driver recibe un `Model` para el DDL, un `Query` para el SQL de consulta y un `Command` para el SQL de comando:

```go
type Driver interface {
	Connect(ctx context.Context, db *DB) (*sql.DB, error)
	ExistModel(db *sql.DB, model *Model) (bool, error)
	Load(model *Model) (string, error)        // DDL: tablas y sus elementos
	Query(query *Query) (string, error)       // SQL de consulta
	Command(command *Command) (string, error) // SQL de comando: INSERT, UPDATE, DELETE
}
```

- Cada driver vive en `jsql/drivers/<nombre>/`, se registra en `init()` con `jsql.Register(nombre, driver)` y se importa por efecto lateral (`_ "github.com/cgalvisleon/et/jsql/drivers/postgres"`).
- El driver solo **genera SQL** (y abre la conexión); ejecutarlo, manejar transacciones y correr los triggers es trabajo del núcleo.
- La conexión se describe con `jsql.Connection` (`GetParams`, `SetDatabase`, `GetDatabase`), creada según `DB_DRIVER` en `conection.go`.

| Driver | Constante | Estado |
|---|---|---|
| `postgres` (`lib/pq`) | `DriverPostgres` | Completo. |
| `sqlite` (`modernc.org/sqlite`) | `DriverSqlite` | Completo; cada fila es un objeto JSON y el JSON anidado llega doblemente codificado. |
| `oracle` (`go-ora/v2`) | `DriverOracle` | Solo `Connect`; `Load`/`Query`/`Command` devuelven `not implemented`. |
| `mysql`, `mssql`, `josefina` | constantes | Sin implementación. |

## 10. Conexión y configuración

`jsql.Load()` (base `DB_NAME`) o `jsql.LoadTo(nombre, host...)` leen el entorno con `envar`; `jsql.ConnectTo(ConnectParams)` recibe los parámetros explícitos.

| Variable | Uso | Defecto |
|---|---|---|
| `DB_DRIVER` | Driver | `postgres` |
| `DB_HOST`, `DB_PORT` | Servidor | `localhost`, 5432 / 1521 |
| `DB_USER`, `DB_PASSWORD`, `DB_NAME` | Credenciales y base / servicio | — / `josephine` |
| `DB_SSL`, `DB_SSL_VERIFY` | TLS (oracle) | `false`, `true` |
| `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_CONN_LIFETIME`, `DB_POOL_CONN_IDLE_TIME` | Pool | driver |
| `DB_RECORD_LIMIT` | Límite de filas por consulta | 1000 |
| `DB_IS_DEBUG` | Registra el SQL generado | `false` |
| `MAX_AUDIT_LOG` | Entradas de auditoría por modelo | 1000 |

## 11. Utilidades

- `jsql.DefineSeries(db, schema)` → `*Series`: consecutivos y códigos (`SetSeries`, `GetSeries`, `DeleteSeries`, `GenSerie`, `GenValue`).
- Helpers (`helpers.go`): `SQLParse`, `Quoted`, `EscapeSQLString`, `RowsToItems`, `ArgWhitAs` (`campo:alias`), `ArgWhitSchema` (`schema.tabla`), `AddAuditLog`.
- Estados (`column.go`): `ACTIVE`, `ARCHIVED`, `CANCELED`, `OF_SYSTEM`, `FOR_DELETE`, `PENDING`, `APPROVED`, `REJECTED` (+ `IN_PROCESS`, `FAILED`).

## 12. Brechas entre la especificación y el código

Estado revisado el 2026-09-26.

| # | Especificación | Código actual | Ubicación |
|---|---|---|---|
| 1 | `DB.Query(json)` interpreta `select`, `from`, `join`… | Todos los `parse*` devuelven `""`, así que la llamada siempre falla con `invalid sql`. Además, iterar el `map` no garantiza el orden de las cláusulas. | `jquery.go` |
| 2 | `INSERT`/`UPDATE`/`DELETE` devuelven los datos del `RETURNING`. | El SQL incluye `RETURNING`, pero `insert`/`update`/`delete` descartan el resultado de `SqlTx` y devuelven `s.New` / `s.Old` armados en memoria. | `command.go` |
| 3 | `select` en JSON recibe `[]interface{}` bajo la clave `select`. | `loadQuery` lee `selects` (y `hiddens`, `groups`, `havings`, `orders`) como `[]string`. | `query.go` `loadQuery` |
| 4 | `Query` admite `from` en JSON. | `loadQuery` no lee `from`: el origen es siempre el modelo que invoca. | `query.go` |
| 5 | Validación de índices únicos antes de insertar/actualizar. | `defaultTrigger` calcula si existe duplicado, pero el resultado de `results.Range` se ignora y nunca devuelve error. | `trigger.go` |
| 6 | Driver para varios motores. | Solo `postgres` y `sqlite` generan SQL; `oracle` solo conecta. | `drivers/` |
