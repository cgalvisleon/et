# Especificación del paquete `jsql`

> Fuente original: [`feature.md`](feature.md). Este documento la formaliza y la contrasta con el código actual de `jsql/` y sus drivers (`jsql/drivers/`). La sección [Brechas](#12-brechas-entre-la-especificación-y-el-código) lista lo que la especificación pide y el código todavía no cumple.

## 1. Objetivo

El objetivo de `jsql` es ser un **SQL builder que recibe estructuras JSON (`et.Json`) como parámetros de entrada** y las traduce a SQL del motor configurado. Debe interpretar las cláusulas:

| Consulta         | Uniones                                        | Filtros              | Agrupación / orden               | Paginación        | Comandos                                          |
| ---------------- | ---------------------------------------------- | -------------------- | -------------------------------- | ----------------- | ------------------------------------------------- |
| `select`, `from` | `join`, `left join`, `right join`, `full join` | `where`, `and`, `or` | `group by`, `having`, `order by` | `limit`, `offset` | `insert`, `update`, `delete` (+ `upsert`, `bulk`) |

Además, cubre la **definición y creación (DDL)** de base de datos, esquema y tablas a partir de modelos.

El SQL concreto lo genera un `Driver` por motor (ver §9); el núcleo de `jsql` solo arma estructuras (`Model`, `Query`, `Command`) independientes del motor.

## 2. Conceptos

| Concepto      | Tipo Go                                                     | Descripción                                                                                                                                                                   |
| ------------- | ----------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Base de datos | `*jsql.DB` (`db.go`)                                        | Conexión (`*sql.DB`), driver, esquemas y límite de registros (`RecordLimit`).                                                                                                 |
| Esquema       | `*jsql.Schema` (`schema.go`)                                | Agrupa modelos por nombre.                                                                                                                                                    |
| Modelo        | `*jsql.Model` (`model.go`)                                  | Definición de una tabla: columnas, índices, llaves, `SourceField`, relaciones y triggers.                                                                                     |
| Columna       | `*jsql.Column` (`column.go`)                                | `Name`, `TypeColumn`, `TypeData` (`et.TypeData`), `Default`.                                                                                                                  |
| Origen        | `*jsql.From` (`query.go`)                                   | Ubicación de un modelo (`Database`, `Schema`, `Name`, `Table`) y su **alias** `As`.                                                                                           |
| Campo         | `et.Field` (`et/condition.go`) / `*jsql.Field` (`query.go`) | `et.Field` es la referencia sintáctica (`Source`, `Name`, `As`, `Agg`, `Page`), sin modelo. `jsql.Field` la embebe y le agrega el `From` resuelto, `TypeColumn` y `TypeData`. |
| Consulta      | `*jsql.Query` (`query.go`)                                  | SQL de tipo consulta (`SELECT`).                                                                                                                                              |
| Comando       | `*jsql.Command` (`command.go`)                              | SQL de tipo comando: `INSERT`, `UPDATE`, `DELETE`, `UPSERT`, `BULK`.                                                                                                          |
| Transacción   | `*jsql.Tx` (`tx.go`)                                        | Envoltura de `*sql.Tx`; `nil` = transacción propia con auto-commit.                                                                                                           |

### 2.1 Tipos de columna (`TypeColumn`)

| Constante  | Valor       | Persistencia                                                                                                                                                                                                                             |
| ---------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `COLUMN`   | `column`    | Columna que se crea en la tabla.                                                                                                                                                                                                         |
| `ATTRIB`   | `atrib`     | Atributo sin columna propia, que se guarda dentro del campo `SourceField`.                                                                                                                                                               |
| `DETAIL`   | `detail`    | Relación maestro-detalle: el modelo del detalle, las keys que unen el maestro con el detalle, los campos que se muestran y cuántos registros se muestran. Se resuelve con una sub-consulta por fila.                                     |
| `MASTER`   | `master`    | Relación a través de una tabla intermedia: el modelo destino, el modelo puente, las keys del maestro al puente y del puente al destino, los campos que se muestran y cuántos registros se muestran; con un registro es 1 a 1 (ver §2.3). |
| `ROLLUP`   | `rollup`    | Consulta hacia otro modelo que devuelve un solo registro (`row`, `object`) o un agregado (`count`, `sum`, `avg`, `min`, `max`) y lo asigna al atributo del rollup (ver §2.3).                                                            |
| `CALCFUNC` | `calc_func` | Función de Go (`CalcFunction`) que se ejecuta sobre cada fila cuando la columna está incluida en el select.                                                                                                                              |
| `CALC`     | `calc`      | Script de JavaScript que se ejecuta con goja (`jrex`) sobre cada fila cuando la columna está incluida en el select.                                                                                                                      |

Las columnas `DETAIL`, `MASTER`, `ROLLUP`, `CALCFUNC` y `CALC` solo se ejecutan si se incluyen de manera literal en el select (o con `Query.Detail`, `Query.Master` y `Query.Calc`). Con el select vacío solo se incluyen las columnas `COLUMN` y, si el modelo tiene `SourceField`, sus atributos (§5.3).

Estructura de una columna (`column.go`):

```go
type Column struct {
    Name       string      `json:"name"`
    TypeColumn TypeColumn  `json:"type_column"`
    TypeData   et.TypeData `json:"type_data"`
    Default    any         `json:"default"`
    model      *Model      `json:"-"`
}
```

### 2.2 Columnas estándar (`column.go`)

`id`, `_idx`, `_source`, `status`, `version`, `tenant_id`, `project_id`, `created_at`, `updated_at` y `result` (alias de la columna JSON devuelta por los `SELECT` con `SourceField`).

### 2.3 Relaciones

| Relación | Definición                                                                                                                                                                                                                                             | Resultado en la fila                                                                                                                                                                                                                                                                                        |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Detalle  | `model.DefineDetail(name, keys, rows, selects...)` crea el modelo `<modelo>_<name>` con la llave foránea. `keys` va del campo del maestro al del detalle, `rows` es cuántos registros se muestran y `selects` (opcional) los campos que se muestran.   | `name` es la lista de registros del detalle (hasta `rows`), con los campos de `selects` o, sin ellos, todos.                                                                                                                                                                                                |
| Maestro  | `model.DefineMaster(name, to, keys, toKeys, selects, rows...)` crea el modelo puente `<modelo>_<to>`. `keys` va del maestro al puente, `toKeys` del destino al puente, `selects` son los campos que se muestran y `rows` (opcional) cuántos registros. | Con `rows = 1` la relación es 1 a 1 y `name` es el registro enlazado como objeto (o `null`). Con `rows > 1`, o sin `rows`, `name` es la lista de registros enlazados (hasta `rows` o `DB_RECORD_LIMIT`). `Model.Bridge(name)` devuelve el puente para crear los enlaces y `Model.Master(name)` la consulta. |
| Rollup   | `model.DefineRollup(name, to, keys, selects, operation)`. `keys` va del campo de la fila al campo de `to`.                                                                                                                                             | Depende de la operación (tabla siguiente).                                                                                                                                                                                                                                                                  |

| Operación                  | Resultado                                                                                                                                           |
| -------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `count`                    | Cantidad de registros de `to` (sin `selects`).                                                                                                      |
| `sum`, `avg`, `min`, `max` | El agregado del campo de `selects` sobre el modelo `to`, asignado a `name`.                                                                         |
| `row`                      | La consulta con `limit 1` del campo de `selects`, cuyo valor se asigna a `name`. Con varios campos en `selects`, se asigna el registro como objeto. |
| `object`                   | El primer registro con los campos de `selects`, como objeto asignado a `name`.                                                                      |

No hace falta seleccionar los campos llave de una relación: si la consulta pide un detalle, un maestro o un rollup y no incluye sus llaves (por ejemplo `tp_documento`), `jsql` las agrega para resolver la relación y las quita del resultado.

Ejemplo: el atributo `tp_documento` guarda `CC`, `NIT` o `RUT`, y su significado está en el modelo `tipo_documentos` (`id`, `title`). Un rollup `row` con `selects: ["title"]` consulta la columna `title` con `limit 1` y la asigna al atributo que lleva el nombre del rollup; uno `object` con `selects: ["id", "title"]` asigna el objeto completo a ese atributo.

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

| Clave       | Destino                |
| ----------- | ---------------------- |
| `id`        | columna `id`           |
| `name`      | columna `name`         |
| `last_name` | `_source->'last_name'` |

La misma regla aplica en:

| Operación                                    | Comportamiento esperado                                                                                                                                                                           |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `INSERT`                                     | Las claves sin columna se guardan en `SourceField`.                                                                                                                                               |
| `UPDATE`                                     | Las claves sin columna se fusionan en `SourceField` sin borrar las otras (Postgres: `COALESCE(_source, '{}') \|\| {...}` y `jsonb_set` para las rutas anidadas, creando los objetos intermedios). |
| `WHERE` / `HAVING` / `ORDER BY` / `GROUP BY` | Un campo sin columna se resuelve como `SourceField->>'campo'`, con _cast_ según el tipo declarado o, si no tiene tipo, según el valor comparado (número, booleano o fecha).                       |
| `SELECT`                                     | Ver §5.3.                                                                                                                                                                                         |

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

`Define` admite además `ForeignKeys`, `Indexes`, `Required`, `Hiddens`, `OmitUpdates`, `Details` (`DefDetail`), `Masters` (`DefMaster`), `Rollups` (`DefRollup`) y `UserId` (auditoría).

2. **Programática**: `db.NewModel(schema, name, version, userId)` + `DefineColumn`, `DefineAttrib`, `DefineIndex`, `DefinePrimaryKey`, `DefineUnique`, `DefineRequired`, `DefineHidden`, `DefineOmitUpdate`, `DefineForeignKeys`, `DefineDetail`, `DefineMaster`, `DefineRollup`, `DefineCalcFunc`, `DefineCalc`, `DefineSource`, `DefineIdxField`.

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

| Forma           | Ejemplo               | Significado                               |
| --------------- | --------------------- | ----------------------------------------- |
| `campo`         | `name`                | Campo del primer `From`.                  |
| `campo:as`      | `name:nombre`         | Con alias.                                |
| `from.campo`    | `A.name`              | Campo de un origen por nombre o alias.    |
| `from.campo:as` | `A.name:nombre`       | Origen + alias.                           |
| `agg(campo)`    | `count(id)`           | Agregación; alias = nombre de la función. |
| `agg(campo):as` | `sum(total):total`    | Agregación con alias.                     |
| `campo->a->b`   | `_source->addr->city` | Ruta JSON anidada.                        |
| `campo\|page:n` | `items\|page:2`       | Página de un detalle (por defecto, 1).    |

`et.ToField` interpreta el texto sin conocer los modelos; `Query.GetField` resuelve después el origen (`Source` → `From`, por nombre o alias) y la columna. Un nombre sin columna en un modelo con `SourceField` se resuelve como `ATTRIB` (§3). Las funciones de agregación válidas son las de `et.ToFunction`; un nombre desconocido no es un campo válido.

### 5.3 Semántica del `SELECT` con `SourceField`

`select` es una lista (`[]interface{}` / `[]string`) de campos.

| `select`   | Modelo **con** `SourceField`                                                                                                                  | Modelo **sin** `SourceField`            |
| ---------- | --------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------- |
| vacío      | `A._source \|\| jsonb_build_object('id', A.id, 'name', A.name, …) AS result`: todos los atributos más todas las columnas `COLUMN` no ocultas. | Todas las columnas `COLUMN` no ocultas. |
| con campos | `jsonb_build_object('id', A.id, 'last_name', A._source->>'last_name', …) AS result`: solo los `Field` pedidos, agrupados en un objeto.        | Lista de columnas pedidas.              |

Los campos ocultos (`Model.Hiddens` y `Query.Hiddens`) se excluyen cuando el select está vacío; si se piden por nombre en el select, se devuelven. Con el select vacío no se resuelven detalles, maestros, rollups ni campos calculados: hay que pedirlos por nombre (§2.1). La fila resultante con `SourceField` es un único JSON en la columna `result`, que `jsql` devuelve como `et.Json`.

### 5.4 Consulta descrita en JSON

`Model.Query(json)` / `Model.QueryTx(tx, json)` construyen un `*Query` a partir de un `et.Json` (`Query.loadQuery`). Sin `from`, el origen es el propio modelo con el alias `A`, que es el alias por defecto de toda consulta. En los `on` de un join, un valor de texto `alias.campo` que nombra un campo de la consulta se toma como columna, no como texto.

```json
{
  "from": { "database": "josephine", "schema": "public", "model": "users", "as": "A" },
  "selects": ["id", "name", "last_name"],
  "hiddens": ["password"],
  "join": [
    { "to": { "schema": "public", "model": "roles", "as": "R" }, "on": [{ "A.role_id": { "eq": "R.id" } }] }
  ],
  "left_join": [{ "to": "public.areas:B", "on": [{ "A.area_id": { "eq": "B.id" } }] }],
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

- `from` es el origen. **Forma estándar (feature.md):** un objeto `{database, schema, model, as}`; `database` es la base donde está el modelo (sin él, la de la conexión), sin `schema` el modelo se busca en todos los esquemas y debe ser único, y sin `as` el origen principal es `A`. **Forma corta:** el texto `schema.tabla:alias`, `tabla:alias` (usa el esquema del modelo que ejecuta la consulta), `schema.tabla` o `tabla` (alias `A`). Hoy el código solo acepta la forma corta (brecha B1). También acepta una lista: la primera referencia reemplaza al origen principal y las demás se agregan como orígenes adicionales (`FROM a, b`). En ese caso los orígenes se relacionan en el `where`, donde un valor de texto `alias.campo` que nombra un campo de la consulta se toma como columna, igual que en el `on` de un join. El modelo debe estar definido en la `DB`.
- `to` en los joins es obligatorio, con la misma estructura de `from` (objeto o forma corta `schema.tabla:alias`), y el alias es obligatorio; el modelo destino debe estar definido en la `DB`. Hoy solo se acepta la forma corta (B1).
- Si el descriptor tiene un error (un `from` o un `to` que no existe, una condición inválida), la consulta lo devuelve al ejecutarse (`All`, `One`, `Count`, `Exists`…) en lugar de correr con lo que se pudo leer.
- Las claves también se aceptan con su forma SQL (`select`, `group by`, `order by`, `having`…, ver §8.2), además de `offset` y las claves `and` / `or` de primer nivel.
- `limit` por defecto: `DB.RecordLimit` (`DB_RECORD_LIMIT`, 1000). `page` calcula el `OFFSET`; `offset` lo fija directamente.
- `orders`: `true` = ASC, `false` = DESC. Un orden puede usar el alias de un campo seleccionado, incluido un agregado (`"orders": [{"n": false}]` con `"count(id):n"`); lo mismo con `Query.OrderBy`.

## 6. Condiciones

Cada condición es un objeto `{campo: {operador: valor}}`, opcionalmente envuelto en un conector `and` / `or`. La primera condición no lleva conector (`et.ToConditions`).

| Operador JSON             | Constructor Go                     | SQL                                                                                                                          |
| ------------------------- | ---------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `eq`                      | `jsql.Eq`                          | `=`                                                                                                                          |
| `neg`                     | `jsql.Neg`                         | `!=`                                                                                                                         |
| `less` / `less_eq`        | `jsql.Less` / `jsql.LessEq`        | `<` / `<=`                                                                                                                   |
| `more` / `more_eq`        | `jsql.More` / `jsql.MoreEq`        | `>` / `>=`                                                                                                                   |
| `like`                    | `jsql.Like`                        | `ILIKE` / `LIKE`                                                                                                             |
| `in` / `not_in`           | `jsql.In` / `jsql.NotIn`           | `IN (…)` / `NOT IN (…)`                                                                                                      |
| `is` / `is_not`           | `jsql.Is` / `jsql.IsNot`           | `IS NULL` / `IS NOT NULL` con `nil`; con un valor, comparación que respeta `NULL` (`IS [NOT] DISTINCT FROM`, `<=>` en MySQL) |
| `null` / `not_null`       | `jsql.Null` / `jsql.NotNull`       | `IS NULL` / `IS NOT NULL`                                                                                                    |
| `between` / `not_between` | `jsql.Between` / `jsql.NotBetween` | `BETWEEN … AND …`                                                                                                            |

Un campo sin columna se busca en `SourceField` (`_source->>'last_name'`), según §3. El ejemplo de `where` de §3 usa este formato.

## 7. Comandos (`Command`)

| Tipo     | Constructor                     | Comportamiento                                                                                                                                       |
| -------- | ------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `INSERT` | `model.Insert(data)`            | Valida `Required`, corre triggers _before_, genera y ejecuta el SQL, corre triggers _after_.                                                         |
| `BULK`   | `model.Bulk([]data)`            | `INSERT` por cada elemento.                                                                                                                          |
| `UPDATE` | `model.Update(data).Where(...)` | Lee las filas que cumplen el `where`, hasta el límite del comando (ver _Límite_ abajo); por cada una, `new = old ⊕ data`, triggers, SQL, triggers.   |
| `DELETE` | `model.Delete().Where(...)`     | Lee las filas que cumplen el `where`, hasta el límite del comando; triggers, SQL, triggers.                                                          |
| `UPSERT` | `model.Upsert(data).Where(...)` | **A partir del resultado del `where`**: si existe → `UPDATE`, si no → `INSERT`. El `where` es obligatorio: sin él devuelve `ErrUpsertWhereRequired`. |

Reglas:

- **`SourceField`**: `INSERT` y `UPDATE` aplican §3 a las claves de `data` sin columna.
- **`RETURNING`**: los tres comandos incluyen `RETURNING`. `INSERT` y `UPDATE` devuelven los **datos actualizados**; `DELETE` devuelve los **datos eliminados**. El resultado es la fila que devuelve la base, con la misma forma que un `SELECT` sin campos: con `SourceField`, `SourceField || jsonb_build_object(columnas) AS result`, sin los campos ocultos. Los triggers _after_ reciben `new` con esos datos fusionados. `Command.Return(fields...)` restringe los campos devueltos y los resuelve igual que `Select`: columnas, atributos de `SourceField`, alias `campo:as` y rutas anidadas; con `SourceField` el resultado es un JSON solo con esos campos.
- **Campos que no se actualizan** (`Model.OmitUpdates`): columnas y atributos que un `update` (o el update de un `upsert`) nunca cambia, por ejemplo `created_at` o `created_by`. Se definen con `model.DefineOmitUpdate(names...)`, el campo `OmitUpdates` de `Define` o la clave `omit_updates` del JSON. Se quitan de `data` (también sus rutas anidadas `campo->…`) y, si un trigger _before_ los cambia, se restauran al valor anterior. El `insert` sí los guarda.
- **Límite**: `update`, `delete` y el update de un `upsert` trabajan sobre un número máximo de filas, para no poner en riesgo la estabilidad de la base con un `where` muy amplio. `Command.Limit(n)` (o `"limit"` en JSON) lo fija: `n > 0` afecta como máximo `n` filas y `0` a todas las que cumplen el `where`. Sin límite se usa `DB_RECORD_LIMIT`, con un máximo de 1000. Qué filas entran cuando hay más que el límite no está garantizado.
- **Filtros**: `Where`/`And`/`Or` reciben `*et.Condition` y admiten el mismo `[]Json` de §6.
- **Ejecución**: `Exec()`, `ExecTx(tx)` → `et.Items`; `One()`, `OneTx(tx)` → `et.Item`. Con `tx == nil`, el comando abre su propia transacción y hace commit.
- **Campos únicos**: antes de cada insert y update, `jsql` verifica los campos definidos con `DefineUnique` (sin contar los vacíos) dentro de la transacción del comando, así que también detecta duplicados dentro de un mismo `bulk`. En un update solo revisa el campo si cambia, y excluye la fila que se actualiza por su llave primaria. Un duplicado devuelve `ErrRecordAlreadyExists` con el campo y el valor (`record already exists: email = ana@example.com`), que se puede comprobar con `errors.Is`. El índice único de la base de datos se mantiene como garantía final.
- **Triggers**: `TriggerFunction func(tx *Tx, old, new et.Json) error` para `Before/After × Insert/Update/Delete` (en `Model` o por `Command`), y scripts JS (`jrex`) registrados con `DefineBeforeInsert(code)`, `DefineBeforeUpdate(name, code)`, etc. En el script los registros son `OLD` y `NEW` (`new` es palabra reservada de JavaScript), por ejemplo `NEW.estado = "revisado";`. Un error en un trigger aborta el comando y, si el comando abrió su propia transacción, la revierte.
- **Campos calculados**: `DefineCalcFunc(name, fn)` en Go y `DefineCalc(name, script)` en JS, que recibe la fila como `item` (por ejemplo `item.inicial = item.name.substring(0, 1);`).

## 8. Consultas y comandos descritos en JSON sobre la `DB` (`jquery.go`)

`DB.Query(json)` / `DB.QueryTx(tx, json)` reciben un descriptor JSON independiente de un modelo: el modelo se indica en el propio JSON (`from`). Según la clave principal, el descriptor es un comando (`insert`, `update`, `delete`, `upsert`, `bulk`), una definición (`define`) o una consulta (`select`, `from`); se revisan en ese orden, sin distinguir mayúsculas, y un descriptor con dos comandos o sin ninguna de esas claves devuelve error. El descriptor se traduce a las mismas estructuras de `jsql` (`Query`, `Command`, `Define`), así que sigue las reglas de `SourceField`, `RETURNING` y triggers de las secciones anteriores.

En `from` va el modelo, como objeto `{database, schema, model}` (forma estándar) o como texto `schema.tabla`; sin esquema, la tabla se busca en todos los esquemas de la `DB` y debe ser única. Hoy solo se acepta el texto (B1).

### 8.1 Comandos

Cada comando va bajo una clave con su nombre y contiene:

| Clave                                         | Uso                                                                                                                                      | Comandos                             |
| --------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| `from`                                        | Modelo destino: `{database, schema, model}` o `schema.tabla`.                                                                            | todos                                |
| `data`                                        | Valores a guardar: un objeto (`insert`, `update`, `upsert`) o una lista de objetos (`bulk`). Los campos sin columna van a `SourceField`. | `insert`, `update`, `upsert`, `bulk` |
| `where`                                       | Condiciones con el formato de §6. En `upsert` nunca puede estar vacío.                                                                   | `update`, `delete`, `upsert`         |
| `limit`                                       | Máximo de filas afectadas: sin enviar, `DB_RECORD_LIMIT` con tope de 1000; `0`, todas las que cumplen `where`.                           | `update`, `delete`, `upsert`         |
| `before_insert`, `after_insert`               | Listas de código JavaScript (goja) que corren antes y después de cada insert.                                                            | `insert`, `bulk`, `upsert`           |
| `before_update`, `after_update`               | Ídem, antes y después de actualizar cada registro.                                                                                       | `update`, `upsert`                   |
| `before_delete`, `after_delete`               | Ídem, antes y después de eliminar cada registro.                                                                                         | `delete`                             |
| `before_insert_update`, `after_insert_update` | Ídem, tanto si el `upsert` inserta como si actualiza.                                                                                    | `upsert`                             |

```json
{
  "update": {
    "from": { "schema": "public", "model": "users" },
    "data": { "name": "Cesar Galvis" },
    "where": [
      { "id": { "eq": 1 } },
      { "and": { "status": { "eq": "active" } } }
    ],
    "limit": 100,
    "before_update": ["NEW.updated_by = 'api';"]
  }
}
```

| Comando  | Comportamiento                                                                                                                               | Resultado                                |
| -------- | -------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| `insert` | Inserta un registro con `data`.                                                                                                              | El registro insertado.                   |
| `bulk`   | Inserta un registro por cada objeto de `data`; los triggers corren por registro.                                                             | Los registros insertados.                |
| `update` | Actualiza los registros que cumplen `where` con `data`, hasta `limit`; los atributos se fusionan en `SourceField` sin borrar los existentes. | Los registros actualizados.              |
| `delete` | Elimina los registros que cumplen `where`, hasta `limit`.                                                                                    | Los registros eliminados.                |
| `upsert` | Si ningún registro cumple `where`, inserta `data`; si alguno lo cumple, actualiza esos registros con `data`, hasta `limit`.                  | Los registros insertados o actualizados. |

En los scripts, los registros están en `NEW` (con los valores que se van a guardar; en `before_*` sus cambios se guardan) y `OLD` (el registro antes del cambio, vacío en un insert), como en §7.

### 8.2 Consultas y definiciones

- **Consulta**: `from` (obligatorio) más las claves de §5.4. Cada clave admite también su forma SQL: `select` (= `selects`), `group by` / `group_by` (= `groups`), `having` (= `havings`), `order by` / `order_by` (= `orders`), `left join` / `right join` / `full join` (= `left_join`…). Además acepta `offset` y las claves `and` / `or` de primer nivel, que agregan condiciones con ese conector después del `where`. El resultado son las filas de la consulta.
- **Definición**: `define` con la estructura `Define` (§4.1) en JSON (`schema`, `name`, `columns`, `primary_keys`, `source_field`…). Define el modelo, crea la tabla si no existe y devuelve la definición del modelo.

## 9. Drivers

El driver recibe un `Model` para el DDL, un `Query` para el SQL de consulta y un `Command` para el SQL de comando:

```go
type Driver interface {
	CreateDB(connection *ConnectParams, timeout ...time.Duration) error // crea la base de la conexión si no existe
	DropDB(db *DB, timeout ...time.Duration) error                      // elimina la base de los params si existe
	Connect(db *DB, timeout ...time.Duration) (*sql.DB, error)
	ExistModel(db *sql.DB, model *Model, timeout ...time.Duration) (bool, error)
	Load(model *Model, timeout ...time.Duration) (string, error)        // DDL: tablas y sus elementos
	Query(query *Query, timeout ...time.Duration) (string, error)       // SQL de consulta
	Command(command *Command, timeout ...time.Duration) (string, error) // SQL de comando: INSERT, UPDATE, DELETE
}
```

- Cada driver vive en `jsql/drivers/<nombre>/`, se registra en `init()` con `jsql.Register(nombre, driver)` y se importa por efecto lateral (`_ "github.com/cgalvisleon/et/jsql/drivers/postgres"`).
- El driver solo **genera SQL** (y abre la conexión); ejecutarlo, manejar transacciones y correr los triggers es trabajo del núcleo.
- **Timeout:** cada método recibe un `timeout` opcional y el driver arma su contexto con `jsql.TimeoutContext(timeout...)`: sin timeout o con `0` no hay límite, así que la operación nunca falla por timeout; con un valor mayor falla con `context.DeadlineExceeded` al vencerse. El núcleo pasa el `Timeout` de `ConnectParams` (`DB.timeout`). `Load`, `Query` y `Command` solo generan SQL y no lo usan.
- `CreateDB` y `DropDB` (expuestos como `jsql.CreateDB(connection, timeout...)` y `jsql.DropDB(db, timeout...)`, que antes cierra el pool) son idempotentes. En postgres, mysql y mssql crean/eliminan la base `database` desde la base del servidor (`postgres`, sin base, `master`), y `Connect` usa `CreateDB`; en sqlite crean/borran el archivo (con sus `-wal` y `-shm`); en oracle el servicio no se crea ni se elimina desde una conexión: `CreateDB` solo comprueba que responde y `DropDB` devuelve `MSG_DRIVER_CANNOT_DROP_DB`. Solo postgres está probado (brecha B8).
- La conexión se describe con `jsql.Connection` (`GetParams`, `SetDatabase`, `GetDatabase`), creada según `DB_DRIVER` en `conection.go`.

| Driver                           | Constante        | Estado                                                                 |
| -------------------------------- | ---------------- | ---------------------------------------------------------------------- |
| `postgres` (`lib/pq`)            | `DriverPostgres` | Completo.                                                              |
| `sqlite` (`modernc.org/sqlite`)  | `DriverSqlite`   | Completo; `RIGHT JOIN` y `FULL JOIN` nativos (SQLite 3.39 o superior). |
| `oracle` (`go-ora/v2`)           | `DriverOracle`   | Completo, para Oracle 19c o superior (probado en 23ai Free). Ver §9.1. |
| `mysql` (`go-sql-driver/mysql`)  | `DriverMysql`    | Completo, para MySQL 8.0 o superior (probado en 8.4). Ver §9.2.        |
| `mssql` (`microsoft/go-mssqldb`) | `DriverMssql`    | Completo, para SQL Server 2022 o superior. Ver §9.3.                   |
| `josefina`                       | constante        | Sin implementación.                                                    |

### 9.1 Particularidades del driver Oracle

- **Nombres**: tablas y columnas se crean como identificadores entre comillas y en minúscula (`JSQL."users"."_source"`), así conservan el nombre del modelo y admiten `_source` o `_idx`. El esquema del modelo es un usuario de Oracle que debe existir (`Load` no lo crea) y se escribe sin comillas, así que se resuelve en mayúscula.
- **Tipos**: el JSON se guarda en `CLOB` con `CHECK (... IS JSON)`; `BOOL` es `NUMBER(1)`, que en los resultados JSON vuelve como `true`/`false`; `ANY` es `VARCHAR2(4000)`.
- **Comandos**: Oracle no devuelve filas con `RETURNING`, así que `Command` genera un bloque PL/SQL que ejecuta el DML, recoge los `ROWID` afectados y devuelve esas filas con `DBMS_SQL.RETURN_RESULT`. El `DELETE` abre el cursor antes de borrar.
- **`SourceField`**: el `UPDATE` fusiona los atributos con `JSON_MERGEPATCH`, que conserva los hermanos anidados; un valor `null` en `data` **borra** la llave, en lugar de guardar `null` como en Postgres. Por la misma razón, las columnas con valor `NULL` no aparecen en el `result`.
- **Literales**: los textos de más de 1000 caracteres se parten en `TO_CLOB('…') || TO_CLOB('…')` para no pasar el límite de 4000 bytes por literal (`ORA-01704`). Oracle guarda `''` como `NULL`.
- **Consultas**: `LIKE` no distingue mayúsculas (`UPPER(x) LIKE UPPER(v)`); la paginación usa `OFFSET … ROWS FETCH NEXT … ROWS ONLY`; `ON UPDATE CASCADE` no existe y se omite; no se crean índices sobre LOB ni un segundo índice sobre una columna ya indexada.

### 9.2 Particularidades del driver MySQL

- **Esquemas**: el esquema de un modelo es una base de datos de MySQL (`CREATE SCHEMA` la crea); tablas y columnas van entre backticks. `Connect` crea la base de `DB_NAME` si no existe.
- **Tipos**: `SourceField` y las columnas JSON son `JSON`; `BOOL` es `TINYINT(1)`, que en los resultados vuelve como `true`/`false`; `DATETIME` es `DATETIME(6)`; `ANY` es `TEXT`. No se indexan columnas `TEXT`, `BLOB` ni `JSON`.
- **Comandos**: MySQL no tiene `RETURNING`; la conexión usa `multiStatements` y cada comando es un lote: el DML y un `SELECT` de la fila por su llave primaria (el `DELETE` hace el `SELECT` antes de borrar).
- **Escapes**: MySQL usa `\` como carácter de escape dentro de las cadenas, así que el driver escapa `\` además de `'`.
- **`SourceField`**: el `UPDATE` fusiona los atributos con `JSON_SET` (conserva los `null`) y crea antes los objetos padre de las rutas anidadas.
- **Consultas**: `FULL JOIN` no existe en MySQL y devuelve `mysql.ErrFullJoin`. En consultas agrupadas, los campos seleccionados se envuelven en `ANY_VALUE`, porque `only_full_group_by` no reconoce la expresión JSON del `GROUP BY`. `LIKE` compara en minúsculas.

### 9.3 Particularidades del driver SQL Server

- **Nombres**: tablas y columnas van entre corchetes (`[schema].[tabla]`); sin esquema se usa `dbo`. `Connect` crea la base de `DB_NAME` si no existe y `Load` crea el esquema.
- **Tipos**: `SourceField` y las columnas JSON son `NVARCHAR(MAX)` con `CHECK (ISJSON(...) = 1)`; `BOOL` es `BIT`; `DATETIME` es `DATETIME2`; `ANY` es `NVARCHAR(4000)`. No se indexan columnas `MAX` ni `NVARCHAR(4000)`.
- **Resultados**: SQL Server no tiene una función que arme objetos JSON conservando los tipos, así que el driver construye el JSON de cada fila como texto (`STRING_ESCAPE` para los textos, números, `true`/`false`, fechas ISO 8601) y lee los atributos de `SourceField` con `OPENJSON` para conservar su tipo.
- **Comandos**: cada comando es un lote con `SET NOCOUNT ON`: el DML y un `SELECT` de la fila por su llave primaria (el `DELETE` hace el `SELECT` antes de borrar).
- **`SourceField`**: el `UPDATE` fusiona los atributos con `JSON_MODIFY` y crea antes los objetos padre; como en Oracle, un valor `null` **borra** la llave.
- **Consultas**: la paginación usa `OFFSET … ROWS FETCH NEXT … ROWS ONLY`, que exige un `ORDER BY` (`ORDER BY (SELECT NULL)` si la consulta no tiene); `IS [NOT] DISTINCT FROM` requiere SQL Server 2022.

## 10. Conexión y configuración

`jsql.Load()` (base `DB_NAME`) o `jsql.LoadTo(nombre, host...)` leen el entorno con `envar`; `jsql.ConnectTo(ConnectParams)` recibe los parámetros explícitos.

| Variable                                                                                  | Uso                                                                   | Defecto                  |
| ----------------------------------------------------------------------------------------- | --------------------------------------------------------------------- | ------------------------ |
| `DB_DRIVER`                                                                               | Driver: `postgres`, `sqlite`, `oracle`, `mysql` o `mssql`             | `postgres`               |
| `DB_HOST`, `DB_PORT`                                                                      | Servidor                                                              | `localhost`, 5432 / 1521 |
| `DB_USER`, `DB_PASSWORD`, `DB_NAME`                                                       | Credenciales y base (en Oracle, el _service name_, p. ej. `FREEPDB1`) | — / `josephine`          |
| `DB_SSL`, `DB_SSL_VERIFY`                                                                 | TLS (oracle)                                                          | `false`, `true`          |
| `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_CONN_LIFETIME`, `DB_POOL_CONN_IDLE_TIME` | Pool                                                                  | driver                   |
| `DB_RECORD_LIMIT`                                                                         | Límite de filas por consulta                                          | 1000                     |
| `DB_IS_DEBUG`                                                                             | Registra el SQL generado                                              | `false`                  |
| `MAX_AUDIT_LOG`                                                                           | Entradas de auditoría por modelo                                      | 1000                     |

## 11. Utilidades

- `jsql.DefineSeries(db, schema)` → `*Series`: consecutivos y códigos (`SetSeries`, `GetSeries`, `DeleteSeries`, `GenSerie`, `GenValue`).
- Helpers (`helpers.go`): `SQLParse` (reemplaza `$1`…`$N` por los argumentos escapados en una sola pasada; `$10` no se confunde con `$1`), `Quoted`, `EscapeSQLString`, `RowsToItems`, `ArgWhitAs` (`campo:alias`), `ArgWhitSchema` (`schema.tabla`), `AddAuditLog`.
- Estados (`column.go`): `ACTIVE`, `ARCHIVED`, `CANCELED`, `OF_SYSTEM`, `FOR_DELETE`, `PENDING`, `APPROVED`, `REJECTED` (+ `IN_PROCESS`, `FAILED`).

## 12. Brechas entre la especificación y el código

Revisado el 2026-09-28 contra el código y la batería `TestCatalog` (`jsql/drivers/test`), corrida ese día en una copia de `et`: **sqlite y postgres pasan 97 de 97** (1 omitido: `Load`/`LoadTo`, que lee el entorno). Oracle y MySQL pasaron 97 de 97 el 2026-09-27 y no estaban disponibles el 28; **SQL Server nunca se ha corrido** en la batería. `go build` y `go vet` de `jsql/...` pasan.

**Cumple:** cláusulas de consulta, `SourceField` en insert/update/where/select, `RETURNING` en los tres comandos, `hiddens`, relaciones (detail, master, rollup), campos calculados (Go y JS), triggers en Go y en JS, `limit` de update/delete/upsert (1000 por defecto, `0` = todas), `upsert` con `where` obligatorio, `bulk`, `define` en JSON, campos únicos, `OmitUpdates`, series y auditoría.

| # | Especificación | Código actual | Ubicación | Impacto |
|---|---|---|---|---|
| B1 | `from` y `to` como objeto `{database, schema, model, as}` (forma estándar de `feature.md`, en consultas, joins y comandos). | Solo texto: `loadFrom` rechaza un objeto (`MSG_INVALID_FROM`), los joins leen `js.Str("to")`, y `parseQuery` / `parseCommand` leen `from` como texto. Todos los ejemplos de `feature.md` fallan hoy. | `query.go` (`loadFrom`, `loadQuery`), `jquery.go` (`parseQuery`, `parseCommand`, `findModel`) | **Alto** |
| B2 | `database` en `from`/`to` indica la base del modelo. | No hay registro de bases por nombre: cada `*DB` es una conexión, y `From.Database` solo se informa (los drivers no lo usan). Una consulta entre bases distintas no es posible en un solo SQL. Propuesta: aceptar `database` solo si coincide con la base de la conexión y, si no, devolver error. | `query.go` (`From`), `catalog.go` | Medio |
| B3 | La clave de maestros en `Define` (JSON) es `masters`, como `details` y `rollups`. | La etiqueta es `json:"master"` (singular). | `define.go` | Bajo |
| B4 | Driver `mssql` completo (§9). | Genera SQL, pero no se ha corrido nunca contra un SQL Server en la batería. | `drivers/mssql` | Medio |
| B5 | Driver para varios motores. | `josefina` es solo una constante, sin implementación. | `drivers/` | Bajo |
| B6 | Los mensajes de error son `MSG_*` de `msg.go`. | `findModel` arma un error con texto fijo («exists in more than one schema»). | `jquery.go` | Bajo |
| B7 | Probar sin efectos laterales. | `TestCatalog` reescribe `result/*.json`, `result/*.sql` y `result/summary.md` en cada corrida; el resumen deja de reflejar los motores que no estaban disponibles. | `drivers/test/catalog_test.go` | Bajo |
| B8 | `CreateDB` / `DropDB` probados en todos los drivers (§9). | `TestCreateDropDB` solo corre contra postgres (pasa, 2026-09-30); sqlite, oracle, mysql y mssql se omiten como pendientes (`createDBDrivers`). Su implementación compila pero nunca se ha ejecutado. | `drivers/test/createdb_test.go`, `drivers/{sqlite,oracle,mysql,mssql}/connect.go` | Medio |

### Orden recomendado

1. **B1 + B2** juntos: un `fromOf(value any)` que acepte objeto, texto o lista y devuelva `(database, schema, model, as)`, usado por `loadFrom`, los cuatro joins, `parseQuery` y `parseCommand`; con `database` validado contra la conexión. Casos nuevos en `TestCatalog` para la forma objeto en consulta, join y comando.
2. **B3:** aceptar `masters` (y `master` por compatibilidad) al leer el JSON.
3. **B6** y **B7:** mensajes y un modo de la batería que no escriba resultados (por ejemplo, con una variable de entorno).
4. **B4:** correr la batería contra un SQL Server 2022 (por ejemplo en Docker) antes de declarar el driver completo.
5. **B8:** agregar cada driver a `createDBDrivers` en `createdb_test.go` y correr `TestCreateDropDB` contra él (sqlite no necesita servidor; en oracle el caso debe esperar el error de `DropDB`).
6. **B5:** fuera de alcance hasta que `josefina` tenga su protocolo estable.
