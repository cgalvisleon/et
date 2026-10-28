# Especificación del paquete `et`

Versión del documento: 1.2 (2026-10-27: §3.2 aclara que las fechas solo se convierten frente a un valor escalar; `IsDifferent` y `Exist` delega en `IsExist`; versión 1.1: archivos reagrupados, ver §8; `MatchLike` pública)

`et` es el núcleo de la librería: el tipo `Json`, los resultados (`Item`, `Items`, `List`), las condiciones (`Condition`) y su evaluación en memoria (`EvaluateObject`, `EvaluateValue`, `From(...).Where(...)`). Las mismas condiciones las traduce `jsql` a SQL; la regla de este documento es que **una condición da el mismo resultado en memoria que en SQL**. Lo comprueba `TestMemoryMatchesSql` (`jsql/drivers/test/evaluate_test.go`) en cada motor disponible.

## 1. `Json`

- `Json` es `map[string]interface{}`. Como cualquier mapa de Go, un `Json` nil no se puede escribir: `Set` sobre nil hace `panic`; hay que empezar con `Json{}`.
- `Set(key, val)`: `"a->b"` escribe una ruta anidada (`SetNested`), creando los objetos que falten. `SetNested` sin claves no hace nada.
- `Put(val)`: reemplaza el contenido (vacía y copia) sobre el mismo mapa, así lo ve quien lo comparte. `Put` de sí mismo, o sobre un `Json` nil, no hace nada.
- `Clone()`: copia **superficial**; objetos y listas anidados quedan compartidos. `DeepClone()` copia también `Json`, `map[string]any`, `[]Json` y `[]any` anidados; el resto de valores se conserva tal cual.
- `Compare`, `IsChanged`, `IsDeferent` comparan valores como JSON: textos, booleanos, nil y números directamente (`1` y `1.0` son iguales; enteros exactos), objetos y listas por su codificación JSON.

## 2. Lectura de JSON (`json.go`)

`parseJSON` decodifica con `UseNumber` (enteros más allá de 2^53 quedan `int64`, el resto `float64`) y repara texto dañado: saltos de línea crudos dentro de cadenas, texto escapado una vez más y doble codificación. Cada intento decodifica sobre un valor nuevo y solo lo asigna si tuvo éxito: un intento fallido no deja claves a medias.

## 3. Condiciones

Una condición es `{campo: {operador: valor}}`, precedida opcionalmente por un conector: `{and: {...}}`, `{or: {...}}` o `{where: {...}}` (sin conector). `ToConditions` arma la lista y **pone `AND` a toda condición sin conector que no sea la primera** (`AddCondition`), igual que `Query.AddCondition`, `Command` y `Wheres.Add`.

### 3.1 Precedencia

`AND` liga más que `OR`, como en SQL: `a OR b AND c` es `a OR (b AND c)`. La lista se evalúa como una disyunción de términos `AND` (`evaluate` en `evaluate.go`, único evaluador de `EvaluateObject`, `EvaluateValue` y `Wheres`). Una lista vacía es `true`. La evaluación se corta en cuanto el resultado no puede cambiar.

### 3.2 Operadores

| Operador | En memoria |
|---|---|
| `eq`, `neg` | Igualdad de textos, fechas, números (por valor, enteros exactos) o valores comparables del mismo tipo. Una lista como valor toma su primer elemento. |
| `less`, `less_eq`, `more`, `more_eq` | Orden de fechas, textos o números. Tipos distintos no comparan (`false`). |
| `like` | Ver 3.3. |
| `in`, `not_in` | El valor está (o no) en la lista. |
| `is`, `is_not` | Igualdad que acepta nil a ambos lados. |
| `null`, `not_null` | El valor es (o no) nil. |
| `between`, `not_between` | Ver 3.4. |

Un valor nil a la izquierda es `false` en todo operador salvo `null`, `is` y los negados (`neg`, `not_in`, `is_not`), que dan `true`. Diferencia conocida con SQL: allí `NULL <> x` y `NULL NOT IN (...)` no son verdaderos.

Fechas: un `time.Time` se compara con un texto en `RFC3339`, `RFC3339Nano`, `2006-01-02 15:04:05` o `2006-01-02`. La conversión solo se hace cuando el valor de la condición es un escalar (`coerceForComparison`): en `in`/`not_in` y `between`/`not_between`, la lista o el rango no se convierten, así que los dos lados deben ser `time.Time`.

### 3.3 `like`

Como en los cinco drivers de `jsql`: `%` es cualquier texto, `_` un carácter, `\` escapa el siguiente, y **no distingue mayúsculas**. Los comodines van en cualquier posición (`a%m%a`). `*` se acepta como alias de `%`, por compatibilidad con la versión anterior (que solo admitía `*` al inicio o al final y distinguía mayúsculas).

Diferencia conocida: sqlite solo ignora mayúsculas en ASCII; en memoria se ignoran también en el resto de Unicode (`Ñ`/`ñ`).

### 3.4 `between`

El rango es un `BetweenValue{Min, Max}`, un objeto `{min, max}` (o `{Min, Max}`, como se codifica un `BetweenValue`) o una lista `[min, max]`; `ToBetween` lo lee y `ToCondition` lo normaliza a `BetweenValue`. Los límites se incluyen. Como en SQL, un valor nil o un rango que no se puede comparar es `false` tanto en `between` como en `not_between`. En SQL, un rango ilegible da `1 = 0` (antes armaba SQL inválido).

### 3.5 Números

Los enteros se comparan **exactamente**, también más allá de 2^53 (IDs Snowflake) y con signo contra sin signo (`-1 < uint64(0)`). Un decimal sin parte fraccionaria cuenta como entero; uno con decimales compara como `float64`. `NaN` no compara. `json.Number` se lee como entero cuando lo es.

### 3.6 Referencias a campos

Un valor `{field: "nombre"}` (`et.FIELD`, `Fld`) se lee del mismo objeto (rutas `a->b`, agregaciones `sum(items->price)`, `extract_year(...)`). `ApplyToValue` con una referencia y sin objeto es `false`. Detalle en el `CLAUDE.md` de la librería («Referencias a campos»).

## 4. Consultas en memoria (`Wheres`)

`From(items, as...)` antepone el alias a cada clave (`a.` por defecto; `""` sin prefijo, las claves en minúsculas): las condiciones, el orden y las llaves de un `Join` usan esos nombres (`{"u.id": "o.user_id"}`). `Where`/`And`/`Or` agregan condiciones, `Join`/`LeftJoin`/`RightJoin`/`FullJoin` cruzan listas, `Asc`/`Desc` ordenan (estable), `Limit(page, rows)` pagina, `Select`/`Hidden` eligen claves. `AddResult` (antes `AdddResult`, que queda como alias obsoleto) agrega un resultado y dice si cabe otro.

## 5. `SafeData` y `SafeMap`

- `SafeData` es una lista con candado; se serializa como lista (`[...]`) y acepta `{data: [...]}` al leer. `Trim(max)` conserva las últimas `max` entradas (las más nuevas: es el recorte del `AuditLog` de `jsql`). `Set` ignora un índice fuera de rango y lo dice (`false`).
- `SafeMap` es un mapa con candado; `NewSafeMap()` lo crea y `Set` crea el mapa si falta.

## 6. Pruebas

```bash
go test -race ./et/                                                # json, goja, evaluate, condiciones, Wheres, SafeData
cd jsql/drivers/test && go test -v -run TestMemoryMatchesSql .     # memoria = SQL en cada motor disponible
```

`et/condition_test.go`: `ToWhere`, índices fuera de rango en `Items`, precedencia, números, `like`, cada operador, `Wheres`, `SafeData`, `Put`/`SetNested`/`DeepClone`, `deepEqualValue` y `parseJSON`.

## 7. Brechas

Ninguna abierta. Cerradas el 2026-10-27:

| Id | Brecha | Solución |
|---|---|---|
| E1 | En memoria `a OR b AND c` se evaluaba de izquierda a derecha | `evaluate` con precedencia SQL |
| E2 | Una condición sin conector después de la primera se ignoraba en memoria y rompía el SQL (`ToConditions`) | `ToConditions` usa `AddCondition` (= `AND`) |
| E3 | Enteros más allá de 2^53 comparados como `float64` | `compareNumbers` exacto |
| E4 | `like` en memoria usaba `*` y distinguía mayúsculas | `MatchLike` con `%`/`_`, sin mayúsculas |
| E5 | El `AuditLog` de `jsql` borraba la entrada recién agregada al llegar al tope | `SafeData.Trim` |
| E6 | `between` con `[min, max]` desde JSON armaba SQL inválido | `ToBetween` en `ToCondition` y en los drivers |
| E7 | `panic` en `SetNested` sin claves, `SafeMap` sin `Data`, `SafeData.Set` fuera de rango; `Put` de sí mismo vaciaba el mapa | Guardas en cada uno |

## 8. Archivos

Reagrupados el 2026-10-27 (de 13 a 8, más 4 de pruebas; solo se movió código):

| Archivo | Contenido |
|---|---|
| `json.go` | `Json`, lectura de JSON (`decodeJSON`, `parseJSON`, `intOf`, `floatOfAny`…), `Serialize`, `ToJson`, `ToString` |
| `result.go` | `Item`, `Items` (`at`: un índice fuera de rango da el valor por defecto), `List` |
| `condition.go` | Tipos (`Operator`, `TypeData`, `Value`, `Field`, `Aggregate`, `Connector`, `Condition`, `Command`), constructores, `ToCondition`, `ToConditions`, `AddCondition`, `ToWhere` |
| `evaluate.go` | Referencias a campos, `ApplyToValue`/`ApplyToObject`, `evaluate`, números exactos (`numberOf`, `compareNumbers`), `MatchLike`, `ToBetween` |
| `where.go` | `Wheres`, `Source`, `From`, joins y selección |
| `goja.go` | `ToJS`/`FromJS`/`JsonOf`/`GoArg` y los wrappers de scripts (`evaluateObject`, `evaluateValue`, `field`…) |
| `safemap.go`, `msg.go` | `SafeData`/`SafeMap`; mensajes |
