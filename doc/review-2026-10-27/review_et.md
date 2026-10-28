# Revisión et/ (vet y staticcheck limpios)

## A. Reglas duplicadas (pequeñas)
1. Fechas: 3 listas de layouts — json.go:432 ValTime (6 layouts), evaluate.go:643 dateLayouts (4), condition.go:75 Time() (solo RFC3339). Unificar en una `dateLayouts` en json.go junto a ValTime; ValTime → parseDate. ~12 líneas.
2. Separador "->" a mano en 5 sitios: json.go:1165 Set, condition.go:193 Value.Fields, evaluate.go:59 pathValue, where.go:576/578 getField. `const pathSep = "->"`. ~6.
3. `Time(val) *time.Time` condition.go:75: público, 0 usos. Deprecated + delegar en timeOf. ~16.
4. "sin conector = AND" en 3 sitios: condition.go:995 AddCondition (fuente), :710 ToConditions (usa), where.go:129 Wheres.Add (reescribe a mano) → usar AddCondition. ToWheRes (:949) no la aplica; 6 llamadores jsql compensan → comentario. ~6.
5. condition.go:42 ToOperator arma el mapa en cada llamada → var de paquete; TypeValues (:122) sin FIELD → agregar. Documentar que desconocido = EQ. ~8.
6. Function.Str (condition.go:292) y ToFunction (:326): dos switch espejo de 14 → tabla `functionNames`. ~60→25.
7. evaluate.go:281 fieldValue vs pathValue (:58): dos reglas de lectura; Value.Fields da [] si Type != TEXT (nombre >255 = MEMO). fieldValue → pathValue, o comentario en Value.Fields. Value.Fields pública (josefina jdb/query.go:1115), no cambiar firma.
8. where.go:604 hidden() = copia de Json.Hidden (json.go:1237). Usar item.Hidden en addItem (:309, :312). −20.
9. json.go:1085 Get repite ValAny (:282); en default pone nil y sigue iterando. Get → ValAny(nil, keys...). −30.
10. marshal→decodeJSON→Json copiado en result.go:46, :399, :951, where.go:109. Privada `jsonOfValue` en json.go. −25.
11. equalScalar (json.go:971) vs equalsAny (evaluate.go:1020): equalsAny devuelve error siempre nil (4 llamadores); unificar: equalsAny bool con caso bool; equalScalar lo usa. ~20.
12. Desempaquetado de valor repetido en applyOpEq (:321), applyOpOrdered (:370), applyOpLike (:443); closure invalidType solo da false (:439,:504). Privada conditionScalar. OJO: eq con objeto da false, like lo desempaqueta — mantener matiz o documentar. −45; test antes.
13. ValArray json.go:521 (~85 líneas) con 9 case []T que ya resuelve listOf (evaluate.go:166). −50.

## B. Nombres, comentarios, orden (riesgo nulo)
14. condition.go:639 closure `condition` oculta func de paquete `condition` (:767); fldCondition :658 `if result != nil {return result}; return result`. Renombrar conditionOf/firstConditionOf.
15. Comentarios que mienten: condition.go:532-600 Year..Second dicen "ExtractYear"; :488 Exp dice Value (es Aggregate). json.go:937 Update @return error (no devuelve); :997 Compare @return bool (Json); :451 ValDuration @return string; :786 Array @return []Json (es []interface{}; igual result.go:278, :714); :1175 Delete @param key string (es []string; result.go:346, :812); :1675 ToString @param vals []Json; :197 ToMap; :32 Scan; :1137/:1142 español en SetNested + if redundante. result.go:27 ToObject @return []byte; :197/:206 ToBase64/FromBase64 @return []string; :826 Items.Exist "ExistKey"; :611 param dx→idx. where.go:58 Wheres comentado "Where"; :97 IsDebug "Returns" (lo activa); :75 newWhere @param; :553 "itmes"; :757 merge "as"; :651 MergeToMap es producto cartesiano. "@return *Where" (tipo inexistente) en json.go:1256, result.go:366, :926, :992, where.go:554.
16. Exportados sin comentario (~90 líneas): condition.go Operator(18), Operator.Str(38), ToOperator(42), TypeData(92), .Str(118), TypeValues(122), IsTypeValue(145), Value(151), BetweenValue(268), Function(273), ToFunction(292), Function.Str(326), Aggregate(361), Connector/NaC/ToConnector/Str(604-624), Condition(626), WheRes(942), Command(1061). result.go Item(9), NewItems(378), List.ToByte(943). where.go Iterator(11), JoinType(18), Join(46), OrderField(52), Source(492). safemap.go SafeMap(8)+Get/Set/Delete, SafeData(46), NewSafeData(51), Len, Get, Add, Delete. json.go:26 Json.
17. Narran: evaluate.go:1156 firstMapValueSorted (5 líneas→1); json.go:946 deepEqualValue (4); json.go:112-115 ScanRows (4→2).
18. Mover (solo gofmt): asJson/sameMap goja.go:119/:128 → json.go; parseDate/dateLayouts evaluate.go:643 → json.go; Field.Str/ValueOf/pathValue/fieldRef evaluate.go:19-119 → condition.go junto a Field; jsonValue/toConditionValue evaluate.go:127-144 → condition.go; getBetweenRange evaluate.go:1103 → junto a ToBetween (:635); Command condition.go:1061 → arriba con tipos; WheRes :942 junto a Condition; públicas ApplyToValue/ApplyToObject/evaluate/EvaluateObject/EvaluateValue evaluate.go:697-822 → inicio; json.go deepEqualValue/equalScalar :953-994 → bloque privadas.
19. Muerto (0 usos fuera de et/et): Json.GetItem json.go:1276 (ignora receptor), Json.Query :1267, ArrayBytes :799, Condition.ApplyToIndex evaluate.go:763, Wheres.IsDebug+isDebug where.go:71/:100, Wheres.Workers :69, Source.Data :533, Prefixer :688, Value.Raw condition.go:160, Exp :492, ToConnector :612, ErrorDataNotFound :15, MSG_OPERATOR/VALUE/CONNECTOR_NOT_FOUND msg.go:7-9, SafeMap (solo tests). Públicos → Deprecated ahora; privados borrar. −80.
20. Erratas: IsDeferent json.go:1048 (1 uso jsql/model.go:864) → IsDifferent + alias Deprecated. Json.Exist (:1200) = IsExist (:217); Exist 0 usos → delega + Deprecated. ToWheRes/WheRes: 6 usos jsql → solo comentar. Str() vs String() — anotar, no cambiar.
21. Índices: Items.One 1-based (result.go:844) vs Wheres.One 0-based (where.go:458) → documentar en cada One.
22. spec §3.2: coerción time↔texto solo en escalar; in/between/listas exigen time.Time ambos lados. 3 líneas.

## C. Escalabilidad
23. Operador nuevo toca 12 archivos (condition.go:20/:42/:807, evaluate.go:706, spec, 5 drivers query.go, wrapper.go, jsql.d.ts, evaluate_test.go). Propuesta: comentario-checklist en Operator(:18), Function(:273), TypeData(:92). ~15.

## D. No recomendado ahora
24. Funciones largas: ToJS goja.go:146, Wheres.All where.go:384, ToCondition condition.go:638, valueType :220, applyFunction evaluate.go:210, ScanRows json.go:72, Joingy where.go:697, MatchLike evaluate.go:967, fromJS goja.go:251.
25. API: ApplyToValue por tabla; WheRes→Conditions; In/NotIn firma; Where dup de Eq; atribs→attribs; Str/String; Items.One base.

## Conservar
AddCondition + evaluate único; number/numberOf/compareNumbers; ValX(def, keys...)+at; parseJSON/decodeFresh/repairJSON; cabecera y puente único de goja.go.
