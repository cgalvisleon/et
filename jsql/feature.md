El objetivo de package jsql es ser un sqlBuilder que usa estructuras json como parámetros de entrada, con soporte para interpretar comandos sql como:

- select
- from
- join
- left join
- right join
- full join
- where
- and
- or
- group by
- having
- order by
- limit
- offset
- insert
- update
- delete

También soporta la definición y creación de bases de datos, esquemas y tablas, con la siguiente particularidad.

Los modelos definidos tienen un atributo llamado SourceField. Si no es "", contiene el nombre de un campo de tipo JSONB que guarda datos json. Por ejemplo, imagina un modelo llamado Users con los campos id, name y \_source, donde \_source es el SourceField. Si se hace un insert con este json:

```json
{ "id": 1, "name": "Cesar", "last_name": "Galvis" }
```

El valor de last_name no tiene una columna definida en el modelo, así que se almacena en la columna SourceField. Lo mismo ocurre con cualquier valor que no tenga una columna, y también con el json que se envía al update.

Además, los comandos Insert, Update y Delete incluyen RETURNING: Insert y Update devuelven los datos actualizados y Delete devuelve los datos eliminados.

Estos comandos también reciben un []Json con las condiciones, como este:

```json
{
  "where": [
    { "id": { "eq": 1 } },
    { "and": { "name": { "eq": "Cesar" } } },
    { "or": { "last_name": { "eq": "Galvis" } } }
  ]
}
```

En este caso el atributo last_name no tiene una columna asociada, entonces se asume que está en SourceField y debe buscarse como atributo en esa columna, por ejemplo: \_source->>"last_name".

Los select también se soportan en json y reciben un array []interface{}. Si está vacío, se une SourceField con un jsonb_build_object de los demás campos, por ejemplo: "\_source || jsonb_build_object('id', id, 'name', name)". Cuando no está vacío, se toman los valores de tipo Field y se crea un jsonb_build_object que los agrupa a todos.

Para facilitar estas operaciones existen los siguientes struct:

- Command, para definir comandos como Insert, Update, Delete y Upsert, que a partir del resultado de un where hace un Insert o un Update.
- Query, para las sql de tipo consulta o select.
- From, para el origen o ubicación de los modelos junto con su As o sinónimo.
- Field, para definir las columnas o atributos: columna es la que existe como columna en el modelo y atributo es la que se guarda en SourceField. También incluye el As o sinónimo.

Para permitir el uso de diferentes motores de base de datos existe la interfaz Driver:

```go
type Driver interface {
	Connect(ctx context.Context, db *DB) (*sql.DB, error)
	ExistModel(db *sql.DB, model *Model) (bool, error)
	Load(model *Model) (string, error)
	Query(query *Query) (string, error)
	Command(command *Command) (string, error)
}
```

Esta interfaz recibe un Model para definir el sql de tipo DDL que crea las tablas y sus elementos, un Query para definir el sql de tipo consulta, y un Command para definir el sql de tipo comando (INSERT, UPDATE, DELETE).

Las columnas tienen la siguiente estructura:

```go
type Column struct {
	Name       string      `json:"name"`
	TypeColumn TypeColumn  `json:"type_column"`
	TypeData   et.TypeData `json:"type_data"`
	Default    any         `json:"default"`
	model      *Model      `json:"-"`
}
```

TypeColumn puede tener estos valores:

```go
COLUMN   TypeColumn = "column"
	ATTRIB   TypeColumn = "atrib"
	DETAIL   TypeColumn = "detail"
	MASTER   TypeColumn = "master"
	ROLLUP   TypeColumn = "rollup"
	CALCFUNC TypeColumn = "calc_func"
	CALC     TypeColumn = "calc"
```

- COLUMN: corresponde a las columnas que se crean en la tabla.
- ATTRIB: corresponde a los atributos que no tienen una columna propia y se guardan dentro del campo SourceField.
- DETAIL: da soporte a relaciones maestro-detalle. Define el modelo del detalle, las keys que unen el maestro con el detalle, los campos que se muestran y cuántos registros se muestran.
- MASTER: da soporte a relaciones 1 a 1 a través de una tabla intermedia. Define el modelo destino, el modelo puente, las keys del maestro al puente y del puente al destino, los campos que se muestran y cuántos registros se muestran.
- ROLLUP: da soporte a consultas hacia modelos que devuelven un solo registro. Por ejemplo, el atributo tp_documento, cuyo valor puede ser CC, NIT o RUT, tiene su significado en la tabla Tipo_documentos con los campos id y title. Un rollup de tipo RollupRow con Select []string{"title"} hace una consulta con limit 1 de la columna title y la asigna al atributo cuyo nombre es la llave del map[string]\*Rollups. Si es RollupObject con Select []string{id, title}, devuelve un objeto que se asigna a ese mismo atributo. También existen RollupCount, RollupSum, RollupAvg, RollupMin y RollupMax, que calculan un count, sum, avg, min o max sobre el modelo To y asignan el resultado al atributo.
- CALCFUNC: da soporte a funciones de Go (CalcFunction) que se ejecutan cuando esta columna está incluida en el select.
- CALC: da soporte a scripts de JavaScript que se ejecutan con el paquete goja cuando esta columna está incluida en el select.

Las columnas de tipo DETAIL, MASTER, ROLLUP, CALCFUNC y CALC solo se ejecutan si se incluyen de manera literal en el select. Cuando el array del select está vacío, solo se incluyen las columnas de tipo COLUMN.

Comandos

Los comandos también se pueden describir en json. Cada comando se escribe bajo una clave con su nombre (insert, update, delete, upsert o bulk) y dentro indica el modelo en from (schema.tabla), los valores en data, las condiciones en where y, de forma opcional, triggers en javascript. Igual que los comandos del modelo, siguen la regla de SourceField: los valores con columna se guardan en su columna y el resto en SourceField. Todos devuelven los registros afectados (RETURNING).

- Insert: inserta un registro en el modelo indicado en from con los valores de data. La estructura json para un insert es la siguiente:

```json
{
  "insert": {
    "from": "public.users",
    "data": {
      "id": 1,
      "status": "active",
      "name": "Cesar"
    },
    "before_insert": ["code javascript...", "code javascript..."],
    "after_insert": ["code javascript...", "code javascript..."]
  }
}
```

before_insert y after_insert son listas de código javascript que goja ejecuta antes y después del insert. En el script, el registro nuevo está en NEW (OLD está vacío en un insert), y los cambios que before_insert haga en NEW son los que se guardan.

- Update: actualiza los registros que cumplen el where con los valores de data. Los atributos se fusionan en SourceField sin borrar los que ya existen. limit indica cuántos registros se actualizan como máximo, para no poner en riesgo la estabilidad de la base de datos con un where muy amplio: si no se envía, se usa un valor por defecto de máximo 1000 registros, y si se envía en 0 se actualizan todos los registros que cumplen el where. La estructura json para un update es la siguiente:

```json
{
  "update": {
    "from": "public.users",
    "data": {
      "name": "Cesar Galvis"
    },
    "where": [
      { "id": { "eq": 1 } },
      { "and": { "status": { "eq": "active" } } }
    ],
    "limit": 100,
    "before_update": ["code javascript...", "code javascript..."],
    "after_update": ["code javascript...", "code javascript..."]
  }
}
```

before_update y after_update son listas de código javascript que goja ejecuta antes y después de actualizar cada registro. En el script, OLD es el registro antes del cambio y NEW el registro con los valores de data aplicados; los cambios que before_update haga en NEW son los que se guardan.

- Delete: elimina los registros que cumplen el where y devuelve los datos eliminados. Igual que en update, limit indica cuántos registros se eliminan como máximo: si no se envía, se usa un valor por defecto de máximo 1000 registros, y si se envía en 0 se eliminan todos los registros que cumplen el where. La estructura json para un delete es la siguiente:

```json
{
  "delete": {
    "from": "public.users",
    "where": [
      { "id": { "eq": 1 } },
      { "and": { "status": { "eq": "active" } } }
    ],
    "limit": 100,
    "before_delete": ["code javascript...", "code javascript..."],
    "after_delete": ["code javascript...", "code javascript..."]
  }
}
```

before_delete y after_delete son listas de código javascript que goja ejecuta antes y después de eliminar cada registro. En el script, OLD es el registro que se elimina.

- Upsert: consulta si existe algún registro que cumpla el where. Si no existe, ejecuta un insert con los valores de data; si existe, ejecuta un update con data de los registros que lo cumplen. El where nunca puede estar vacío. Su estructura es la siguiente:

```json
{
  "upsert": {
    "from": "public.users",
    "data": {
      "name": "Cesar Galvis"
    },
    "where": [
      { "id": { "eq": 1 } },
      { "and": { "status": { "eq": "active" } } }
    ],
    "limit": 100,
    "before_insert": ["code javascript...", "code javascript..."],
    "after_insert": ["code javascript...", "code javascript..."],
    "before_update": ["code javascript...", "code javascript..."],
    "after_update": ["code javascript...", "code javascript..."],
    "before_insert_update": ["code javascript...", "code javascript..."],
    "after_insert_update": ["code javascript...", "code javascript..."]
  }
}
```

before_insert, after_insert, before_update, after_update, before_insert_update y after_insert_update son listas de código javascript que ejecuta goja. Los de insert solo se ejecutan cuando el upsert inserta, los de update solo cuando actualiza, y before_insert_update y after_insert_update se ejecutan en ambos casos.

- Bulk: inserta varios registros en una sola operación; data es una lista con un objeto por registro. La estructura json para una inserción a granel es la siguiente:

```json
{
  "bulk": {
    "from": "public.users",
    "data": [
      { "id": 1, "status": "active", "name": "Cesar" },
      { "id": 2, "status": "active", "name": "Ana" }
    ],
    "before_insert": ["code javascript...", "code javascript..."],
    "after_insert": ["code javascript...", "code javascript..."]
  }
}
```

before_insert y after_insert son listas de código javascript que goja ejecuta antes y después de insertar cada registro de la lista, igual que en insert.
