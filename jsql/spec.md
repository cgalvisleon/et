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
| Esquema       | `*jsql.Schema` (`db.go`)                                    | Agrupa modelos por nombre.                                                                                                                                                    |
| Modelo        | `*jsql.Model` (`model.go`)                                  | Definición de una tabla: columnas, índices, llaves, `SourceField`, relaciones y triggers.                                                                                     |
| Columna       | `*jsql.Column` (`define.go`)                                | `Name`, `TypeColumn`, `TypeData` (`et.TypeData`), `Default`.                                                                                                                  |
| Origen        | `*jsql.From` (`query.go`)                                   | Ubicación de un modelo (`Database`, `Schema`, `Name`, `Table`) y su **alias** `As`.                                                                                           |
| Campo         | `et.Field` (`et/condition.go`) / `*jsql.Field` (`query.go`) | `et.Field` es la referencia sintáctica (`Source`, `Name`, `As`, `Agg`, `Page`), sin modelo. `jsql.Field` la embebe y le agrega el `From` resuelto, `TypeColumn` y `TypeData`. |
| Consulta      | `*jsql.Query` (`query.go`)                                  | SQL de tipo consulta (`SELECT`).                                                                                                                                              |
| Comando       | `*jsql.Command` (`command.go`)                              | SQL de tipo comando: `INSERT`, `UPDATE`, `DELETE`, `UPSERT`, `BULK`.                                                                                                          |
| Transacción   | `*jsql.Tx` (`db.go`)                                        | Envoltura de `*sql.Tx`; `nil` = transacción propia con auto-commit.                                                                                                           |

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

Estructura de una columna (`define.go`):

```go
type Column struct {
    Name       string      `json:"name"`
    TypeColumn TypeColumn  `json:"type_column"`
    TypeData   et.TypeData `json:"type_data"`
    Default    any         `json:"default"`
    model      *Model      `json:"-"`
}
```

### 2.2 Columnas estándar (`define.go`)

`id`, `_idx`, `_source`, `status`, `version`, `tenant_id`, `project_id`, `created_at`, `updated_at` y `result` (alias de la columna JSON devuelta por los `SELECT` con `SourceField`).

### 2.3 Relaciones

| Relación | Definición                                                                                                                                                                                                                                             | Resultado en la fila                                                                                                                                                                                                                                                                                        |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Detalle  | `model.DefineDetail(name, keys, rows, selects...)` crea el modelo `<modelo>_<name>` con la llave foránea. `keys` va del campo del maestro al del detalle, `rows` es cuántos registros se muestran y `selects` (opcional) los campos que se muestran.   | `name` es la lista de registros del detalle (hasta `rows`), con los campos de `selects` o, sin ellos, todos.                                                                                                                                                                                                |
| Maestro  | `model.DefineMaster(name, to, keys, toKeys, selects, rows...)` crea el modelo puente `<modelo>_<to>`. `keys` va del maestro al puente, `toKeys` del destino al puente, `selects` son los campos que se muestran y `rows` (opcional) cuántos registros. | Con `rows = 1` la relación es 1 a 1 y `name` es el registro enlazado como objeto (o `null`). Con `rows > 1`, o sin `rows`, `name` es la lista de registros enlazados (hasta `rows` o `DB_RECORD_LIMIT`). `Model.Bridge(name)` devuelve el puente para crear los enlaces y `Model.Master(name)` la consulta. |
| Rollup   | `model.DefineRollup(name, to, keys, selects, operation, where...)`. `keys` va del campo de la fila al campo de `to`; `where` (opcional, `*et.Condition`) filtra `to` (ver abajo).                                                                     | Depende de la operación (tabla siguiente).                                                                                                                                                                                                                                                                  |

| Operación                  | Resultado                                                                                                                                           |
| -------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `count`                    | Cantidad de registros de `to` (sin `selects`).                                                                                                      |
| `sum`, `avg`, `min`, `max` | El agregado del campo de `selects` sobre el modelo `to`, asignado a `name`.                                                                         |
| `row`                      | La consulta con `limit 1` del campo de `selects`, cuyo valor se asigna a `name`. Con varios campos en `selects`, se asigna el registro como objeto. |
| `object`                   | El primer registro con los campos de `selects`, como objeto asignado a `name`.                                                                      |

No hace falta seleccionar los campos llave de una relación: si la consulta pide un detalle, un maestro o un rollup y no incluye sus llaves (por ejemplo `tp_documento`), `jsql` las agrega para resolver la relación y las quita del resultado.

**Resolución por lotes.** Con más de una fila, cada detalle, maestro y rollup de una sola llave se resuelve con una consulta por cada 500 filas padre (`WHERE fk IN (...)`, `setRelations` en `relation.go`) en lugar de una por fila; los agregados se agrupan por la llave (`GROUP BY`) y los registros de cada padre se reparten en memoria (página y `rows` por padre). Se resuelven fila por fila una relación de varias llaves, un maestro sin `selects` y un rollup cuyo `where` lee campos de la fila. El resultado es el mismo que fila por fila (lo comprueba `TestCatalog`, «Relaciones por lotes = fila por fila»); sin registros relacionados, `count` y `sum` dan 0 y `avg`, `min`, `max` null en los cinco motores (Oracle pasó a `COALESCE(SUM(x), 0)` como los demás). Un detalle o maestro trae del motor todos los registros de sus padres y corta `rows` en memoria: con muchos registros por padre, pedir uno solo (`Where(id)`) evita traerlos. Una fila sin valor en la llave recibe una lista vacía (antes, sin la llave, la consulta del detalle no tenía filtro y traía todos los registros). Un error en una relación se devuelve (antes la fila quedaba incompleta sin aviso).

**Condiciones de un rollup.** `where` filtra el modelo `to` y se une a las llaves con `AND`: `DefineRollup("pedidos_pagados", pedidos, keys, nil, RollupCount, jsql.Eq("status", "paid"))` cuenta solo los pedidos pagados de cada fila. Un valor `{field: "x"}` (`et.Fld("x")` en Go, `field("x")` en scripts) **se lee de la fila resultante**, no de `to`: `jsql.More("amount", et.Fld("limite"))` compara cada pedido con el `limite` de su fila. Si la fila no trae un campo referenciado, el rollup no se aplica (como con una llave que falta), y si la consulta no lo seleccionó, `jsql` lo agrega y lo quita del resultado, igual que las llaves. Acepta `OR`: los drivers escriben las condiciones en orden y sin paréntesis, y SQL evalúa `AND` antes que `OR`, así que `jsql` reparte las llaves en cada término `OR` del `where` (`llaves AND (a AND b OR c)` se escribe `llaves AND a AND b OR llaves AND c`, `rollupConditions`): un `OR` nunca pierde el filtro de la llave. En JSON (`DefRollup`, el `ToJson` del modelo, `QueryRollups`) va como lista: `"where": [{"status": {"eq": "paid"}}, {"and": {"amount": {"more": {"field": "limite"}}}}]`. En scripts, `defineRollup(name, to, keys, selects, operation, where?)` recibe un objeto, una lista o `null`.

**Orden de un rollup.** Sin orden, un rollup `row` u `object` toma cualquier registro. `rollup.OrderBy(campo, asc...)` (Go y scripts; `order_by: [{name, sorted}]` en `DefRollup`) fija cuál es el primero; un agregado lo ignora.

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
  "from": {
    "database": "josephine",
    "schema": "public",
    "model": "users",
    "as": "A"
  }, // o "public.users:A"
  "selects": ["A.id", "A.name", "A.last_name"],
  "hiddens": ["A.password"],
  "join": [
    {
      "to": { "schema": "public", "model": "roles", "as": "R" },
      "on": [{ "A.role_id": { "eq": "R.id" } }]
    }
  ],
  "left_join": [
    { "to": "public.areas:B", "on": [{ "A.area_id": { "eq": "B.id" } }] }
  ],
  "right_join": [],
  "full_join": [],
  "where": [
    { "A.id": { "eq": 1 } },
    { "and": { "A.name": { "eq": "Cesar" } } },
    { "or": { "A.last_name": { "eq": "Galvis" } } }
  ],
  "groups": ["status"],
  "havings": [{ "count(id)": { "more": 1 } }],
  "orders": [{ "name": true }, { "created_at": false }],
  "limit": 20,
  "page": 1
}
```

- `from` es el origen. **Forma estándar (feature.md):** un objeto `{database, schema, model, as}`; `database` es la base donde está el modelo (sin él, la de la conexión), sin `schema` el modelo se busca en todos los esquemas y debe ser único, y sin `as` el origen principal es `A`. **Forma corta:** el texto `schema.tabla:alias`, `tabla:alias` (usa el esquema del modelo que ejecuta la consulta), `schema.tabla` o `tabla` (alias `A`). `database`, si viene, debe ser la base de la conexión; si no, es un error (`MSG_INVALID_DATABASE`). También acepta una lista: la primera referencia reemplaza al origen principal y las demás se agregan como orígenes adicionales (`FROM a, b`). En ese caso los orígenes se relacionan en el `where`, donde un valor de texto `alias.campo` que nombra un campo de la consulta se toma como columna, igual que en el `on` de un join. El modelo debe estar definido en la `DB`.
- `to` en los joins es obligatorio, con la misma estructura de `from` (objeto o forma corta `schema.tabla:alias`), y el alias es obligatorio; el modelo destino debe estar definido en la `DB`. Sin `schema`, se usa el del origen principal.
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

**Valor que es otro campo** (2026-10-15): `{"field": "<referencia>"}` como valor (`et.FIELD`, leída con `et.ToField`: `name`, `A.name`, `a->b`, `sum(A.x)`…) compara contra una columna o agregación de la consulta en vez de un literal: `{"A.spent": {"more": {"field": "A.budget"}}}`, y en `havings` `{"sum(A.spent)": {"less": {"field": "sum(A.budget)"}}}`. Se resuelve con `getField` en los cinco drivers (`<p>FieldValueRef`); un atributo sin tipo se castea al tipo del otro lado; un nombre que no es columna es un atributo de `SourceField` (ausente: `NULL`); una referencia que no se resuelve (origen desconocido) hace la condición `1 = 0`, nunca un texto citado. En los scripts, `field("x")`, `sum("x")`, `year("x")`… devuelven ese JSON. `TestFieldValue` lo prueba en los cinco motores.

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
- **Triggers**: `TriggerFunction func(tx *Tx, old, new et.Json) error` para `Before/After × Insert/Update/Delete` (en `Model` o por `Command`), y scripts JS (`jrex`) registrados con `DefineBeforeInsert(name, code, version)`, `DefineBeforeUpdate(name, code, version)`, etc. (se guardan como `[]*jwf.Script`; uno con el mismo `name` se reemplaza) En el script los registros son `OLD` y `NEW` (`new` es palabra reservada de JavaScript), por ejemplo `NEW.estado = "revisado";`. Un error en un trigger aborta el comando y, si el comando abrió su propia transacción, la revierte.
- **Campos calculados**: `DefineCalcFunc(name, fn)` en Go y `DefineCalc(name, script)` en JS, que recibe la fila como `item` (por ejemplo `item.inicial = item.name.substring(0, 1);`).

## 8. Consultas y comandos descritos en JSON sobre la `DB` (`jquery.go`)

`DB.Query(json)` / `DB.QueryTx(tx, json)` reciben un descriptor JSON independiente de un modelo: el modelo se indica en el propio JSON (`from`). Según la clave principal, el descriptor es un comando (`insert`, `update`, `delete`, `upsert`, `bulk`), una definición (`define`) o una consulta (`select`, `from`); se revisan en ese orden, sin distinguir mayúsculas, y un descriptor con dos comandos o sin ninguna de esas claves devuelve error. El descriptor se traduce a las mismas estructuras de `jsql` (`Query`, `Command`, `Define`), así que sigue las reglas de `SourceField`, `RETURNING` y triggers de las secciones anteriores.

En `from` va el modelo, como objeto `{database, schema, model}` (forma estándar) o como texto `schema.tabla`; sin esquema, la tabla se busca en todos los esquemas de la `DB` y debe ser única.

### 8.1 Comandos

Cada comando va bajo una clave con su nombre y contiene:

| Clave                                         | Uso                                                                                                                                                                | Comandos                             |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------ |
| `from`                                        | Modelo destino: `{database, schema, model}` o `schema.tabla`.                                                                                                      | todos                                |
| `data`                                        | Valores a guardar: un objeto (`insert`, `update`, `upsert`) o una lista de objetos (`bulk`). Los campos sin columna van a `SourceField`.                           | `insert`, `update`, `upsert`, `bulk` |
| `where`                                       | Condiciones con el formato de §6. En `upsert` nunca puede estar vacío.                                                                                             | `update`, `delete`, `upsert`         |
| `limit`                                       | Máximo de filas afectadas: sin enviar, `DB_RECORD_LIMIT` con tope de 1000; `0`, todas las que cumplen `where`.                                                     | `update`, `delete`, `upsert`         |
| `before_insert`, `after_insert`               | Listas de scripts JavaScript (goja) que corren antes y después de cada insert; cada uno es el código como string o `{name, code, language, version, description}`. | `insert`, `bulk`, `upsert`           |
| `before_update`, `after_update`               | Ídem, antes y después de actualizar cada registro.                                                                                                                 | `update`, `upsert`                   |
| `before_delete`, `after_delete`               | Ídem, antes y después de eliminar cada registro.                                                                                                                   | `delete`                             |
| `before_insert_update`, `after_insert_update` | Ídem, tanto si el `upsert` inserta como si actualiza.                                                                                                              | `upsert`                             |

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
- **Definición**: `define` con la estructura `Define` (§4.1) en JSON (`schema`, `name`, `columns`, `primary_keys`, `source_field`, `details`, `masters` —también se lee `master`—, `rollups`…). Define el modelo, crea la tabla si no existe y devuelve la definición del modelo.

## 9. Drivers

El driver recibe un `Model` para el DDL, un `Query` para el SQL de consulta y un `Command` para el SQL de comando:

```go
type Driver interface {
	UseSchema() bool                                                    // true: tabla schema.nombre; false: schema_nombre
	CreateDB(connection *ConnectParams, timeout ...time.Duration) error // crea la base de la conexión si no existe
	DropDB(db *DB, timeout ...time.Duration) error                      // elimina la base de los params si existe
	Connect(db *DB, timeout ...time.Duration) (*sql.DB, error)
	ExistModel(db *sql.DB, model *Model, timeout ...time.Duration) (bool, error)
	Load(model *Model, timeout ...time.Duration) (string, error)        // DDL: tablas y sus elementos
	Drop(model *Model, timeout ...time.Duration) error                  // elimina la tabla del modelo
	Query(query *Query, timeout ...time.Duration) (string, error)       // SQL de consulta
	Command(command *Command, timeout ...time.Duration) (string, error) // SQL de comando: INSERT, UPDATE, DELETE
}
```

- Cada driver vive en `jsql/drivers/<nombre>/`, se registra en `init()` con `jsql.Register(nombre, driver)` y se importa por efecto lateral (`_ "github.com/cgalvisleon/et/jsql/drivers/postgres"`).
- El driver solo **genera SQL** (y abre la conexión); ejecutarlo, manejar transacciones y correr los triggers es trabajo del núcleo.
- **Timeout:** cada método recibe un `timeout` opcional y el driver arma su contexto con `jsql.TimeoutContext(timeout...)`: sin timeout o con `0` no hay límite, así que la operación nunca falla por timeout; con un valor mayor falla con `context.DeadlineExceeded` al vencerse. El núcleo pasa el `Timeout` de `ConnectParams` (`DB.timeout`). `Load`, `Query` y `Command` solo generan SQL y no lo usan.
- `CreateDB` y `DropDB` (expuestos como `jsql.CreateDB(connection, timeout...)` y `jsql.DropDB(db, timeout...)`, que antes cierra el pool) son idempotentes. En postgres, mysql y mssql crean/eliminan la base `database` desde la base del servidor (`postgres`, sin base, `master`), y `Connect` usa `CreateDB`; en sqlite crean/borran el archivo (con sus `-wal` y `-shm`); en oracle el servicio no se crea ni se elimina desde una conexión: `CreateDB` solo comprueba que responde y `DropDB` devuelve `MSG_DRIVER_CANNOT_DROP_DB`. Probado en los cinco drivers (`TestCreateDropDB`).
- **Nombre de tabla** (`Model.Table`): lo asigna el núcleo al crear el modelo y en `Model.Init`, según `Driver.UseSchema()`. Con esquemas (postgres, mssql) es `schema.nombre`; sin ellos (sqlite, mysql, oracle) es `schema_nombre`, en la base o el usuario de la conexión. El esquema es obligatorio en `Define` (`MSG_SCHEMA_REQUIRED`). En sqlite, una tabla de antes de v1.0.36 (llamada solo `nombre`) se renombra a `schema_nombre` al iniciar su modelo. Los drivers arman el SQL con `Table` (`From.Table`); mssql cita sus partes como `[schema].[nombre]`.
- La conexión se describe con `jsql.Connection` (`ID`, `SetId`, `GetParams`, `SetDatabase`, `GetDatabase`, `SetPassword`; los drivers la implementan completa; `josefina` usa la de Postgres, `PgConection`), creada según `DB_DRIVER` en `conection.go`.

| Driver                           | Constante        | Estado                                                                 |
| -------------------------------- | ---------------- | ---------------------------------------------------------------------- |
| `postgres` (`lib/pq`)            | `DriverPostgres` | Completo.                                                              |
| `sqlite` (`modernc.org/sqlite`)  | `DriverSqlite`   | Completo; `RIGHT JOIN` y `FULL JOIN` nativos (SQLite 3.39 o superior). |
| `oracle` (`go-ora/v2`)           | `DriverOracle`   | Completo, para Oracle 19c o superior (probado en 23ai Free). Ver §9.1. |
| `mysql` (`go-sql-driver/mysql`)  | `DriverMysql`    | Completo, para MySQL 8.0 o superior (probado en 8.4). Ver §9.2.        |
| `mssql` (`microsoft/go-mssqldb`) | `DriverMssql`    | Completo, para SQL Server 2022 o superior (probado en 2022). Ver §9.3. |
| `josefina` (`lib/pq`)            | `DriverJosefina` | Completo, sobre el protocolo de Postgres de Josefina con `language=jql`. Ver §9.4. |

### 9.1 Particularidades del driver Oracle

- **Nombres**: tablas y columnas se crean como identificadores entre comillas y en minúscula (`"public_users"."_source"`), así conservan el nombre del modelo y admiten `_source` o `_idx`. En Oracle un esquema es un usuario, así que jsql no lo usa: la tabla es `schema_nombre` en el esquema de la conexión.
- **Tipos**: el JSON se guarda en `CLOB` con `CHECK (... IS JSON)`; `BOOL` es `NUMBER(1)`, que en los resultados JSON vuelve como `true`/`false`; `ANY` es `VARCHAR2(4000)`.
- **Comandos**: Oracle no devuelve filas con `RETURNING`, así que `Command` genera un bloque PL/SQL que ejecuta el DML, recoge los `ROWID` afectados y devuelve esas filas con `DBMS_SQL.RETURN_RESULT`. El `DELETE` abre el cursor antes de borrar.
- **`SourceField`**: el `UPDATE` fusiona los atributos con `JSON_MERGEPATCH`, que conserva los hermanos anidados; un valor `null` en `data` **borra** la llave, en lugar de guardar `null` como en Postgres. Por la misma razón, las columnas con valor `NULL` no aparecen en el `result`.
- **Literales**: los textos de más de 1000 caracteres se parten en `TO_CLOB('…') || TO_CLOB('…')` para no pasar el límite de 4000 bytes por literal (`ORA-01704`). Oracle guarda `''` como `NULL`.
- **Consultas**: `LIKE` no distingue mayúsculas (`UPPER(x) LIKE UPPER(v)`); la paginación usa `OFFSET … ROWS FETCH NEXT … ROWS ONLY`; `ON UPDATE CASCADE` no existe y se omite; no se crean índices sobre LOB ni un segundo índice sobre una columna ya indexada.

### 9.2 Particularidades del driver MySQL

- **Esquemas**: en MySQL un esquema es otra base de datos, así que jsql no lo usa: la tabla es `schema_nombre` en la base de la conexión (`Load` no crea esquemas). Tablas y columnas van entre backticks. `Connect` crea la base de `DB_NAME` si no existe.
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

### 9.4 Particularidades del driver Josefina

- Paquete `jsql/drivers/josefina` (`DriverJosefina`): Josefina (base de datos de documentos del espacio de trabajo) habla el protocolo de PostgreSQL en su `PG_PORT` y entiende SQL con su lenguaje `jql`, que la conexión elige con `options=-c language=jql`. El driver embebe al de Postgres: el SQL (DDL con `_source JSONB`, consultas, comandos con `jsonb_set`, `jsonb_build_object` y `||`) es el mismo, y la conexión es la de `lib/pq` con `PgConection` (por omisión `sslmode=disable`; usar `require` con un usuario de Josefina sin verificador SCRAM).
- Josefina no crea ni borra bases ni modelos por SQL: `Connect` y `CreateDB` solo comprueban que la base exista (se crea con la API de Josefina), `DropDB` responde `MSG_DRIVER_CANNOT_DROP_DB` (como Oracle) y `Drop` de un modelo responde `josefina.ErrCannotDrop`.
- Lo que PostgreSQL verifica y Josefina no (el padre de una llave foránea, una llave primaria compuesta repetida) lo verifica el lenguaje de Josefina al insertar, leyendo antes de escribir.
- Probado con `jsql/drivers/test` contra un nodo de Josefina (`JOSEFINA_HOST`, `JOSEFINA_PORT` (5433, el clúster de docker compose), `JOSEFINA_SSLMODE` (`require`), `JOSEFINA_USER`, `JOSEFINA_PASSWORD`, `JOSEFINA_DATABASE`): `TestCatalog` 102 de 102 (el caso de `RemoveModel` espera `ErrCannotDrop`), `TestInsertUpdate`, `TestJoinGroupHaving`, `TestFieldValue` y `TestCreateDropDB`. Cada corrida usa un schema nuevo (`jsql_t<n>`): Josefina no borra schemas.

## 10. Conexión y configuración

`jsql.Load()` (base `DB_NAME`) o `jsql.LoadTo(dbName, hostName...)` leen el entorno con `envar`; `jsql.ConnectTo(ConnectParams)` recibe los parámetros explícitos.

| Variable | Uso | Defecto |
| --- | --- | --- |
| `DB_DRIVER` | Driver: `postgres`, `sqlite`, `oracle`, `mysql` o `mssql` | `postgres` |
| `DB_HOST`, `DB_PORT` | Servidor | `localhost`; 5432 (postgres), 1521 (oracle), 3306 (mysql), 1433 (mssql) |
| `DB_USER`, `DB_PASSWORD`, `DB_NAME` | Credenciales y base (en Oracle, el _service name_, p. ej. `FREEPDB1`; en sqlite, el archivo) | `test` (`root` mysql, `sa` mssql) / `josephine` (`josephine.db` sqlite) |
| `DB_SSLMODE` | TLS (postgres) | `disable` |
| `DB_SSL`, `DB_SSL_VERIFY` | TLS (oracle) | `false`, `true` |
| `DB_APP_NAME` | Nombre de la aplicación en la conexión (postgres, sqlite) | `josephine` |
| `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_LIFETIME`, `DB_POOL_IDLE_TIME` | Pool (minutos en `LIFETIME` e `IDLE_TIME`); todas las conexiones. Sin ellos: sqlite 10, los demás los del driver (3, 1, 30, 2) | 10 / del driver |
| `DB_RECORD_LIMIT` | Límite de filas por consulta | 1000 |
| `DB_IS_DEBUG` | Registra el SQL generado | `false` |
| `MAX_AUDIT_LOG` | Entradas de auditoría por modelo | 1000 |

## 11. Utilidades

- `jsql.DefineSeries(db, schema)` → `*Series`: consecutivos y códigos (`SetSeries`, `GetSeries`, `DeleteSeries`, `GenSerie`, `GenValue`; `GenSerieTx`/`GenValueTx` dentro de una transacción, la que debe usar un trigger por script: en sqlite, escribir fuera de ella espera para siempre la única conexión). En scripts: `genSerie(tag, tx?)`, `genValue(tag, tx?)` y `db.series(schema)`.
- `Bind(source, vars)` (`helpers.go`): aplica variables `{{nombre}}` a un json (`"{{x}}"` toma el valor tal cual; dentro de un texto, su texto; una lista se expande dentro de otra; `{{user.id}}` es una ruta; las desconocidas quedan igual). `BindWrapper()` es su versión para goja (`jsql.bind`). No es para SQL escrito a mano.
- Helpers (`helpers.go`): `SQLParse` (reemplaza `$1`…`$N` por los argumentos escapados en una sola pasada; `$10` no se confunde con `$1`), `Quoted`, `EscapeSQLString`, `RowsToItems`, `ArgWhitAs` (`campo:alias`), `ArgWhitSchema` (`schema.tabla`), `AddAuditLog`.
- Estados (`define.go`): `ACTIVE`, `ARCHIVED`, `CANCELED`, `OF_SYSTEM`, `FOR_DELETE`, `PENDING`, `APPROVED`, `REJECTED` (+ `IN_PROCESS`, `FAILED`).

## 12. Brechas entre la especificación y el código

Revisado el 2026-10-12 contra el código y la batería de `jsql/drivers/test` (`go test -count=1 -v .`), con los cinco motores disponibles: postgres 15, sqlite, Oracle 23ai Free (`gvenzl/oracle-free:slim`), MySQL 8.4 y SQL Server 2022, en contenedores desechables con las credenciales por defecto de la batería (SQL Server necesita la base `jsql` creada antes).

| Batería | sqlite | postgres | oracle | mysql | mssql |
| --- | --- | --- | --- | --- | --- |
| `TestCatalog` (103 casos) | 102 pasan, 1 omitido | 102 pasan, 1 omitido | 102 pasan, 1 omitido | 102 pasan, 1 omitido | 102 pasan, 1 omitido |
| `TestInsertUpdate`, `TestJoinGroupHaving` | pasan | pasan | pasan | pasan | pasan |
| `TestWrapper` | pasa | pasa | — | — | — |
| `TestCreateDropDB` | pasa | pasa | pasa (§9: solo comprueba el servicio) | pasa | pasa |

El caso omitido es `Load`/`LoadTo`, que lee el entorno. `go build ./...` y `go vet ./...` pasan. `TestCatalog` solo escribe `result/` con `-update` (B7).

**Cerradas el 2026-10-12:**

- **B3:** `Define` lee los maestros en `masters` (como `details` y `rollups`) y sigue aceptando `master` (`Define.UnmarshalJSON`).
- **B4:** la batería corrió completa contra SQL Server 2022 por primera vez: pasa.
- **B7:** `TestCatalog` solo escribe `result/*.json`, `result/*.sql` y `result/summary.md` con `-update`; un motor no disponible conserva en el resumen su `result/<driver>.json` anterior.
- **B8:** `TestCreateDropDB` corre en los cinco drivers (en sqlite `DropDB` ahora respeta un timeout vencido; en Oracle comprueba que `CreateDB` responde y `DropDB` devuelve `MSG_DRIVER_CANNOT_DROP_DB`).
- **B9:** `LoadDb` reconstruye la conexión desde `params` (su `driver` es el tipo de conexión, así funciona con un driver registrado con otro nombre), la `DB` con `newDB`, sus esquemas, modelos y `audit_log`. No la inicia: falta `Init` para conectar.
- **B10:** `et.SafeData` se serializa como lista (y lee también la forma vieja `{data: [...]}`); `ToJson` de un modelo y de la base dan `audit_log` como lista (`SafeData.All`).
- **B11:** `conn.id` existe en todos los drivers, como dicen el test y `jsql.d.ts`.
- **B12:** §9 se corrigió: el esquema es obligatorio en `Define`. En sqlite, una tabla con el nombre de antes de v1.0.36 (`nombre`) se renombra sola a `schema_nombre` al iniciar su modelo si la nueva no existe (`Sqlite.ExistModel`), sin perder datos.
- **B13:** postgres, oracle, mysql y mssql leen `DB_POOL_MAX_OPEN`, `DB_POOL_MAX_IDLE`, `DB_POOL_LIFETIME` y `DB_POOL_IDLE_TIME` (`jsql.Pool`, embebido en sus conexiones) y los pasan en `GetParams` cuando valen más de 0; sin ellos, el driver usa sus valores (3, 1, 30 y 2). sqlite sigue con los suyos (10).

**Revisado el 2026-10-14** (sin correr la batería; `go build` y `go vet` pasan): `Model`/`Query`/`Command` suman `WhereByJson([]et.Json)` (condiciones en JSON, `et.ToWheRes`); `Query.AddCondition` y `Command` usan `et.AddCondition` (pone `AND` si la condición no trae conector y no es la primera); en el wrapper, `where` de modelo, consulta y comando recibe un objeto o una lista (`toWhere`) y se quitaron `and`/`or` (`jsql.d.ts`: `JsqlWhereItem`); `Wrapper` suma `toJsMap` (`ToJsMapWrapper`); `update`/`delete` propagan `Debug` a la consulta de filas. Se detectó B14.

**Revisado el 2026-10-15** con la batería completa en los cinco motores (contenedores desechables): `TestCatalog`, `TestInsertUpdate`, `TestJoinGroupHaving`, `TestCreateDropDB` y el nuevo `TestFieldValue` pasan; `TestWrapper` falla en sqlite y postgres por B14 (ver abajo). Se corrigió la recursión `Model.Query` ↔ `Model.query` (desde `454e6bc` todo `model.Query(json)` reventaba la pila).

**Revisado el 2026-10-27** (condiciones en rollups): `Rollups` y `QueryRollups` suman `Where []*et.Condition` (JSON como lista de condiciones), `DefineRollup(..., where ...*et.Condition)`, `DefRollup.Where`, `Rollups.QueryRollups()` (lo usan los cinco drivers) y `where` en `defineRollup` del wrapper. `TestCatalog` suma dos casos en §3 (where con valor fijo y con campo de la fila; `OR` rechazado y round trip JSON): pasa en sqlite, postgres y josefina; oracle, mysql y mssql no estaban disponibles (omitidos). Se abre B15. El mismo día se reagruparon los archivos del paquete (24 → 13, solo se movió código): `conection.go` (hoy `connection.go`) absorbe `driver.go`, `tester.go` y `jsql.go`; `db.go` absorbe `schema.go`, `tx.go` y `event.go`; `define.go` absorbe `column.go`; `model.go` absorbe `trigger.go`; `relation.go` (antes `detail.go`) absorbe `master.go`; `query.go` absorbe `query_star.go`; `helpers.go` absorbe `condition.go`. Con eso, `TestCatalog`, `TestInsertUpdate`, `TestJoinGroupHaving`, `TestFieldValue`, `TestCreateDropDB`, `TestStepScript` y `TestWrapper` pasan en sqlite, postgres y josefina (`TestWrapper` solo corre en sqlite y postgres).

**Revisado el 2026-10-27** (mejoras de la revisión del paquete): `DB.Schemas` y el `AuditLog` con lock (`getModel` ya no crea el esquema que busca: solo `getSchema` al definir); las definiciones de un modelo se serializan (`muDef`) y reemplazan sus mapas y su slice de columnas en vez de escribir en ellos (copia al escribir); relaciones por lotes y errores propagados (§2.3); `quoted` del núcleo con enteros sin signo, punteros y JSON citado para listas y tipos desconocidos (antes iban con el `%v` de Go); los cálculos `CALC` reutilizan un runtime de goja por consulta (`jwf.Runner`); `Rollups.OrderBy`; el modelo escribe sus maestros en `masters` y también en `master` (lo lee core-studio) y lee los dos; `conection.go` pasó a `connection.go`. Pruebas unitarias sin base en `jsql/jsql_test.go` (`go test -race ./jsql/`). La batería pasa en sqlite, postgres y josefina; oracle, mysql y mssql no estaban disponibles. Sigue sin haber agrupación general de condiciones con paréntesis en un `where` (no hay forma de pedirla en la API).

**Revisado el 2026-10-27** (revisión del núcleo `et`, ver `et/spec.md` §7): un `where` JSON con una condición sin conector después de la primera armaba SQL inválido (`... AND x = 1 A.y = 2`): `et.ToConditions` le pone ahora `AND`. `between`/`not_between` con `[min, max]` o `{min, max}` desde JSON devolvían `""` (SQL roto): los cinco drivers leen el rango con `et.ToBetween` y uno ilegible da `1 = 0`. El `AuditLog` del `DB` y del modelo se recorta con `et.SafeData.Trim` (antes borraba la entrada recién agregada al llegar a `MAX_AUDIT_LOG`). Nuevo `TestMemoryMatchesSql` (`drivers/test/evaluate_test.go`): el mismo where da las mismas filas en SQL y en memoria (precedencia, sin conector, `like`, `between`); con toda la batería pasa en los seis motores: sqlite, postgres, josefina, oracle, mysql y mssql (contenedores desechables `jsql-oracle`, `jsql-mysql`, `jsql-mssql`).

**Brecha:** ninguna abierta. Cerradas el 2026-10-27: B5 (driver para josefina, §9.4), B14 y B15.

**B15, cerrada el 2026-10-27:** el `where` de un rollup rechazaba `OR` porque los drivers no agrupan condiciones y `llave AND a OR b` mezclaba filas de otros padres. Ahora las llaves se repiten en cada término `OR` (`rollupConditions`, §2.3); `TestCatalog` («DefineRollup con where or») lo comprueba por lotes y fila por fila.

**B14, cerrada el 2026-10-27:** un `where` de un script perdía su condición. `et.ToWhere` tomaba los valores de un objeto `{campo: {op: valor}}` (perdía el nombre del campo) y no reconocía un `*et.Condition` (`jsql.eq(...)`), y `whereByJson` ignoraba el error de `ToWheRes`: la consulta leía todas las filas y un `update`/`delete` tocaba todas. Ahora `ToWhere` da una condición por llave del objeto (con `where` primero), acepta `*Condition`, `[]*Condition`, `WheRes` y listas de cualquiera de ellos, y `Query`/`Command.whereByJson` guardan el error y la operación falla al correr (`Command.err`). `TestToWhere` (`et/towhere_test.go`) y `TestWrapper` pasan en sqlite y postgres.

