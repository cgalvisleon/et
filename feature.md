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
	AGG      TypeColumn = "agg"
```

- Column: corresponde a las columnas que se crean en la tabla.
- Atrib: corresponde a los atributos que no tienen una columna propia y se guardan dentro del campo SourceField.
- Detail: da soporte a relaciones maestro-detalle. Define el modelo del detalle, las keys que unen el maestro con el detalle, los campos que se muestran y cuántos registros se muestran.
- Master: da soporte a relaciones 1 a 1 a través de una tabla intermedia. Define el modelo destino, el modelo puente, las keys del maestro al puente y del puente al destino, los campos que se muestran y cuántos registros se muestran.
- Rollup: da soporte a consultas hacia modelos que devuelven un solo registro. Por ejemplo, el atributo tp_documento, cuyo valor puede ser CC, NIT o RUT, tiene su significado en la tabla Tipo_documentos con los campos id y title. Un rollup de tipo RollupRow con Select []string{"title"} hace una consulta con limit 1 de la columna title y la asigna al atributo cuyo nombre es la llave del map[string]\*Rollups. Si es RollupObject con Select []string{id, title}, devuelve un objeto que se asigna a ese mismo atributo. También existen RollupCount, RollupSum, RollupAvg, RollupMin y RollupMax, que calculan un count, sum, avg, min o max sobre el modelo To y asignan el resultado al atributo.
