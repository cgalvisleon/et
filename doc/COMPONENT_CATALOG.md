# COMPONENT_CATALOG.md

> Catálogo de la API pública de `github.com/cgalvisleon/et`, **generado desde `go doc`** (funciones, métodos e interfaces exportadas). Para regenerarlo: `go doc -all ./<paquete> | grep '^func '`. Contexto narrativo en `LIBRARY_CONTEXT.md`, arquitectura en `ARCHITECTURE_SUMMARY.md`, reglas para IA en `AI_USAGE_GUIDE.md`.
>
> Se omiten `cmd/`, `create/` y `jcli/` (WIP huérfano).

Generado: 2026-10-10.

## `aws`

S3 y SMS.

Funciones y métodos:

```go
func NewS3AWS(params Params) (*S3AWS, error)
func (s *S3AWS) Delete(bucket, filePath string) (et.Item, error)
func (s *S3AWS) Download(bucket, filePath string) (et.Item, error)
func (s *S3AWS) Uploader(bucket, filename, contentType string, contentFile []byte) (et.Item, error)
func (s *S3AWS) UploaderB64(bucket, b64, fileName, contentType string) (et.Item, error)
func (s *S3AWS) UploaderFile(r *http.Request, bucket, folder, fileName string) (et.Item, error)
func NewSenderAWS(params Params) (*SenderAWS, error)
func (s *SenderAWS) SendSMS(contactNumbers []string, content string, params et.Json, tpMessage string) (et.Item, error)
```

## `brevo`

Email, SMS y WhatsApp transaccional/promocional.

Funciones y métodos:

```go
func SendEmail(sender et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tp string) (et.Items, error)
func SendEmailPromotional(sender et.Json, to []et.Json, subject string, htmlContent string, params et.Json) (et.Items, error)
func SendEmailTransactional(sender et.Json, to []et.Json, subject string, htmlContent string, params et.Json) (et.Items, error)
func SendSmsPromotional(sender, organisation string, contactNumbers []string, content string, params []et.Json) (et.Items, error)
func SendSmsTransactional(sender, organisation string, contactNumbers []string, content string, params []et.Json) (et.Items, error)
func SendWhatsapp(contactNumbers []string, templateId string, params []et.Json, tp string) (et.Items, error)
func SendWhatsappPromotional(contactNumbers []string, templateId string, params []et.Json) (et.Items, error)
func SendWhatsappTransactional(contactNumbers []string, templateId string, params []et.Json) (et.Items, error)
```

## `cache`

Redis. `Get*` devuelven `(valor, existe, error)`; clave inexistente = `def, false, nil`.

Interfaces:

```go
type Router interface {
	Protect(method, path string, handler func(http.ResponseWriter, *http.Request))
}
```

Funciones y métodos:

```go
func AllCache(search string, page, rows int) (et.List, error)
func Close()
func CollectionDelete(name, key string) error
func CollectionFind(name, key string) (string, bool, error)
func CollectionGet(name string) (map[string]string, error)
func CollectionPut(name, key, val string) error
func CollectionSet(name string, val map[string]string) error
func Decr(key string) int64
func DecrCtx(ctx context.Context, key string) int64
func Delete(key string) (int64, error)
func DeleteByPrefix(prefix string) error
func DeleteByPrefixCtx(ctx context.Context, prefix string) error
func DeleteCtx(ctx context.Context, key string) (int64, error)
func DeleteVerify(device string, key string) (int64, error)
func Empty(match string) error
func Exists(key string) bool
func ExistsCtx(ctx context.Context, key string) bool
func Expire(key string, expiration time.Duration) error
func ExpireCtx(ctx context.Context, key string, expiration time.Duration) error
func FromId() string
func Get(key, defaultvalue string) (string, bool, error)
func GetBool(key string, def bool) (bool, bool, error)
func GetBytes(key string) ([]byte, bool, error)
func GetCtx(ctx context.Context, key, def string) (string, bool, error)
func GetDuration(key string, def time.Duration) (time.Duration, bool, error)
func GetFloat(key string, def float64) (float64, bool, error)
func GetInt(key string, def int) (int, bool, error)
func GetInt64(key string, def int64) (int64, bool, error)
func GetItem(key string) (et.Item, bool, error)
func GetItems(key string) (et.Items, bool, error)
func GetJson(key string) (et.Json, bool, error)
func GetList(key string) (et.List, bool, error)
func GetObject(key string, dest any) (bool, error)
func GetStr(key, def string) (string, bool, error)
func GetTime(key string, def time.Time) (time.Time, bool, error)
func GetVerify(device string, key string) (string, bool, error)
func HDeleteCtx(ctx context.Context, key, atr string) error
func HGetCtx(ctx context.Context, key string) (map[string]string, error)
func HSetCtx(ctx context.Context, key string, val map[string]string) error
func HealthCheck() bool
func HttpAll(w http.ResponseWriter, r *http.Request)
func HttpDelete(w http.ResponseWriter, r *http.Request)
func HttpGet(w http.ResponseWriter, r *http.Request)
func Incr(key string, expiration time.Duration) int64
func IncrCtx(ctx context.Context, key string, expiration time.Duration) int64
func IncrDuration(key string, expiration time.Duration) int64
func IsLoad() bool
func LPush(key, val string) error
func LPushCtx(ctx context.Context, key string, val string) error
func LRange(key string, start, stop int64) ([]string, error)
func LRangeCtx(ctx context.Context, key string, start int64, stop int64) ([]string, error)
func LRem(key, val string) error
func LRemCtx(ctx context.Context, key string, val string) error
func LTrim(key string, start, stop int64) error
func LTrimCtx(ctx context.Context, key string, start int64, stop int64) error
func Load() error
func LoadRouter(r Router)
func ObjetAll(name string) ([]et.Json, error)
func ObjetDelete(name, key string) error
func ObjetGet(name, key string) (et.Json, bool, error)
func ObjetSet(name string, key string, obj et.Json) error
func Set(key string, val interface{}, expiration time.Duration) interface{}
func SetCtx(ctx context.Context, key, val string, expiration time.Duration) string
func SetD(key string, val interface{}, expiration int) interface{}
func SetH(key string, val interface{}, expiration int) interface{}
func SetM(key string, val interface{}, expiration int) interface{}
func SetObject(key string, val interface{}, expiration time.Duration) (interface{}, error)
func SetVerify(device, key, val string, expiration time.Duration) interface{}
func SetW(key string, val interface{}, expiration int) interface{}
func SetWithDuration(key string, val interface{}, expiration time.Duration) interface{}
func SetY(key string, val interface{}, expiration int) interface{}
func New() (*Conn, error)
func (s *Conn) Close()
func (s *Conn) HealthCheck() bool
func (s *Conn) Pub(channel string, message []byte) error
func (s *Conn) Sub(channel string, f func(*redis.Message))
func (s *Conn) Unsub(channel string) error
func (s Message) ToJson() (et.Json, error)
func (s Message) ToString() string
func CallMetrics(key string, limit int64) (Metrics, error)
func (s *Metrics) ToJson() et.Json
```

## `claim`

Claims JWT firmados HS256 con `SECRET`.

Funciones y métodos:

```go
func NewAuthenticationToken(app, device, userId, name string, payload et.Json, duration time.Duration) (string, error)
func NewToken(app, device, userId, tenantId, roleId, name string, payload et.Json, duration time.Duration) (string, error)
func NewClaim(duration time.Duration) *Claim
func ParceToken(token string) (*Claim, error)
func (s *Claim) SetPayload(payload et.Json)
func (s *Claim) ToJson() (et.Json, error)
```

## `cmds`

Etapas y pasos de comandos del sistema.

Funciones y métodos:

```go
func Load(fileName string) (*Stage, error)
func NewStage(name, description string) *Stage
func (s *Stage) AppendStep(step *Step) *Stage
func NewStep(name, description string) *Step
func (s *Step) AppendCmd(args string) *Step
func (s *Step) RunOS(idx int, args et.Json) ([]byte, error)
func (s *Step) RunSSH(idx int, args et.Json) ([]byte, error)
```

## `color`

Colores ANSI.

Funciones y métodos:

```go
func Black(str string) string
func Blue(str string) string
func Cyan(str string) string
func Gray(str string) string
func Green(str string) string
func Purple(str string) string
func Red(str string) string
func White(str string) string
func Yellow(str string) string
```

## `crontab`

Cron y tareas programadas guiadas por eventos; singleton por paquete con `Load(tag, store)`.

Interfaces:

```go
type Store interface {
	Set(collection, id, ownerId string, obj any) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	Query(query et.Json) (et.Items, error)
}
```

Funciones y métodos:

```go
func CronJob(tag, ownerId string, spec Cron, repetitions int, params et.Json, fn func(params et.Json) error) error
func HttpRemoveJob(w http.ResponseWriter, r *http.Request)
func HttpStartJob(w http.ResponseWriter, r *http.Request)
func HttpStopJob(w http.ResponseWriter, r *http.Request)
func Load(tag string, store Store) error
func ScheduleJob(tag, ownerId string, spec time.Time, params et.Json, fn func(params et.Json) error) error
func New(tag string, store Store) (*Crontab, error)
func (s *Crontab) Debug() *Crontab
func (s *Job) ToJson() et.Json
func (s *Job) ToString() string
```

## `csv`

Leer y escribir hojas de cálculo / CSV.

Funciones y métodos:

```go
func IndexOf(list []string, value string) int
func NewCsv(data []et.Json, columns ...Column) *Csv
func (s *Csv) ToFile(path string) error
func (s *Csv) ToHttp(w http.ResponseWriter, filename string) error
func (s *Csv) ToWriter(w io.Writer) error
func ReadCsv(data []byte, comma rune) (*CsvReader, error)
func ReadCsvFile(path string, comma rune) (*CsvReader, error)
func ReadCsvMultipart(fileData multipart.File, comma rune) (*CsvReader, error)
func (s *CsvReader) GetData(columns []string) ([]et.Json, error)
func (s *CsvReader) GetRows() ([][]string, error)
```

## `dt`

Objetos en Redis (`PRODUCTION=true`, por defecto) o solo en memoria.

Funciones y métodos:

```go
func Drop(key string) error
func GetObject(key string, dest any) (bool, error)
func Get(key string) *Object
func Up(key string, data any) Object
func (s *Object) Array() ([]any, error)
func (s *Object) ArrayDuration() ([]time.Duration, error)
func (s *Object) ArrayFloat() ([]float64, error)
func (s *Object) ArrayInt() ([]int, error)
func (s *Object) ArrayInt64() ([]int64, error)
func (s *Object) ArrayJson() ([]et.Json, error)
func (s *Object) ArrayStr() ([]string, error)
func (s *Object) ArrayTime() ([]time.Time, error)
func (s *Object) Bool() (bool, error)
func (s *Object) Byte() ([]byte, error)
func (s *Object) Duration() (time.Duration, error)
func (s *Object) Float() (float64, error)
func (s *Object) Int() (int, error)
func (s *Object) Int64() (int64, error)
func (s *Object) Item() (et.Item, error)
func (s *Object) Items() (et.Items, error)
func (s *Object) Json() (et.Json, error)
func (s *Object) List() (et.List, error)
func (s *Object) String() (string, error)
func (s *Object) Time() (time.Time, error)
func (s *Object) ToString() string
```

## `envar`

Variables de entorno con default, args de CLI y validación.

Interfaces:

```go
type Store interface {
	Get(name string, def interface{}) (interface{}, bool)
	Set(name string, value interface{})
}
```

Funciones y métodos:

```go
func ArgBool(name string, defaultVal bool) (bool, bool)
func ArgFloat64(name string, defaultVal float64) (float64, bool)
func ArgInt(name string, defaultVal int) (int, bool)
func ArgInt64(name string, defaultVal int64) (int64, bool)
func ArgStr(name, defaultVal string) (string, bool)
func Bool(name string) bool
func Float(name string) float64
func Get(name string, def interface{}) interface{}
func GetBool(name string, def bool) bool
func GetConfig() map[string]interface{}
func GetDuration(name string, def time.Duration) time.Duration
func GetFloat(name string, def float64) float64
func GetInt(name string, def int) int
func GetInt64(name string, def int64) int64
func GetStr(name, def string) string
func Int(name string) int
func Int64(name string) int64
func Load(store Store)
func Set(name string, value interface{}) interface{}
func SetBool(name string, value bool) bool
func SetBoolByArg(arg, name string, def bool) bool
func SetInt(name string, value int) int
func SetInt64(name string, value int64) int64
func SetInt64ByArg(arg, name string, def int64) int64
func SetIntByArg(arg, name string, def int) int
func SetNumber(name string, value float64) float64
func SetStr(name string, value string) string
func SetStrByArg(arg, name, def string) string
func Str(name string) string
func Validate(keys []string) error
```

## `ephemeral`

Mapa en memoria con expiración por instancia.

Funciones y métodos:

```go
func NewInstance(expiration time.Duration) *Instance
func (s *Instance) Del(key string)
func (s *Instance) Get(key string) (interface{}, bool)
func (s *Instance) Set(key string, value interface{})
```

## `et`

Tipo universal `Json`, resultados (`Item`, `Items`, `List`), condiciones y consultas en memoria (`From(...).Where(...)`).

Interfaces:

```go
type Iterator interface {
	Next() (Json, bool)
	As() string
	Add(item Json)
	LimitRows() int
}
```

Funciones y métodos:

```go
func EvaluateObject(obj Json, conditions []*Condition) bool
func EvaluateValue(value any, conditions []*Condition) bool
func IsTypeValue(v string) bool
func Serialize(src any) ([]byte, error)
func Time(val any) *time.Time
func ToString(vals interface{}) string
func Avg(value any) Aggregate
func Day(value any) Aggregate
func Exp(value any) Aggregate
func Float(value any) Aggregate
func Hour(value any) Aggregate
func Int(value any) Aggregate
func Max(value any) Aggregate
func Min(value any) Aggregate
func Minute(value any) Aggregate
func Month(value any) Aggregate
func Second(value any) Aggregate
func Sum(value any) Aggregate
func ToAggregate(s string) (Aggregate, bool)
func Year(value any) Aggregate
func And(field interface{}, operator Operator, value interface{}) *Condition
func Between(field string, min, max any) *Condition
func Eq(field, value interface{}) *Condition
func In(field string, value []interface{}) *Condition
func Is(field, value interface{}) *Condition
func IsNot(field, value interface{}) *Condition
func Less(field, value interface{}) *Condition
func LessEq(field, value interface{}) *Condition
func Like(field, value interface{}) *Condition
func More(field, value interface{}) *Condition
func MoreEq(field, value interface{}) *Condition
func Neg(field, value interface{}) *Condition
func NotBetween(field string, min, max any) *Condition
func NotIn(field string, value []interface{}) *Condition
func NotNull(field string) *Condition
func Null(field string) *Condition
func Or(field interface{}, operator Operator, value interface{}) *Condition
func ToCondition(params Json) (*Condition, error)
func ToConditions(params []Json) ([]*Condition, error)
func Where(field, value interface{}) *Condition
func (s *Condition) And() *Condition
func (s *Condition) ApplyToIndex(keys []string) []string
func (s *Condition) ApplyToObject(obj Json) bool
func (s *Condition) ApplyToValue(val any) bool
func (s *Condition) Or() *Condition
func (s *Condition) ToJson() Json
func ToConnector(s string) Connector
func (s Connector) Str() string
func ToField(s string) (Field, bool)
func ToFunction(s string) Function
func (s Function) Str() string
func NewItem(data Json) Item
func (s Item) Any(def interface{}, atribs ...string) interface{}
func (s Item) Array(atrib string) []interface{}
func (s Item) ArrayInt(atribs ...string) []int
func (s Item) ArrayInt64(atribs ...string) []int64
func (s Item) ArrayJson(atribs ...string) []Json
func (s Item) ArrayStr(atribs ...string) []string
func (s Item) Bool(atribs ...string) bool
func (s Item) Byte(atribs ...string) ([]byte, error)
func (s Item) Delete(keys []string) bool
func (s Item) Exist(key string) bool
func (s Item) From(as string) *Wheres
func (s Item) FromBase64(atribs ...string) string
func (s Item) Get(key string) interface{}
func (s Item) Int(atribs ...string) int
func (s Item) Int64(atribs ...string) int64
func (s *Item) IsEmpty() bool
func (s Item) Json(atrib string) Json
func (s Item) Num(atribs ...string) float64
func (s *Item) Set(key string, val interface{})
func (s Item) Str(atribs ...string) string
func (s Item) String(atribs ...string) string
func (s Item) Time(atribs ...string) time.Time
func (s Item) ToBase64(atribs ...string) string
func (s Item) ToByte() ([]byte, error)
func (s Item) ToJson() Json
func (s Item) ToMap() map[string]interface{}
func (s Item) ToObject(dest any) error
func (s Item) ToString() string
func (s *Item) ValAny(def interface{}, atribs ...string) interface{}
func (s Item) ValArray(def []interface{}, atribs ...string) []interface{}
func (s *Item) ValBool(def bool, atribs ...string) bool
func (s *Item) ValInt(def int, atribs ...string) int
func (s Item) ValInt64(def int64, atribs ...string) int64
func (s *Item) ValJson(def Json, atribs ...string) Json
func (s *Item) ValNum(def float64, atribs ...string) float64
func (s *Item) ValStr(def string, atribs ...string) string
func (s *Item) ValTime(def time.Time, atribs ...string) time.Time
func NewItems(data []Json) Items
func (s *Items) Add(item ...Json)
func (s *Items) AddMany(items []Json)
func (s *Items) Any(idx int, def interface{}, atribs ...string) interface{}
func (s Items) Array(idx int, atrib string) []interface{}
func (s Items) ArrayInt(idx int, atribs ...string) []int
func (s Items) ArrayInt64(idx int, atribs ...string) []int64
func (s Items) ArrayJson(idx int, atribs ...string) []Json
func (s Items) ArrayStr(idx int, atribs ...string) []string
func (s Items) Bool(idx int, atribs ...string) bool
func (s *Items) Delete(idx int, keys []string) bool
func (s Items) Exist(idx int, key string) bool
func (s *Items) First() (Item, error)
func (s Items) From(as string) *Wheres
func (s Items) Get(idx int, key string) interface{}
func (s Items) Int(idx int, atribs ...string) int
func (s Items) Int64(idx int, atribs ...string) int64
func (s Items) Json(idx int, atrib string) Json
func (s *Items) Last() (Item, error)
func (s Items) Num(idx int, atribs ...string) float64
func (s Items) One(idx int) (Item, error)
func (s *Items) Set(idx int, key string, val interface{})
func (s Items) Str(dx int, atribs ...string) string
func (s Items) String(idx int, atribs ...string) string
func (s Items) Time(idx int, atribs ...string) time.Time
func (s Items) ToByte() ([]byte, error)
func (s Items) ToJson() Json
func (s *Items) ToList(all, page, rows int) List
func (s *Items) ToMap() map[string]interface{}
func (s Items) ToString() string
func (s *Items) ValAny(idx int, def any, atribs ...string) interface{}
func (s *Items) ValArray(idx int, def []interface{}, atribs ...string) []interface{}
func (s *Items) ValBool(idx int, def bool, atribs ...string) bool
func (s *Items) ValInt(idx int, def int, atribs ...string) int
func (s *Items) ValInt64(idx int, def int64, atribs ...string) int64
func (s *Items) ValJson(idx int, def Json, atribs ...string) Json
func (s *Items) ValNum(idx int, def float64, atribs ...string) float64
func (s *Items) ValStr(idx int, def string, atribs ...string) string
func (s *Items) ValTime(idx int, def time.Time, atribs ...string) time.Time
func Joingy(left, right Iterator, keys map[string]string, joinType JoinType) Iterator
func (j JoinType) String() string
func MergeToKeyValue(left []Json, key string, value any) []Json
func MergeToMap(left, right []Json) []Json
func Prefixer(item Json, as string) Json
func ToJson(src string) (Json, error)
func (s Json) Any(def interface{}, atribs ...string) interface{}
func (s Json) Append(from Json) Json
func (s Json) Array(atrib ...string) []interface{}
func (s Json) ArrayBytes(atribs ...string) [][]byte
func (s Json) ArrayInt(atribs ...string) []int
func (s Json) ArrayInt64(atribs ...string) []int64
func (s Json) ArrayJson(atribs ...string) []Json
func (s Json) ArrayNumber(atribs ...string) []float64
func (s Json) ArrayStr(atribs ...string) []string
func (s Json) Bool(atribs ...string) bool
func (s Json) Byte(atribs ...string) ([]byte, error)
func (s Json) Clone() Json
func (s Json) Compare(from Json) Json
func (s Json) Delete(keys []string) bool
func (s Json) Exist(key string) bool
func (s Json) From(as string) *Wheres
func (s Json) FromBase64(atribs ...string) string
func (s Json) Get(keys ...string) (result interface{})
func (s Json) GetItem(items []Json, index int) (Json, bool)
func (s Json) Hidden(keys []string) Json
func (s Json) Int(atribs ...string) int
func (s Json) Int64(atribs ...string) int64
func (s Json) IsChanged(from Json) bool
func (s Json) IsDeferent(atrib string, val interface{}) bool
func (s Json) IsEmpty() bool
func (s Json) IsExist(key string) bool
func (s Json) Json(atrib string) Json
func (s Json) LowerString(atribs ...string) string
func (s Json) MapFloat(atrib string) map[string]float64
func (s Json) MapInt(atrib string) map[string]int
func (s Json) MapStr(atrib string) map[string]string
func (s Json) Num(atribs ...string) float64
func (s Json) Query(condition *Condition) bool
func (s Json) Remove(keys ...string)
func (s *Json) Scan(src interface{}) error
func (s *Json) ScanRows(rows *sql.Rows) error
func (s Json) Select(keys []string) Json
func (s Json) Set(key string, val interface{})
func (j Json) SetNested(keys []string, value interface{})
func (s Json) Str(atribs ...string) string
func (s Json) String(atribs ...string) string
func (s Json) Time(atribs ...string) time.Time
func (s Json) ToBase64(atribs ...string) string
func (s Json) ToByte() ([]byte, error)
func (s Json) ToEscapeHTML() string
func (s Json) ToMap() map[string]interface{}
func (s Json) ToObject(dest any) error
func (s Json) ToString() string
func (s Json) TrimString(atribs ...string) string
func (s Json) Update(from Json)
func (s Json) UpperString(atribs ...string) string
func (s Json) ValAny(def interface{}, atribs ...string) (result interface{})
func (s Json) ValArray(def []interface{}, atribs ...string) []interface{}
func (s Json) ValBool(def bool, atribs ...string) bool
func (s Json) ValDuration(_default time.Duration, atribs ...string) time.Duration
func (s Json) ValInt(def int, atribs ...string) int
func (s Json) ValInt64(def int64, atribs ...string) int64
func (s Json) ValJson(def Json, atribs ...string) Json
func (s Json) ValNum(def float64, atribs ...string) float64
func (s Json) ValStr(def string, atribs ...string) string
func (s Json) ValTime(def time.Time, atribs ...string) time.Time
func (s List) From(as string) *Wheres
func (s *List) ToByte() ([]byte, error)
func (s *List) ToJson() Json
func (s *List) ToMap() map[string]interface{}
func (s *List) ToString() string
func ToOperator(s string) Operator
func (s Operator) Str() string
func NewSafeData() *SafeData
func (s *SafeData) Add(value Json)
func (s *SafeData) Delete(index int)
func (s *SafeData) Get(index int) (Json, bool)
func (s *SafeData) Len() int
func (s *SafeData) Set(index int, value Json)
func (m *SafeMap) Delete(key string)
func (m *SafeMap) Get(key string) (interface{}, bool)
func (m *SafeMap) Set(key string, value interface{})
func (s *Source) Add(item Json)
func (s *Source) As() string
func (s *Source) Data(index int) Json
func (s *Source) LimitRows() int
func (s *Source) Next() (Json, bool)
func (s TypeData) Str() string
func NewValue(v any) Value
func (v Value) Fields() []string
func (v Value) Is(tpData string) bool
func (v Value) Raw() any
func (v Value) String() string
func ToWheRes(params []Json) (WheRes, error)
func (s WheRes) Add(condition *Condition)
func (s WheRes) Len() int
func (s WheRes) ToJson() []Json
func From(items []Json, as ...string) *Wheres
func (s *Wheres) Add(condition *Condition) *Wheres
func (s *Wheres) AdddResult(item Json) (next bool)
func (s *Wheres) All() []Json
func (s *Wheres) And(condition *Condition) *Wheres
func (s *Wheres) Asc(field string) *Wheres
func (s *Wheres) Desc(field string) *Wheres
func (s *Wheres) First() Json
func (s *Wheres) FullJoin(to []Json, as string, keys map[string]string) *Wheres
func (s *Wheres) Hidden(fields ...string) *Wheres
func (s *Wheres) IsDebug() *Wheres
func (s *Wheres) Join(to []Json, as string, keys map[string]string) *Wheres
func (s *Wheres) Last() Json
func (s *Wheres) LeftJoin(to []Json, as string, keys map[string]string) *Wheres
func (s *Wheres) Limit(page int, rows int) *Wheres
func (s *Wheres) One(idx int) Json
func (s *Wheres) Or(condition *Condition) *Wheres
func (s *Wheres) Order(field string, asc bool) *Wheres
func (s *Wheres) RightJoin(to []Json, as string, keys map[string]string) *Wheres
func (s *Wheres) Select(fields ...string) *Wheres
func (s *Wheres) ToJson() (Json, error)
func (s *Wheres) Where(condition *Condition) *Wheres
```

## `ettp/v1`

Gateway anterior (proxy, port-forward, tokens). Aún usado por algunos `cmd/`.

Funciones y métodos:

```go
func CorsAllowAll(allowedOrigins []string) *cors.Cors
func NewApiFunc(method, path string, h http.HandlerFunc, packageName string) *ApiFunc
func (s *Package) ToJson() et.Json
func NewProxy(id, path, name, description, solver, packageName string, s *Server) *Proxy
func (s *Proxy) ResetPortForward() error
func (s *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request)
func (s *Proxy) StartPortForward() error
func (s *Proxy) StopPortForward() error
func (p *Proxy) ToJson() et.Json
func (s *Resolver) GetResolve() string
func (s *Resolver) ToJson() et.Json
func (s *Resolver) ToString() string
func (r *Router) Connect(path string, handlerFn http.HandlerFunc, packageName string)
func (r *Router) Delete(path string, handlerFn http.HandlerFunc, packageName string)
func (r *Router) Get(path string, handlerFn http.HandlerFunc, packageName string)
func (r *Router) Head(path string, handlerFn http.HandlerFunc, packageName string)
func (r *Router) Options(path string, handlerFn http.HandlerFunc, packageName string)
func (r *Router) Patch(path string, handlerFn http.HandlerFunc, packageName string)
func (r *Router) Post(path string, handlerFn http.HandlerFunc, packageName string)
func (r *Router) Put(path string, handlerFn http.HandlerFunc, packageName string)
func (r *Router) ToJson() et.Json
func (r *Router) Trace(path string, handlerFn http.HandlerFunc, packageName string)
func (r *Router) With(middlewares ...func(http.Handler) http.Handler) *Router
func New(cnf *Config) (*Server, error)
func (s *Server) Close()
func (s *Server) Connect(path string, handlerFn http.HandlerFunc, packageName string)
func (s *Server) Delete(path string, handlerFn http.HandlerFunc, packageName string)
func (s *Server) DeleteProxyById(id string, save bool) error
func (s *Server) DeleteProxys(w http.ResponseWriter, r *http.Request)
func (s *Server) DeleteRouteById(id string, save bool) error
func (s *Server) DeleteTokenByKey(key string) error
func (s *Server) Get(path string, handlerFn http.HandlerFunc, packageName string)
func (s *Server) GetPackages(name string) et.Items
func (s *Server) GetProxys(w http.ResponseWriter, r *http.Request)
func (s *Server) GetProxysById(w http.ResponseWriter, r *http.Request)
func (s *Server) GetRouteById(id string) *Router
func (s *Server) GetTokenByKey(key string) (et.Item, error)
func (s *Server) HandlerValidToken(key string) (et.Item, error)
func (s *Server) Head(path string, handlerFn http.HandlerFunc, packageName string)
func (s *Server) Options(path string, handlerFn http.HandlerFunc, packageName string)
func (s *Server) Patch(path string, handlerFn http.HandlerFunc, packageName string)
func (s *Server) Post(path string, handlerFn http.HandlerFunc, packageName string)
func (s *Server) Private() *Router
func (s *Server) PrivateRoute(method, path string, h http.HandlerFunc, packageName string)
func (s *Server) ProtectRoute(method, path string, h http.HandlerFunc, packageName string)
func (s *Server) PublicRoute(method, path string, h http.HandlerFunc, packageName string)
func (s *Server) Put(path string, handlerFn http.HandlerFunc, packageName string)
func (s *Server) Reset() error
func (s *Server) Save() error
func (s *Server) SetAddr(port int)
func (s *Server) SetIdleTimeout(value time.Duration)
func (s *Server) SetPortForward(id, name, description, remoteHost string, remotePort, localPort int, packageName string, save bool) (*Proxy, error)
func (s *Server) SetPortForwards(w http.ResponseWriter, r *http.Request)
func (s *Server) SetProxy(id, path, name, description, solver, packageName string, save bool) (*Proxy, error)
func (s *Server) SetProxys(w http.ResponseWriter, r *http.Request)
func (s *Server) SetReadTimeout(value time.Duration)
func (s *Server) SetRouter(private bool, method, path, resolve string, header et.Json, tpHeader rt.TpHeader, excludeHeader []string, packageName string, saved bool) (*Router, error)
func (s *Server) SetWriteTimeout(value time.Duration)
func (s *Server) Start() error
func (s *Server) Trace(path string, handlerFn http.HandlerFunc, packageName string)
func (s *Server) Use(middlewares ...func(http.Handler) http.Handler)
func (s *Server) UseAutentication(middleware func(http.Handler) http.Handler) *Server
func (s *Server) With(middlewares ...func(http.Handler) http.Handler) *Router
func NewStorage() *Storage
func (t TpParams) String() string
func IntToTypeApi(i int) TypeApi
func (t TypeApi) String() string
```

## `ettp/v2`

API gateway: `New(name, *Config)`; sincroniza rutas entre réplicas por NATS. No llama `cache.Load`/`event.Load`.

Funciones y métodos:

```go
func CorsAllowAll(allowedOrigins []string) *cors.Cors
func (s *Codec) ReadRequestBody(body interface{}) error
func (s *Codec) ReadRequestHeader(r *rpc.Request) error
func (s *Codec) WriteResponse(r *rpc.Response, body interface{}) error
func (s *Package) ToJson() (et.Json, error)
func (r *Resolver) ToJson() et.Json
func (s *Router) ToJson() et.Json
func New(name string, cnf *Config) (*Server, error)
func (s *Server) Close()
func (s *Server) FindResolver(r *http.Request) (*Resolver, error)
func (s *Server) HTTPError(resolver *Resolver, metric *middleware.Metrics, w http.ResponseWriter, r *http.Request, status int, message string)
func (s *Server) HTTPSuccess(resolver *Resolver, metric *middleware.Metrics, rw *middleware.ResponseWriterWrapper)
func (s *Server) Private(method, path string, handlerFn http.HandlerFunc, packageName string) (*Solver, error)
func (s *Server) Public(method, path string, handlerFn http.HandlerFunc, packageName string) (*Solver, error)
func (s *Server) RemoveRouterById(id string, save bool) error
func (s *Server) Reset()
func (s *Server) Save() error
func (s *Server) SetRouter(method, path, resolve string, typeHeader int, header et.Json, excludeHeader []string, version int, packageName string, saved bool) (*Solver, error)
func (s *Server) Start()
func (s *Server) ToJson() et.Json
func (s *Server) ToString() string
func (s *Server) Use(middlewares ...func(http.Handler) http.Handler)
func (s *Server) UseAutentication(middleware func(http.Handler) http.Handler) *Server
func (s *Solver) ToJson() et.Json
func (s Status) String() string
func (t TpHeader) String() string
func StringToTypeRouter(s string) TypeRouter
func (t TypeRouter) String() string
```

## `event`

Pub/Sub sobre NATS.

Interfaces:

```go
type Router interface {
	Protect(method, path string, handler func(http.ResponseWriter, *http.Request))
}
```

Funciones y métodos:

```go
func Close()
func Emiter(channel string, data et.Json)
func Error(data et.Json)
func ErrorContext(tenantId, workflowId, nodeId, customerId string, ctx et.Json)
func ErrorDetails(tenantId string, statusCode int, provider, providerError, response string, attempt int)
func ErrorMessage(tenantId, message, method, functionName string)
func HealthCheck() bool
func HttpEventPublish(w http.ResponseWriter, r *http.Request)
func IsLoad() bool
func Load() error
func LoadRouter(r Router)
func Log(event string, data et.Json)
func On(channel string, handler Handler)
func Overflow(data et.Json)
func Publish(channel string, data et.Json) error
func Queue(channel, queue string, f func(Message)) (err error)
func Source(channel string, f func(Message)) error
func Stack(channel string, f func(Message)) error
func State(id string, status WorkStatus, data et.Json)
func Subscribe(channel string, f func(Message)) (err error)
func Unsubscribe(channel string) error
func Work(event string, data et.Json) et.Json
func New() (*Conn, error)
func (s *Conn) Close()
func (s *Conn) HealthCheck() bool
func DecodeMessage(data []byte) (Message, error)
func NewEvenMessage(channel string, data et.Json) Message
func (m Message) Encode() ([]byte, error)
func (s Message) Serialize() ([]byte, error)
func (s Message) ToJson() (et.Json, error)
func (s Message) ToString() string
```

## `file`

Archivos JSON, plantillas y watcher del sistema de archivos.

Funciones y métodos:

```go
func GetExtencion(filename string) string
func Load[T any](path string, obj *T) (bool, error)
func LoadOrSave[T any](path string, obj *T) (bool, error)
func MakeFile(path, name, model string, args ...any) (string, error)
func MakeFolder(names ...string) (string, error)
func Remove(path string) (bool, error)
func Save(path string, obj any) error
func WatcherPath(root string) error
func ExistPath(path string) FileInfo
func (s *FileInfo) Json() et.Json
func NewWatcher(root string) (*Watcher, error)
func (s *Watcher) Close() error
func (s *Watcher) Debug() *Watcher
func (s *Watcher) Load() error
func (s *Watcher) OnChmod(fn func(FileInfo, fsnotify.Event)) *Watcher
func (s *Watcher) OnCreate(fn func(FileInfo, fsnotify.Event)) *Watcher
func (s *Watcher) OnError(fn func(err error)) *Watcher
func (s *Watcher) OnReload(fn func(FileInfo, fsnotify.Event)) *Watcher
func (s *Watcher) OnRemove(fn func(FileInfo, fsnotify.Event)) *Watcher
func (s *Watcher) OnRename(fn func(FileInfo, fsnotify.Event)) *Watcher
func (s *Watcher) OnWatching(fn OnWatching) *Watcher
func (s *Watcher) OnWrite(fn func(FileInfo, fsnotify.Event)) *Watcher
```

## `graph`

Conexión a Neo4j (`NEO4J_*`).

Funciones y métodos:

```go
func Load() (*Conn, error)
func (s *Conn) Close() error
func (s *Conn) HealthCheck() bool
```

## `ia`

RAG multitenant sobre `jsql`: ingesta de archivos/SQL, embeddings, `Ask`.

Funciones y métodos:

```go
func Load(db *jsql.DB, schema string, cnf Config) (*Rag, error)
func (s *Rag) Ask(ctx context.Context, tenantId, projectId, conversationId, userId, question string) (et.Json, error)
func (s *Rag) DeleteDocument(tenantId, projectId, id string) error
func (s *Rag) IngestFile(ctx context.Context, tenantId, projectId, name string, data []byte, userId string) (et.Json, error)
func (s *Rag) IngestSQL(ctx context.Context, tenantId, projectId, name, query string, sourceDB *jsql.DB, userId string) (et.Json, error)
func (s *Rag) ListDocuments(tenantId, projectId string) (et.Items, error)
func (s *Rag) ListMessages(tenantId, projectId, conversationId string) (et.Items, error)
func (s *Rag) LoadRouter(mux chi.Router)
```

## `infobip`

SMS, email y plantillas de WhatsApp.

Funciones y métodos:

```go
func NewSenderInfobip(params Params) (*SenderInfobip, error)
func (s *SenderInfobip) SendEmail(to []string, subject, htmlContent string, params et.Json, tpMessage string) (et.Item, error)
func (s *SenderInfobip) SendSMS(contactNumbers []string, content string, params et.Json, tpMessage string) (et.Item, error)
func (s *SenderInfobip) SendWhatsApp(contactNumbers []string, templateName, language string, placeholders [][]string, tpMessage string) (et.Items, error)
```

## `iterate`

Medición de tiempos por segmentos.

Funciones y métodos:

```go
func End(tag, msg string, isDebug bool) time.Duration
func Segment(tag, msg string, isDebug bool) time.Duration
func Start(tag string)
```

## `jia`

Agentes OpenAI: agentes, participantes, conversaciones, skills.

Interfaces:

```go
type Router interface {
	Protect(method, path string, handler func(http.ResponseWriter, *http.Request))
}

type Sender interface {
	SendTextMessage(to string, content string) (et.Item, error)
}

type Skill interface {
	Tag() string
	Name() string
	Description() string
	Execute(
		ctx context.Context,
		input et.Json,
	) (et.Json, error)
}

type Store interface {
	Set(collection, id, ownerId string, obj any) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	Query(query et.Json) (et.Items, error)
}
```

Funciones y métodos:

```go
func (s *Agent) Debug() *Agent
func (s *Agent) OnDelete(fn func(agent *Agent) error) *Agent
func (s *Agent) OnSave(fn func(agent *Agent) error) *Agent
func (s *Agent) ToJson() et.Json
func (s *Agent) ToString() string
func NewApiSkill(tag, name, description, url, method string, headers et.Json, body et.Json) (*ApiSkill, error)
func (s *ApiSkill) Description() string
func (s *ApiSkill) Execute(ctx context.Context, input et.Json) (et.Json, error)
func (s *ApiSkill) Name() string
func (s *ApiSkill) Tag() string
func (s *Conversation) AddParticipant(participant *Participant) *Conversation
func (s *Conversation) OnDelete(fn func(conversation *Conversation) error) *Conversation
func (s *Conversation) OnSave(fn func(conversation *Conversation) error) *Conversation
func (s *Conversation) SendTextMessage(content, userId, to string) (*Message, error)
func (s *Conversation) SetConvId(convId string) *Conversation
func (s *Conversation) SetLimitMessages(limit int) *Conversation
func (s *Conversation) ToJson() et.Json
func (s *Conversation) ToString() string
func (s *ConversationResult) ToJson() et.Json
func Load(id string, store Store) (*Ia, error)
func New(tag string, store Store, userId string) (*Ia, error)
func (s *Ia) Conversation(ctx context.Context, tagAgent, to, name, prompt string) (*Conversation, error)
func (s *Ia) Embed(ctx context.Context, agentName string, text string) ([]float64, error)
func (s *Ia) HttpConversation(w http.ResponseWriter, r *http.Request)
func (s *Ia) HttpDeleteAgent(w http.ResponseWriter, r *http.Request)
func (s *Ia) HttpDeleteConversation(w http.ResponseWriter, r *http.Request)
func (s *Ia) HttpDeleteParticipant(w http.ResponseWriter, r *http.Request)
func (s *Ia) HttpGetAgent(w http.ResponseWriter, r *http.Request)
func (s *Ia) HttpGetParticipant(w http.ResponseWriter, r *http.Request)
func (s *Ia) HttpNewAgent(w http.ResponseWriter, r *http.Request)
func (s *Ia) HttpNewParticipant(w http.ResponseWriter, r *http.Request)
func (s *Ia) HttpSetAgent(w http.ResponseWriter, r *http.Request)
func (s *Ia) HttpSetParticipant(w http.ResponseWriter, r *http.Request)
func (s *Ia) LoadRouter(r Router)
func (s *Ia) OnDelete(fn func(ia *Ia) error) *Ia
func (s *Ia) OnSave(fn func(ia *Ia) error) *Ia
func (s *Ia) SetContextAgent(tag string, context, userId string) (*Agent, error)
func (s *Ia) SetModelAgent(tag string, model, userId string) (*Agent, error)
func (s *Ia) SetSkillAgent(tag string, skill Skill, userId string) (*Agent, error)
func (s *Ia) ToJson() et.Json
func (s *Ia) ToString() string
func (s *Message) OnDelete(fn func(message *Message) error) *Message
func (s *Message) OnSave(fn func(message *Message) error) *Message
func (s *Message) ToJson() et.Json
func (s *Message) ToString() string
func (s *Participant) OnDelete(fn func(participant *Participant) error) *Participant
func (s *Participant) OnSave(fn func(participant *Participant) error) *Participant
func (s *Participant) SetConvId(convId, userId string) *Participant
func (s *Participant) SetName(name, userId string) *Participant
func (s *Participant) ToJson() et.Json
func (s *Participant) ToString() string
```

## `jrex`

Runtime JavaScript (goja) con módulos, bindings y hot-reload.

Interfaces:

```go
type Store interface {
	Set(collection, id, ownerId string, obj any) error
	Get(collection, id string, dest any) (bool, error)
}
```

Funciones y métodos:

```go
func GetArgs(args []goja.Value, idx int, defaultValue any) any
func GetVersion(version string, part Part) string
func NewStore(baseDir string) (*FileStore, error)
func (s *FileStore) Get(collection, id string, dest any) (bool, error)
func (s *FileStore) Set(collection, id, ownerId string, obj any) error
func NewInstance(id ...string) *Instance
func (s *Instance) Error(err error) *goja.Object
func (s *Instance) Get(name string) goja.Value
func (s *Instance) GetCode() string
func (s *Instance) GetJson(name string) et.Json
func (s *Instance) Run() (et.Json, error)
func (s *Instance) Save(store Store) error
func (s *Instance) SetBinding(name string, value interface{}) *Instance
func (s *Instance) SetCode(code string) *Instance
func (s *Instance) SetCtx(ctx et.Json) *Instance
func (s *Instance) SetStore(store Store) *Instance
func (s *Instance) Value(value interface{}) goja.Value
func Load(tag string, store Store) (*Jrex, error)
func (s *Jrex) Debug() *Jrex
func (s *Jrex) NewInstance(module string) (*Instance, error)
func (s *Jrex) NewModule(tag, userId string) (*Module, error)
func (s *Jrex) Notify(kind, message string)
func (s *Jrex) OnSave(fn func(jrex *Jrex, userId string) error) *Jrex
func (s *Jrex) Run() (et.Json, error)
func (s *Jrex) RunDev() error
func (s *Jrex) RunInstance(instance *Instance) (et.Json, error)
func (s *Jrex) RunModule(module string) (et.Json, error)
func (s *Jrex) Save(userId string) error
func (s *Jrex) Set(name string, value interface{}) *Jrex
func (s *Jrex) ToJson() et.Json
func (s *Jrex) ToString() string
func (s *Jrex) Up(store Store) *Jrex
func (s *Module) SetMetadata(metadata et.Json)
func ToPart(value string) (Part, bool)
```

## `jrpc`

`net/rpc` sobre TCP con registro de solvers.

Funciones y métodos:

```go
func Call(method string, args any) (any, error)
func CallItem(method string, args et.Json) (et.Item, error)
func CallItems(method string, args et.Json) (et.Items, error)
func CallJson(method string, args et.Json) (et.Json, error)
func Close()
func HttpListRouters(w http.ResponseWriter, r *http.Request)
func Start(port int) error
func Mount(host string, port int, services any, packageName string) (*Package, error)
func (s *Package) Add(method string, imputs []string, output []string) *Solver
func (s *Package) ToJson() et.Json
func GetSolver(method string) (*Solver, error)
```

## `jsql`

SQL builder/ORM guiado por JSON: modelos, consultas, comandos, triggers, transacciones, series. API completa en `jsql/catalog.go`; spec en `jsql/spec.md`.

Interfaces:

```go
type Connection interface {
	ID() string
	SetId(string)
	GetParams() et.Json
	SetDatabase(string)
	GetDatabase() string
	SetPassword(string)
}

type Driver interface {
	UseSchema() bool
	CreateDB(connection *ConnectParams, timeout ...time.Duration) error
	DropDB(db *DB, timeout ...time.Duration) error
	Connect(db *DB, timeout ...time.Duration) (*sql.DB, error)
	ExistModel(db *sql.DB, model *Model, timeout ...time.Duration) (bool, error)
	Load(model *Model, timeout ...time.Duration) (string, error)
	Drop(model *Model, timeout ...time.Duration) error
	Query(query *Query, timeout ...time.Duration) (string, error)
	Command(command *Command, timeout ...time.Duration) (string, error)
}

type Tester interface {
	Test(ctx context.Context, connection Connection, query string) error
}
```

Funciones y métodos:

```go
func AddAuditLog(auditLog []et.Json, userId string, action string) []et.Json
func And(field interface{}, operator et.Operator, value interface{}) *et.Condition
func ArgWhitAs(arg string) ([]string, bool)
func ArgWhitSchema(arg string) ([]string, bool)
func Between(field string, min, max any) *et.Condition
func CreateDB(connection *ConnectParams, timeout ...time.Duration) error
func DetailKeys(key, foreignKey string) map[string]string
func DropDB(db *DB, timeout ...time.Duration) error
func Eq(field string, value interface{}) *et.Condition
func EscapeSQLString(s string) string
func In(field string, value []interface{}) *et.Condition
func Is(field string, value interface{}) *et.Condition
func IsNot(field string, value interface{}) *et.Condition
func JsonString(val any) (string, error)
func Less(field string, value interface{}) *et.Condition
func LessEq(field string, value interface{}) *et.Condition
func Like(field string, value interface{}) *et.Condition
func More(field string, value interface{}) *et.Condition
func MoreEq(field string, value interface{}) *et.Condition
func Neg(field string, value interface{}) *et.Condition
func NotBetween(field string, min, max any) *et.Condition
func NotIn(field string, value []interface{}) *et.Condition
func NotNull(field string) *et.Condition
func Null(field string) *et.Condition
func Or(field interface{}, operator et.Operator, value interface{}) *et.Condition
func Probe(ctx context.Context, db *sql.DB, query string) error
func Quoted(val any) any
func Register(name string, driver Driver)
func RollupKeys(key, foreignKey string) map[string]string
func RowsToItems(rows *sql.Rows) et.Items
func SQLParse(sql string, args ...any) string
func StatusList() []interface{}
func TestConnection(ctx context.Context, connection Connection, query string) error
func TimeoutContext(timeout ...time.Duration) (context.Context, context.CancelFunc)
func ToArrayAny(statuses []string) []interface{}
func Where(field, value interface{}) *et.Condition
func Wrapper(db *DB) map[string]any
func (s *Column) ToJson() et.Json
func (s *Command) AfterDelete(fn TriggerFunction) *Command
func (s *Command) AfterInsert(fn TriggerFunction) *Command
func (s *Command) AfterInsertOrUpdate(fn TriggerFunction) *Command
func (s *Command) AfterUpdate(fn TriggerFunction) *Command
func (s *Command) And(cond *et.Condition) *Command
func (s *Command) BeforeDelete(fn TriggerFunction) *Command
func (s *Command) BeforeInsert(fn TriggerFunction) *Command
func (s *Command) BeforeInsertOrUpdate(fn TriggerFunction) *Command
func (s *Command) BeforeUpdate(fn TriggerFunction) *Command
func (s *Command) Debug() *Command
func (s *Command) Exec() (et.Items, error)
func (s *Command) ExecTx(tx *Tx) (et.Items, error)
func (s *Command) Limit(rows int) *Command
func (s *Command) One() (et.Item, error)
func (s *Command) OneTx(tx *Tx) (et.Item, error)
func (s *Command) Or(cond *et.Condition) *Command
func (s *Command) Return(fields ...string) *Command
func (s *Command) Test() *Command
func (s *Command) ToJson() et.Json
func (s *Command) Where(cond *et.Condition) *Command
func ConnectionFromJson(driver, host string, params et.Json) (Connection, error)
func GetConnection(driver, host string) (Connection, error)
func ConnectTo(params ConnectParams) (*DB, error)
func Load() (*DB, error)
func LoadDb(params et.Json) (*DB, error)
func LoadTo(dbName string, hostName ...string) (*DB, error)
func NewDB(params ConnectParams) (*DB, error)
func (s *DB) Close() error
func (s *DB) Debug()
func (s *DB) Define(define Define) (*Model, error)
func (s *DB) DefineModel(schema, name string, version int, userId string) (*Model, error)
func (s *DB) DefineProjectModel(schema, name string, version int, userId string) (*Model, error)
func (s *DB) DefineTenantModel(schema, name string, version int, userId string) (*Model, error)
func (s *DB) GetModel(schema, name string) (*Model, error)
func (s *DB) Init() error
func (s *DB) LoadModel(schema string, def et.Json) (*Model, error)
func (s *DB) NewModel(schema, name string, version int, userId string) *Model
func (s *DB) OnAuditLog(fn func(userId string, action string) error)
func (s *DB) Query(sql et.Json) (et.Items, error)
func (s *DB) QueryTx(tx *Tx, sql et.Json) (et.Items, error)
func (s *DB) RemoveModel(schema, name string) error
func (s *DB) SetDebug(debug bool)
func (s *DB) Sql(query string, args ...any) (et.Items, error)
func (s *DB) SqlTx(tx *Tx, query string, arg ...any) (et.Items, error)
func (s *DB) ToJson() et.Json
func (s *Detail) GetQuery(item et.Json, page, rows int) *Query
func (s *Detail) Ref() et.Json
func (s *From) Ref() et.Json
func (s *Master) Ref() et.Json
func (s *Model) AddAuditLog(userId string, action string)
func (s *Model) AfterDelete(fn TriggerFunction) *Model
func (s *Model) AfterInsert(fn TriggerFunction) *Model
func (s *Model) AfterInsertOrUpdate(fn TriggerFunction) *Model
func (s *Model) AfterUpdate(fn TriggerFunction) *Model
func (s *Model) As(as ...string) *Query
func (s *Model) BeforeDelete(fn TriggerFunction) *Model
func (s *Model) BeforeInsert(fn TriggerFunction) *Model
func (s *Model) BeforeInsertOrUpdate(fn TriggerFunction) *Model
func (s *Model) BeforeUpdate(fn TriggerFunction) *Model
func (s *Model) Bridge(name string) (*Model, bool)
func (s *Model) Bulk(data []et.Json) *Command
func (s *Model) Calc(fields ...string) *Query
func (s *Model) Count() (int, error)
func (s *Model) Db() *DB
func (s *Model) Debug() *Model
func (s *Model) DefineAfterDelete(name, code string, version int) *Model
func (s *Model) DefineAfterInsert(name, code string, version int) *Model
func (s *Model) DefineAfterUpdate(name, code string, version int) *Model
func (s *Model) DefineAttrib(name string, tp et.TypeData, deFault any) *Column
func (s *Model) DefineBeforeDelete(name, code string, version int) *Model
func (s *Model) DefineBeforeInsert(name, code string, version int) *Model
func (s *Model) DefineBeforeUpdate(name, code string, version int) *Model
func (s *Model) DefineCalc(name, script string) *Model
func (s *Model) DefineCalcFunc(name string, calc CalcFunction) *Model
func (s *Model) DefineColumn(name string, tp et.TypeData, deFault any) *Column
func (s *Model) DefineDetail(name string, keys map[string]string, rows int, selects ...string) (*Model, error)
func (s *Model) DefineForeignKeys(to *Model, keys map[string]string, onDeleteCascade, onUpdateCascade bool) *Detail
func (s *Model) DefineHidden(name ...string)
func (s *Model) DefineIdxField() *Index
func (s *Model) DefineIndex(name string, tp et.TypeData, deFault any) *Index
func (s *Model) DefineMaster(name string, to *Model, keys, toKeys map[string]string, selects []string, rows ...int) (*Model, error)
func (s *Model) DefineModel() *Model
func (s *Model) DefineOmitUpdate(names ...string) *Model
func (s *Model) DefinePrimaryKey(name string, tp et.TypeData, deFault any) *Index
func (s *Model) DefineRequired(name string, tp et.TypeData, deFault any) *Index
func (s *Model) DefineRollup(name string, to *Model, keys map[string]string, selects []string, operation RollupOperation) (*Rollups, error)
func (s *Model) DefineSource() *Column
func (s *Model) DefineUnique(name string, tp et.TypeData, deFault any) *Index
func (s *Model) Delete() *Command
func (s *Model) Detail(name string) (*Model, bool)
func (s *Model) First(n int) (et.Items, error)
func (s *Model) GetCalcFunc(name string) (CalcFunction, bool)
func (s *Model) GetColumn(name string) (*Column, bool)
func (s *Model) GetField(name string) (*Field, bool)
func (s *Model) GetFrom() *From
func (s *Model) GetKind() string
func (s *Model) GetModel(schema, name string) (*Model, error)
func (s *Model) GetSchema() *Schema
func (s *Model) Init() error
func (s *Model) Insert(data et.Json) *Command
func (s *Model) Join(to *Model, as string, on []*et.Condition) *Query
func (s *Model) Master(name string) (*Query, bool)
func (s *Model) Query(query et.Json) *Query
func (s *Model) QueryTx(tx *Tx, query et.Json) *Query
func (s *Model) Replace(other *Model)
func (s *Model) Select(fields ...string) *Query
func (s *Model) SetDb(db *DB)
func (s *Model) SetTable()
func (s *Model) SqlDB() *sql.DB
func (s *Model) Stricted()
func (s *Model) ToJson() et.Json
func (s *Model) Update(data et.Json) *Command
func (s *Model) Upsert(data et.Json) *Command
func (s *Model) Where(cond *et.Condition) *Query
func (s *MssqlConection) GetDatabase() string
func (s *MssqlConection) GetParams() et.Json
func (s *MssqlConection) ID() string
func (s *MssqlConection) SetDatabase(name string)
func (s *MssqlConection) SetId(id string)
func (s *MssqlConection) SetPassword(password string)
func (s *MysqlConection) GetDatabase() string
func (s *MysqlConection) GetParams() et.Json
func (s *MysqlConection) ID() string
func (s *MysqlConection) SetDatabase(name string)
func (s *MysqlConection) SetId(id string)
func (s *MysqlConection) SetPassword(password string)
func (s *OracleConection) GetDatabase() string
func (s *OracleConection) GetParams() et.Json
func (s *OracleConection) ID() string
func (s *OracleConection) SetDatabase(name string)
func (s *OracleConection) SetId(id string)
func (s *OracleConection) SetPassword(password string)
func (s *PgConection) GetDatabase() string
func (s *PgConection) GetParams() et.Json
func (s *PgConection) ID() string
func (s *PgConection) SetDatabase(name string)
func (s *PgConection) SetId(id string)
func (s *PgConection) SetPassword(password string)
func NewQuery(model *Model, as ...string) *Query
func (s *Query) AddCondition(cond *et.Condition) *Query
func (s *Query) All() (et.Items, error)
func (s *Query) AllTx(tx *Tx) (et.Items, error)
func (s *Query) And(cond *et.Condition) *Query
func (s *Query) Calc(fields ...string) *Query
func (s *Query) Count() (int, error)
func (s *Query) CountTx(tx *Tx) (int, error)
func (s *Query) Debug() *Query
func (s *Query) Detail(fields ...string) *Query
func (s *Query) Exists() (bool, error)
func (s *Query) ExistsTx(tx *Tx) (bool, error)
func (s *Query) First(n int) (et.Items, error)
func (s *Query) FirstTx(tx *Tx, n int) (et.Items, error)
func (s *Query) FullJoin(model *Model, as string, on []*et.Condition) *Query
func (s *Query) GetField(field string) (*Field, bool)
func (s *Query) GetSelectField(name string) (*Field, bool)
func (s *Query) GroupBy(fields ...string) *Query
func (s *Query) HasStar() bool
func (s *Query) Having(cond *et.Condition) *Query
func (s *Query) Hidden(fields ...string) *Query
func (s *Query) Join(model *Model, as string, on []*et.Condition) *Query
func (s *Query) LeftJoin(model *Model, as string, on []*et.Condition) *Query
func (s *Query) Limit(page, rows int) (et.Items, error)
func (s *Query) LimitTx(tx *Tx, page, rows int) (et.Items, error)
func (s *Query) Master(fields ...string) *Query
func (s *Query) One() (et.Item, error)
func (s *Query) OneTx(tx *Tx) (et.Item, error)
func (s *Query) Or(cond *et.Condition) *Query
func (s *Query) OrderBy(field string, sorted ...bool) *Query
func (s *Query) Page(page int) *Query
func (s *Query) RightJoin(model *Model, as string, on []*et.Condition) *Query
func (s *Query) Select(fields ...string) *Query
func (s *Query) SourceHiddens(from *From) []string
func (s *Query) StarFields() []string
func (s *Query) Test() *Query
func (s *Query) ToJson() et.Json
func (s *Query) Where(cond *et.Condition) *Query
func (s *QueryDetail) GetQuery(item et.Json) *Query
func (s *QueryRollups) GetQuery(item et.Json) (*Query, bool)
func (s RollupOperation) IsAggregate() bool
func (s RollupOperation) IsValid() bool
func (s *Schema) GetModel(name string) (*Model, error)
func (s *Schema) LoadModel(def et.Json) (*Model, error)
func (s *Schema) ToJson() et.Json
func DefineSeries(db *DB, schema string) (*Series, error)
func (s *Series) DeleteSeries(tag string) error
func (s *Series) GenSerie(tag string) (string, error)
func (s *Series) GenValue(tag string) (int, error)
func (s *Series) GetSeries(tag string) (et.Item, error)
func (s *Series) Model() *Model
func (s *Series) NewSeries(tag, format string) error
func (s *Series) SetSeries(tag string, format string, value int) error
func (s *SqliteConection) GetDatabase() string
func (s *SqliteConection) GetParams() et.Json
func (s *SqliteConection) ID() string
func (s *SqliteConection) SetDatabase(name string)
func (s *SqliteConection) SetId(id string)
func (s *SqliteConection) SetPassword(password string)
func NewTx() *Tx
func (s *Tx) Commit() error
func (s *Tx) Query(db *sql.DB, query string, args ...any) (*sql.Rows, error)
func (s *Tx) Rollback() error
func (s TypeColumn) Str() string
```

## `jsql/drivers/mssql`

Driver de `jsql`; se registra en `init()` y se importa por efecto lateral.

Funciones y métodos:

```go
func Quoted(v et.Value) string
func QuotedAs(tp et.TypeData, val any) string
func (s *Mssql) Command(command *jsql.Command, timeout ...time.Duration) (string, error)
func (s *Mssql) Connect(db *jsql.DB, timeout ...time.Duration) (*sql.DB, error)
func (s *Mssql) CreateDB(connection *jsql.ConnectParams, timeout ...time.Duration) error
func (s *Mssql) Drop(model *jsql.Model, timeout ...time.Duration) error
func (s *Mssql) DropDB(db *jsql.DB, timeout ...time.Duration) error
func (s *Mssql) ExistModel(db *sql.DB, model *jsql.Model, timeout ...time.Duration) (bool, error)
func (s *Mssql) Load(model *jsql.Model, timeout ...time.Duration) (string, error)
func (s *Mssql) Query(query *jsql.Query, timeout ...time.Duration) (string, error)
func (s *Mssql) Test(ctx context.Context, connection jsql.Connection, query string) error
func (s *Mssql) UseSchema() bool
```

## `jsql/drivers/mysql`

Driver de `jsql`; se registra en `init()` y se importa por efecto lateral.

Funciones y métodos:

```go
func Quoted(v et.Value) string
func QuotedAs(tp et.TypeData, val any) string
func (s *Mysql) Command(command *jsql.Command, timeout ...time.Duration) (string, error)
func (s *Mysql) Connect(db *jsql.DB, timeout ...time.Duration) (*sql.DB, error)
func (s *Mysql) CreateDB(connection *jsql.ConnectParams, timeout ...time.Duration) error
func (s *Mysql) Drop(model *jsql.Model, timeout ...time.Duration) error
func (s *Mysql) DropDB(db *jsql.DB, timeout ...time.Duration) error
func (s *Mysql) ExistModel(db *sql.DB, model *jsql.Model, timeout ...time.Duration) (bool, error)
func (s *Mysql) Load(model *jsql.Model, timeout ...time.Duration) (string, error)
func (s *Mysql) Query(query *jsql.Query, timeout ...time.Duration) (string, error)
func (s *Mysql) Test(ctx context.Context, connection jsql.Connection, query string) error
func (s *Mysql) UseSchema() bool
```

## `jsql/drivers/oracle`

Driver de `jsql`; se registra en `init()` y se importa por efecto lateral.

Funciones y métodos:

```go
func Quoted(v et.Value) string
func QuotedAs(tp et.TypeData, val any) string
func (s *Oracle) Command(command *jsql.Command, timeout ...time.Duration) (string, error)
func (s *Oracle) Connect(db *jsql.DB, timeout ...time.Duration) (*sql.DB, error)
func (s *Oracle) CreateDB(connection *jsql.ConnectParams, timeout ...time.Duration) error
func (s *Oracle) Drop(model *jsql.Model, timeout ...time.Duration) error
func (s *Oracle) DropDB(db *jsql.DB, timeout ...time.Duration) error
func (s *Oracle) ExistModel(db *sql.DB, model *jsql.Model, timeout ...time.Duration) (bool, error)
func (s *Oracle) Load(model *jsql.Model, timeout ...time.Duration) (string, error)
func (s *Oracle) Query(query *jsql.Query, timeout ...time.Duration) (string, error)
func (s *Oracle) Test(ctx context.Context, connection jsql.Connection, query string) error
func (s *Oracle) UseSchema() bool
```

## `jsql/drivers/postgres`

Driver de `jsql`; se registra en `init()` y se importa por efecto lateral.

Funciones y métodos:

```go
func CreateDatabase(ctx context.Context, db *sql.DB, name string) error
func DropDatabase(ctx context.Context, db *sql.DB, name string) error
func ExistDatabase(ctx context.Context, db *sql.DB, name string) (bool, error)
func Quoted(v et.Value) string
func (s *Postgres) Command(command *jsql.Command, timeout ...time.Duration) (string, error)
func (s *Postgres) Connect(db *jsql.DB, timeout ...time.Duration) (*sql.DB, error)
func (s *Postgres) CreateDB(connection *jsql.ConnectParams, timeout ...time.Duration) error
func (s *Postgres) Drop(model *jsql.Model, timeout ...time.Duration) error
func (s *Postgres) DropDB(db *jsql.DB, timeout ...time.Duration) error
func (s *Postgres) ExistModel(db *sql.DB, model *jsql.Model, timeout ...time.Duration) (bool, error)
func (s *Postgres) Load(model *jsql.Model, timeout ...time.Duration) (string, error)
func (s *Postgres) Query(query *jsql.Query, timeout ...time.Duration) (string, error)
func (s *Postgres) Test(ctx context.Context, connection jsql.Connection, query string) error
func (s *Postgres) UseSchema() bool
```

## `jsql/drivers/sqlite`

Driver de `jsql`; se registra en `init()` y se importa por efecto lateral.

Funciones y métodos:

```go
func Quoted(v et.Value) string
func (s *Sqlite) Command(command *jsql.Command, timeout ...time.Duration) (string, error)
func (s *Sqlite) Connect(db *jsql.DB, timeout ...time.Duration) (*sql.DB, error)
func (s *Sqlite) CreateDB(connection *jsql.ConnectParams, timeout ...time.Duration) error
func (s *Sqlite) Drop(model *jsql.Model, timeout ...time.Duration) error
func (s *Sqlite) DropDB(db *jsql.DB, timeout ...time.Duration) error
func (s *Sqlite) ExistModel(db *sql.DB, model *jsql.Model, timeout ...time.Duration) (bool, error)
func (s *Sqlite) Load(model *jsql.Model, timeout ...time.Duration) (string, error)
func (s *Sqlite) Query(query *jsql.Query, timeout ...time.Duration) (string, error)
func (s *Sqlite) Test(ctx context.Context, connection jsql.Connection, query string) error
func (s *Sqlite) UseSchema() bool
```

## `jtcp`

Nodo TCP distribuido con elección de líder tipo Raft, balanceador y TLS.

Interfaces:

```go
type Service interface {
	Execute(method string, request *Message) *Response
}
```

Funciones y métodos:

```go
func NewClientTLSConfig(caFile string) (*tls.Config, error)
func NewMTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error)
func NewSelfSignedTLS() (*tls.Config, error)
func NewTLSFromFiles(certFile, keyFile string) (*tls.Config, error)
func StartConsole(client *Client)
func (s *Balancer) AddNode(addr string)
func NewClient(addr string, tlsConfig ...*tls.Config) *Client
func (s *Client) Close() error
func (s *Client) Connect() error
func (s *Client) OnConnect(fn func(*Client))
func (s *Client) OnDisconnect(fn func(*Client))
func (s *Client) OnError(fn func(*Client, error))
func (s *Client) OnInbox(fn func(*Client, *Message))
func (s *Client) OnSend(fn func(*Client, []byte))
func (s *Client) Request(method string, args ...any) *Response
func (s *Client) Send(ms *Message) error
func (s *Client) SetDebug(debug bool)
func (s *Client) SetTimeOut(timeout time.Duration)
func (s *Client) Start() error
func (s *Client) ToJson() et.Json
func NewMessage(tp TpMessage, message any) (*Message, error)
func ToMessage(bt []byte) (*Message, error)
func (s *Message) Get(dest any) error
func (s *Message) GetArgs(dest ...any) error
func (s *Message) Response() (*Response, error)
func (s *Message) ToJson() et.Json
func (s *Msg) Get(dest any) error
func (s *Msg) ID() string
func NewNode(port int, tlsConfig ...*tls.Config) *Node
func (s *Node) AddNode(addr string)
func (s *Node) Broadcast(destination []string, tp TpMessage, message any)
func (s *Node) Close() error
func (s *Node) Error(c *Client, err error) error
func (s *Node) GetMethods() et.Json
func (s *Node) GetPeers() []*Client
func (s *Node) LeaderID() (string, bool)
func (s *Node) Mount(service Service) error
func (s *Node) Next() *Client
func (s *Node) OnBecomeLeader(fn func(*Node))
func (s *Node) OnChangeLeader(fn func(*Node))
func (s *Node) OnConnect(fn func(*Client))
func (s *Node) OnDisconnect(fn func(*Client))
func (s *Node) OnError(fn func(*Client, error))
func (s *Node) OnInbox(fn func(*Msg))
func (s *Node) OnSend(fn func(*Msg))
func (s *Node) RemoveNode(addr string)
func (s *Node) Request(to *Client, method string, args ...any) *Response
func (s *Node) ResponseError(msg *Msg, err error) error
func (s *Node) Send(to *Client, tp TpMessage, message any) error
func (s *Node) SendError(to *Client, err error) error
func (s *Node) Start() (err error)
func (s *Raft) LeaderID() (leader string, imLeader bool)
func TcpError(msg any) *Response
func TcpResponse(res ...any) *Response
func (s *Response) Get(dest ...any) error
func (s *Response) ToJson() (et.Json, error)
func (s *Response) ToString() string
func (s *Tcp) Execute(method string, request *Message) *Response
```

## `jwf`

Motor de workflows en memoria: flujos, pasos (Go o JS), instancias, reintentos. Persistencia vía `OnChange`.

Funciones y métodos:

```go
func GetArgs(args []goja.Value, idx int, defaultValue any) any
func RunCode(code string, ctx et.Json, bindings map[string]any) (any, error)
func RunScripts(scripts []*Script, ctx et.Json, bindings map[string]any) (any, error)
func LoadFlow(def et.Json) (*Flow, error)
func NewFlow(tag, name, version, ownerId, userId string) *Flow
func (s *Flow) AddStep(stepDef et.Json, userId string) (*Flow, error)
func (s *Flow) Error(tag, version, title string, fn fnStep) *Flow
func (s *Flow) LoadInstance(def et.Json) (*Instance, error)
func (s *Flow) NewInstance(id, code, name string, tags et.Json, trigger *Trigger, userId string) *Instance
func (s *Flow) OnChange(fn func(data et.Json) error) *Flow
func (s *Flow) Publish() error
func (s *Flow) SetBinding(key string, value interface{}) *Flow
func (s *Flow) SetTimeAwait(time time.Duration, userId string) *Flow
func (s *Flow) Step(tag, title string, fn fnStep) *Flow
func (s *Flow) ToJson() et.Json
func (s *Flow) ToString() string
func (s *Instance) Done() *Instance
func (s *Instance) OnChange(fn func(data et.Json)) *Instance
func (s *Instance) Run(ctx et.Json, await bool) (et.Json, error)
func (s *Instance) SetBinding(key string, value any) *Instance
func (s *Instance) SetParams(params et.Json) et.Json
func (s *Instance) ToJson() et.Json
func (s *Instance) ToString() string
func (s Status) Str() string
func LoadStep(def et.Json) (*Step, error)
func (s *Step) OnChange(fn func(data et.Json))
func (s *Step) OnStatus(fn func(status Status))
func (s *Step) Run(instance *Instance, ctx et.Json) (et.Json, error)
func (s *Step) RunFunction(instance *Instance, ctx et.Json) (et.Json, error)
func (s *Step) RunScript(instance *Instance, ctx et.Json) (et.Json, error)
func (s *Step) SetBinding(name string, value interface{}) *Step
func (s *Step) ToJson() et.Json
func (s *Step) ToString() string
func New(id string) (*WorkFlow, error)
func (s *WorkFlow) GetInstance(tag, triggerTag, id, code string, tags et.Json, userId string) (*Instance, error)
func (s *WorkFlow) NewFlow(tag, name, version, ownerId, userId string) *Flow
func (s *WorkFlow) OnChange(fn func(data et.Json) error) *WorkFlow
func (s *WorkFlow) Run(tag, triggerTag, id, code string, ctx, tags et.Json, await bool, userId string) (et.Json, error)
func (s *WorkFlow) RunInstance(instance *Instance, ctx, tags et.Json, await bool, userId string) (et.Json, error)
func (s *WorkFlow) SetBinding(key string, value any)
func (s *WorkFlow) SetTimeAwait(time time.Duration, userId string) *WorkFlow
func (s *WorkFlow) StopInstance(id string)
func (s *WorkFlow) ToJson() et.Json
func (s *WorkFlow) ToString() string
```

## `jws`

Hub de WebSocket con canales (topic, queue, stack).

Funciones y métodos:

```go
func SendError(socket *websocket.Conn, err error) error
func Upgrader(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error)
func (s *Client) Send(tp int, bt []byte)
func (s *Client) SendError(err error)
func (s *Client) SendHola()
func (s *Client) SendMessage(message interface{})
func (s *Client) ToJson() et.Json
func New() *Hub
func (s *Hub) Close()
func (s *Hub) Connect(socket *websocket.Conn, ctx context.Context) (*Client, error)
func (s *Hub) OnChannel(fn func(Channel))
func (s *Hub) OnConnection(fn func(*Client))
func (s *Hub) OnDisconnection(fn func(*Client))
func (s *Hub) OnListener(fn func(*Client, []byte))
func (s *Hub) OnPublish(fn func(ch Channel, ms Message))
func (s *Hub) OnRemove(fn func(string))
func (s *Hub) OnSend(fn func(to string, ms Message))
func (s *Hub) Publish(channel string, message Message) ([]string, error)
func (s *Hub) Queue(channel string) *Channel
func (s *Hub) Remove(channel string) error
func (s *Hub) SendTo(to []string, message Message) ([]string, error)
func (s *Hub) SetDebug(debug bool)
func (s *Hub) Stack(channel string) *Channel
func (s *Hub) Start()
func (s *Hub) Subscribe(channel string, subscribe string) error
func (s *Hub) Topic(channel string) *Channel
func (s *Hub) Unsubscribe(cache string, subscribe string) error
func DecodeMessage(data []byte) (Message, error)
func NewMessage(from et.Json, to []string) Message
func (s *Message) Bytes() ([]byte, error)
func (s *Message) ToJson() et.Json
func (s *Message) ToString() string
```

## `jwsp`

WhatsApp Business (Graph API de Meta).

Funciones y métodos:

```go
func NewSender(token, phoneNumberId string) *Whatsapp
func (s *Whatsapp) Debug() *Whatsapp
func (s *Whatsapp) IsTest() *Whatsapp
func (s *Whatsapp) SendAudioMessageById(to, audioObjectID string) (et.Json, error)
func (s *Whatsapp) SendAudioMessageByURL(to, url string) (et.Json, error)
func (s *Whatsapp) SendCatalog(to, Text, footer, section string) (et.Json, error)
func (s *Whatsapp) SendCatalogTemplate(to, template, parameter, section string) (et.Json, error)
func (s *Whatsapp) SendContact(to, address, contact, email, phone, tp, url string) (et.Json, error)
func (s *Whatsapp) SendDocumentById(to, documentObjectID, documentCaptionText, documentFilename string) (et.Json, error)
func (s *Whatsapp) SendDocumentByURL(to, url, documentCaptionText string) (et.Json, error)
func (s *Whatsapp) SendImageMessageById(to, imageObjectID string) (et.Json, error)
func (s *Whatsapp) SendImageMessageByURL(to, url string) (et.Json, error)
func (s *Whatsapp) SendList(to, text, header, footer, button, section string) (et.Json, error)
func (s *Whatsapp) SendLocation(to, location string) (et.Json, error)
func (s *Whatsapp) SendReplyAudioMessageById(to, audioObjectID, messageID string) (et.Json, error)
func (s *Whatsapp) SendReplyAudioMessageByURL(to, url, messageID string) (et.Json, error)
func (s *Whatsapp) SendReplyButton(to, text string, buttons []Button) (et.Json, error)
func (s *Whatsapp) SendReplyContact(to, messageID, address, contact, email, phone, tp, url string) (et.Json, error)
func (s *Whatsapp) SendReplyDocumentById(to, messageID, documentObjectID, documentCaptionText, documentFilename string) (et.Json, error)
func (s *Whatsapp) SendReplyDocumentByURL(to, messageID, url, documentCaptionText string) (et.Json, error)
func (s *Whatsapp) SendReplyImageMessageById(to, imageObjectID, messageID string) (et.Json, error)
func (s *Whatsapp) SendReplyImageMessageByURL(to, url, messageID string) (et.Json, error)
func (s *Whatsapp) SendReplyList(to, messageId, text, header, footer, button, section string) (et.Json, error)
func (s *Whatsapp) SendReplyLocation(to, messageID, location string) (et.Json, error)
func (s *Whatsapp) SendReplyStickerMessageById(to, mediaObjectID, messageID string) (et.Json, error)
func (s *Whatsapp) SendReplyStickerMessageByURL(to, messageID, url string) (et.Json, error)
func (s *Whatsapp) SendReplyTextMessage(to, message_id, text string) (et.Json, error)
func (s *Whatsapp) SendReplyVideoMessageById(to, messageID, videoCaptionText, videoObjectID string) (et.Json, error)
func (s *Whatsapp) SendReplyVideoMessageByURL(to, url, videoCaptionText string) (et.Json, error)
func (s *Whatsapp) SendReplyWithReactionMessage(to, messageId, emoji string) (et.Json, error)
func (s *Whatsapp) SendSingleProduct(to, Text, footer, section string) (et.Json, error)
func (s *Whatsapp) SendStickerMessageById(to, mediaObjectID string) (et.Json, error)
func (s *Whatsapp) SendStickerMessageByURL(to, url string) (et.Json, error)
func (s *Whatsapp) SendTemplate(to, parameter, template string) (et.Json, error)
func (s *Whatsapp) SendTemplateInteractive(to, parameter, template string) (et.Json, error)
func (s *Whatsapp) SendTemplateMedia(to, parameter, template string) (et.Json, error)
func (s *Whatsapp) SendTextMessage(to, text string) (et.Json, error)
func (s *Whatsapp) SendTextMessageWithPreviewURL(to, url string) (et.Json, error)
func (s *Whatsapp) SendVideoMessageById(to, videoCaptionText, videoObjectID string) (et.Json, error)
func (s *Whatsapp) SendVideoMessageByURL(to, url, videoCaptionText string) (et.Json, error)
func (s *Whatsapp) SendmultiProduct(to, Text, header, footer, section string) (et.Json, error)
func (s *Whatsapp) SetEventHandler(fn func(et.Json)) *Whatsapp
func (s *Whatsapp) SetEventHandlerError(fn func(error)) *Whatsapp
func (s *Whatsapp) SetVerifyToken(verifyToken string) *Whatsapp
func (s *Whatsapp) Webhooks(w http.ResponseWriter, r *http.Request)
```

## `jwt`

Tokens guardados en `cache`.

Funciones y métodos:

```go
func DeleteTokeByToken(token string) error
func DeleteToken(app, device, sessionID string) error
func GetKey(app, device, userId string) string
func GetToken(key string) (string, bool, error)
func NewAppToken(app, device string, duration time.Duration) (string, error)
func NewAuthentication(app, device, userId, name string, duration time.Duration) (string, error)
func NewAuthorization(app, device, userId, name, tenantId, roleId string, duration time.Duration) (string, error)
func NewEphemeralToken(app, device, userId, name string, duration time.Duration) (string, error)
func NewToken(app, device, userId, name string, payload et.Json, duration time.Duration) (string, error)
func RenewToken(token string, duration time.Duration) (string, error)
func SetToken(app, device, sessionID, token string, duration time.Duration) error
func Validate(token string) (*claim.Claim, error)
```

## `logs`

Logging por niveles; `Fatal` termina el proceso.

Funciones y métodos:

```go
func Alert(err error) error
func Alertf(format string, args ...any) error
func Alertm(message string) error
func Debug(v ...any)
func Debugf(format string, args ...any)
func Debugger(v ...any)
func Error(err error) error
func Errorf(format string, args ...any) error
func Errorm(message string) error
func Fatal(err error) error
func Fatalm(message string) error
func Info(v ...any)
func Infof(format string, args ...any)
func Log(kind string, args ...any) error
func Logf(kind string, format string, args ...any)
func Panic(err error) error
func Panicf(format string, args ...any) error
func Ping()
func Pong()
func Rpc(args ...any) error
func Tracer(kind string, err error) error
func Warn(message string)
```

## `mem`

Caché en memoria con expiración.

Funciones y métodos:

```go
func Clear(match string)
func Delete(key string) bool
func Empty()
func Exists(key string) bool
func Get(key string) (interface{}, bool)
func GetArrayJson(key string) ([]et.Json, bool, error)
func GetBool(key string) (bool, bool, error)
func GetDuration(key string) (time.Duration, bool, error)
func GetFloat64(key string) (float64, bool, error)
func GetInt(key string) (int, bool, error)
func GetInt64(key string) (int64, bool, error)
func GetJson(key string) (et.Json, bool, error)
func GetStr(key string) (string, bool, error)
func GetTime(key string) (time.Time, bool, error)
func Keys() []string
func Len() int
func More(key string, expiration time.Duration) (int64, error)
func Values() []string
func GetEntry(key string) (*Entry, bool)
func New(key string, value interface{}, expiration time.Duration) (*Entry, error)
func Set(key string, value interface{}, expiration time.Duration) (*Entry, error)
func (s *Entry) ArrayDuration() ([]time.Duration, error)
func (s *Entry) ArrayFloat() ([]float64, error)
func (s *Entry) ArrayInt() ([]int, error)
func (s *Entry) ArrayJson() ([]et.Json, error)
func (s *Entry) ArrayMap() ([]map[string]interface{}, error)
func (s *Entry) ArrayStr() ([]string, error)
func (s *Entry) ArrayTime() ([]time.Time, error)
func (s *Entry) Bool() (bool, error)
func (s *Entry) Duration() (time.Duration, error)
func (s *Entry) Float() (float64, error)
func (s *Entry) Get() []byte
func (s *Entry) Int() (int, error)
func (s *Entry) Int64() (int64, error)
func (s *Entry) Json() (et.Json, error)
func (s *Entry) Map() (map[string]interface{}, error)
func (s *Entry) Set(value interface{}, expiration time.Duration) (*Entry, error)
func (s *Entry) Str() (string, error)
func (s *Entry) Time() (time.Time, error)
func Load() *Mem
func (s *Mem) Clear(match string)
func (s *Mem) Delete(key string) bool
func (s *Mem) Empty()
func (s *Mem) Exists(key string) bool
func (s *Mem) Get(key string) (interface{}, bool)
func (s *Mem) GetArrayFloat(key string, def []float64) ([]float64, bool, error)
func (s *Mem) GetArrayInt(key string, def []int) ([]int, bool, error)
func (s *Mem) GetArrayJson(key string, def []et.Json) ([]et.Json, bool, error)
func (s *Mem) GetArrayStr(key string, def []string) ([]string, bool, error)
func (s *Mem) GetBool(key string, def bool) (bool, bool, error)
func (s *Mem) GetDuration(key string, def time.Duration) (time.Duration, bool, error)
func (s *Mem) GetEntry(key string) (*Entry, bool)
func (s *Mem) GetFloat(key string, def float64) (float64, bool, error)
func (s *Mem) GetInt(key string, def int) (int, bool, error)
func (s *Mem) GetInt64(key string, def int64) (int64, bool, error)
func (s *Mem) GetJson(key string, def et.Json) (et.Json, bool, error)
func (s *Mem) GetStr(key string) (string, bool, error)
func (s *Mem) GetTime(key string, def time.Time) (time.Time, bool, error)
func (s *Mem) Keys() []string
func (s *Mem) Len() int
func (s *Mem) More(key string, expiration time.Duration) (int64, error)
func (s *Mem) Set(key string, value interface{}, expiration time.Duration) (*Entry, error)
func (s *Mem) Type() string
func (s *Mem) Values() []string
func NewPeticiones(capacity, timeWait int) *Peticiones
func (c *Peticiones) Ejecucion(executeFc func(params et.Json) (et.Items, error), params et.Json) (et.Items, error)
func (c *Peticiones) GetConfig() et.Json
func (c *Peticiones) GetPeticiones() int
```

## `middleware`

Middlewares HTTP: autenticación, CORS, logger, request id, recoverer, métricas.

Interfaces:

```go
type LogEntry interface {
	Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{})
	Panic(v interface{}, stack []byte)
}

type LogFormatter interface {
	NewLogEntry(r *http.Request) LogEntry
}

type LoggerInterface interface {
	Print(v ...interface{})
}

type WrapResponseWriter interface {
	http.ResponseWriter
	// Status returns the HTTP status of the request, or 0 if one has not
	// yet been sent.
	Status() int
	// BytesWritten returns the total number of bytes sent to the client.
	BytesWritten() int
	// Tee causes the response body to be written to the given io.Writer in
	// addition to proxying the writes through. Only one io.Writer can be
	// tee'd to at once: setting a second one will overwrite the first.
	// Writes will be sent to the proxy before being written to this
	// io.Writer. It is illegal for the tee'd writer to be modified
	// concurrently with writes.
	Tee(io.Writer)
	// Unwrap returns the original proxied target.
	Unwrap() http.ResponseWriter
}
```

Funciones y métodos:

```go
func AllowAll(allowedOrigins []string) *cors.Cors
func Authentication(next http.Handler) http.Handler
func GetBearerToken(r *http.Request) (string, error)
func GetReqID(ctx context.Context) string
func Logger(next http.Handler) http.Handler
func NextRequestID() uint64
func PrintPrettyStack(rvr interface{})
func PushTelemetry(data et.Json)
func PushTelemetryLog(data string)
func PushTelemetryOverflow(data et.Json)
func PushTokenLastUse(data et.Json)
func Recoverer(next http.Handler) http.Handler
func RequestID(next http.Handler) http.Handler
func RequestLogger(f LogFormatter) func(next http.Handler) http.Handler
func WithLogEntry(r *http.Request, entry LogEntry) *http.Request
func (l *DefaultLogFormatter) NewLogEntry(r *http.Request) LogEntry
func GetLogEntry(r *http.Request) LogEntry
func GetMetrics(r *http.Request) *Metrics
func NewMetric(r *http.Request) *Metrics
func NewRpcMetric(method string) *Metrics
func (s *Metrics) CallLatency()
func (s *Metrics) CallResponseTime()
func (s *Metrics) CallSearchTime()
func (s *Metrics) DoneHTTP(rw *ResponseWriterWrapper) et.Json
func (s *Metrics) DoneRpc(r any) et.Json
func (s *Metrics) HTTPError(w http.ResponseWriter, r *http.Request, statusCode int, message string) error
func (s *Metrics) ITEM(w http.ResponseWriter, r *http.Request, statusCode int, dt et.Item) error
func (s *Metrics) ITEMS(w http.ResponseWriter, r *http.Request, statusCode int, dt et.Items) error
func (s *Metrics) JSON(w http.ResponseWriter, r *http.Request, statusCode int, dt interface{}) error
func (s *Metrics) RESULT(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) error
func (s *Metrics) SetPath(val string)
func (s *Metrics) ToJson() et.Json
func (s *Metrics) WriteResponse(w http.ResponseWriter, r *http.Request, statusCode int, e []byte) error
func (rw *ResponseWriterWrapper) Write(b []byte) (int, error)
func (rw *ResponseWriterWrapper) WriteHeader(code int)
func NewWrapResponseWriter(w http.ResponseWriter, protoMajor int) WrapResponseWriter
```

## `msg`

Constantes de mensajes compartidas.

Sin funciones exportadas (solo constantes).

## `queue`

Cola genérica por lotes: despacha al juntar `maxEvents` o al pasar `period`.

Funciones y métodos:

```go
func New[T any](queueSize int, maxEvents int, period time.Duration, handler func(context.Context, []T) error) *Queue[T]
func (q *Queue[T]) Close()
func (q *Queue[T]) Len() int
func (q *Queue[T]) Push(item T)
```

## `race`

Valor protegido por mutex.

Funciones y métodos:

```go
func NewValue(value interface{}) *Value
func (s *Value) Array() []interface{}
func (s *Value) ArrayRange(f func(key, value interface{}) bool)
func (s *Value) Bool() bool
func (s *Value) Delete()
func (s *Value) Float64() float64
func (s *Value) Float64Array() []float64
func (s *Value) Get() interface{}
func (s *Value) Increase(n int)
func (s *Value) Int() int
func (s *Value) IntArray() []int
func (s *Value) IsNil() bool
func (s *Value) Map() map[string]interface{}
func (s *Value) MapRange(f func(key, value interface{}) bool)
func (s *Value) Range(f func(key, value interface{}) bool)
func (s *Value) Set(value interface{})
func (s *Value) String() string
func (s *Value) StringArray() []string
func (s *Value) Time() time.Time
```

## `reg`

IDs: UUID, ULID, XID, Snowflake.

Funciones y métodos:

```go
func GenHashKey(args ...interface{}) string
func GenIndex() int64
func GenKey(args ...interface{}) string
func GenSnowflake() string
func GenULID(tag string) string
func GenUUId(tag string) string
func GenXID(tag string) string
func GetULID(id string) string
func GetUUID(id string) string
func GetXID(id string) string
func TagULID(tag, id string) string
func TagUUID(tag, id string) string
func TagXID(tag, id string) string
func ULID() string
func UUID() string
func XID() string
```

## `request`

Lectura del request entrante y cliente HTTP saliente (`Fetch`, `Get`, `Post`…).

Funciones y métodos:

```go
func App(r *http.Request) string
func Delete(uRL string, header et.Json) (*Body, Status)
func DeleteWithTimeout(uRL string, header et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status)
func DeleteWithTls(uRL string, header et.Json, tlsConfig *tls.Config) (*Body, Status)
func DeleteWithTlsTimeout(uRL string, header et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status)
func Device(r *http.Request) string
func Duration(r *http.Request) time.Duration
func Fetch(method, uRL string, header, body et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status)
func FetchWithTls(method, uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status)
func Get(uRL string, header et.Json) (*Body, Status)
func GetBody(r *http.Request) (et.Json, error)
func GetStr(r *http.Request) (string, error)
func GetWithTimeout(uRL string, header et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status)
func GetWithTls(uRL string, header et.Json, tlsConfig *tls.Config) (*Body, Status)
func GetWithTlsTimeout(uRL string, header et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status)
func Http(method, uRL string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status)
func HttpCtxWithTimeout(ctx context.Context, method, uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status)
func HttpWithContext(ctx context.Context, method, uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status)
func HttpWithTimeout(method, uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status)
func IsStatusOk(status int) bool
func Name(r *http.Request) string
func NewTlsConfig(caFile, certFile, keyFile string) (*tls.Config, error)
func Options(uRL string, header et.Json) (*Body, Status)
func OptionsWithTimeout(uRL string, header et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status)
func OptionsWithTls(uRL string, header et.Json, tlsConfig *tls.Config) (*Body, Status)
func OptionsWithTlsTimeout(uRL string, header et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status)
func Patch(uRL string, header, body et.Json) (*Body, Status)
func PatchWithTimeout(uRL string, header, body et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status)
func PatchWithTls(uRL string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status)
func PatchWithTlsTimeout(uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status)
func Payload(r *http.Request) et.Json
func Post(uRL string, header, body et.Json) (*Body, Status)
func PostWithTimeout(uRL string, header, body et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status)
func PostWithTls(uRL string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status)
func PostWithTlsTimeout(uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status)
func Put(uRL string, header, body et.Json) (*Body, Status)
func PutWithTimeout(uRL string, header, body et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status)
func PutWithTls(uRL string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status)
func PutWithTlsTimeout(uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status)
func RoleID(r *http.Request) string
func ServiceId(r *http.Request) string
func SetApp(ctx context.Context, app string) context.Context
func SetDevice(ctx context.Context, device string) context.Context
func SetDuration(ctx context.Context, duration time.Duration) context.Context
func SetName(ctx context.Context, name string) context.Context
func SetPayload(ctx context.Context, payload et.Json) context.Context
func SetRoleID(ctx context.Context, roleId string) context.Context
func SetServiceId(ctx context.Context, serviceId string) context.Context
func SetTenantID(ctx context.Context, tenantId string) context.Context
func SetUserID(ctx context.Context, userId string) context.Context
func TenantID(r *http.Request) string
func UserID(r *http.Request) string
func ReadBody(body io.ReadCloser) (*Body, error)
func (b Body) ToArrayJson() ([]et.Json, error)
func (b Body) ToBool() (bool, error)
func (b Body) ToFloat() (float64, error)
func (b Body) ToInt() (int, error)
func (b Body) ToInt64() (int64, error)
func (b Body) ToItem() (et.Item, error)
func (b Body) ToItems() (et.Items, error)
func (b Body) ToJson() (et.Json, error)
func (b Body) ToString() string
func (b Body) ToTime() (time.Time, error)
func (s ContextKey) Duration(ctx context.Context, def time.Duration) time.Duration
func (s ContextKey) Json(ctx context.Context, def et.Json) et.Json
func (s ContextKey) String(ctx context.Context, def string) string
func (s Status) ToJson() et.Json
func (s Status) ToString() string
func Query(r *http.Request, key string) *Value
func URLParam(r *http.Request, key string) *Value
func (s *Value) Array() []any
func (s *Value) ArrayFloat() []float64
func (s *Value) ArrayInt() []int
func (s *Value) ArrayJson() []et.Json
func (s *Value) ArrayString() []string
func (s *Value) Bool() bool
func (s *Value) DateTime() time.Time
func (s *Value) Float() float64
func (s *Value) Int() int
func (s *Value) Object() et.Json
func (s *Value) Str() string
```

## `resilience`

Reintentos con intentos e intervalo; usado por `jwf`.

Funciones y métodos:

```go
func (s *Instance) IsDone() bool
func (s *Instance) IsFailed() bool
func (s *Instance) Run() ([]any, error)
func (s *Instance) ToJson() et.Json
func (s *Instance) ToString() string
func New() (*Resilience, error)
func (s *Resilience) LoadInstance(params Params) (*Instance, error)
func (s *Resilience) Stop(id string) error
func (s *Resilience) ToJson() et.Json
```

## `response`

Respuestas HTTP unificadas (`ITEM`, `ITEMS`, `LIST`, `HTTPError`).

Funciones y métodos:

```go
func ANY(w http.ResponseWriter, r *http.Request, statusCode int, result interface{}) error
func DATA(w http.ResponseWriter, r *http.Request, statusCode int, data et.Json) error
func Forbidden(w http.ResponseWriter, r *http.Request)
func GetArray(r *http.Request) ([]et.Json, error)
func GetBody(r *http.Request) (et.Json, error)
func GetParam(r *http.Request, key string) string
func GetQuery(r *http.Request) et.Json
func HTTPAlert(w http.ResponseWriter, r *http.Request, message string) error
func HTTPError(w http.ResponseWriter, r *http.Request, statusCode int, message string) error
func ITEM(w http.ResponseWriter, r *http.Request, statusCode int, data et.Item) error
func ITEMS(w http.ResponseWriter, r *http.Request, statusCode int, data et.Items) error
func InternalServerError(w http.ResponseWriter, r *http.Request, err error)
func JSON(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) error
func LIST(w http.ResponseWriter, r *http.Request, statusCode int, data et.List) error
func RESULT(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) error
func ScanJson(value map[string]interface{}) (et.Json, error)
func ScanStr(value string) (et.Json, error)
func Stream(w http.ResponseWriter, r *http.Request, rows int, getData DataFunction)
func Unauthorized(w http.ResponseWriter, r *http.Request)
func WriteResponse(w http.ResponseWriter, statusCode int, e []byte) error
```

## `router`

Router `chi` con catálogo de endpoints: `NewApi` + `Public`/`Protected`/`Session(group, method, path, name, handler)`.

Interfaces:

```go
type Router interface {
	UseAutentication(fn func(http.Handler) http.Handler)
	Protect(method, path string, handler func(http.ResponseWriter, *http.Request))
	Public(method, path string, handler func(http.ResponseWriter, *http.Request))
	With(method, path string, middlewares []func(http.Handler) http.Handler, handler func(http.ResponseWriter, *http.Request))
}
```

Funciones y métodos:

```go
func GetRoutes() map[string]et.Json
func HttpDelete(w http.ResponseWriter, r *http.Request)
func HttpGet(w http.ResponseWriter, r *http.Request)
func HttpSet(w http.ResponseWriter, r *http.Request)
func OnDelete(f DeleteFn)
func OnGet(f GetFn)
func OnSet(f SetFn)
func Publish(r *chi.Mux, route Route) *chi.Mux
func PushApiGateway(method, path, resolve string, tpHeader TpHeader, header et.Json, excludeHeader []string, params, body et.Json, version int, packageName string)
func RemoveApiGateway(id string)
func With(r *chi.Mux, route Route, middlewares []func(http.Handler) http.Handler) *chi.Mux
func NewApi(name, path, host string, port, rpc int, version int) *Api
func (s *Api) AddErrorStatusNotFound(statuses ...string)
func (s *Api) Item(status int, fn func(r *http.Request, body et.Json) (et.Item, error)) http.HandlerFunc
func (s *Api) Items(fn func(r *http.Request, body et.Json) (et.Items, error)) http.HandlerFunc
func (s *Api) List(fn func(r *http.Request) (et.List, error)) http.HandlerFunc
func (s *Api) Protected(group, method, path, name string, handler http.HandlerFunc)
func (s *Api) Public(group, method, path, name string, handler http.HandlerFunc)
func (s *Api) RegisterEndpoint(packageName, group, method, path, name string) error
func (s *Api) Session(group, method, path, name string, handler http.HandlerFunc)
func (s *Api) UseAuthentication(middlewares ...func(http.Handler) http.Handler)
func (s *Api) UseAuthorization(authorization func(method, path string) func(http.Handler) http.Handler)
func (s *Api) UseIdenpotency(middlewares ...func(http.Handler) http.Handler)
func (s *Api) UseLimitRate(limitRate int64)
func (s *Api) UseRegisterEndpoint(registerEndpoint func(packageName, group, method, path, name string) error)
func IntToTpHeader(tp int) TpHeader
func ToTpHeader(tp int) TpHeader
func (t TpHeader) String() string
```

## `server`

Servidor HTTP ligero sobre `chi`, sin Redis ni NATS.

Funciones y métodos:

```go
func NewApi(name, path, host string, port int, version string) *Api
func (s *Api) Authentication(r *chi.Mux, route router.Route)
func (s *Api) Authorization(r *chi.Mux, route router.Route)
func (s *Api) Private(r *chi.Mux, route router.Route)
func (s *Api) Public(r *chi.Mux, route router.Route)
func (s *Api) Use(middlewares ...func(http.Handler) http.Handler)
func (s *Api) UseAutentication(middleware func(http.Handler) http.Handler)
func (s *Api) UseAuthorization(middleware func(http.Handler) http.Handler)
func New(name string, port int) *Ettp
func (s *Ettp) Close()
func (s *Ettp) HandleFunc(pattern string, handlerFn http.HandlerFunc)
func (s *Ettp) Mount(pattern string, handler http.Handler)
func (s *Ettp) NotFound(handlerFn http.HandlerFunc)
func (s *Ettp) OnClose(onClose func())
func (s *Ettp) OnStart(onStart func())
func (s *Ettp) Start()
func (s *Ettp) Use(middlewares ...func(http.Handler) http.Handler)
```

## `service`

Envío multicanal (email, SMS, WhatsApp) con OTP y almacenamiento local.

Interfaces:

```go
type SenderEmail interface {
	SendEmail(from et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tpMessage string) (et.Item, error)
}

type SenderPushNotification interface {
	SendPushNotification(deviceToken string, title string, body string, tpMessage string) (et.Item, error)
}

type SenderSMS interface {
	SendSMS(contactNumbers []string, content string, params et.Json, tpMessage string) (et.Item, error)
}

type SenderWhatsapp interface {
	SendWhatsapp(contactNumbers []string, content string, params []et.Json, tpMessage string) (et.Item, error)
	SendWhatsappByTemplateId(contactNumbers []string, templateId string, params []et.Json, tpMessage string) (et.Item, error)
}

type Storage interface {
	Uploader(bucket, filename, contentType string, contentFile []byte) (et.Item, error)
	UploaderFile(r *http.Request, folder, name string) (et.Item, error)
}
```

Funciones y métodos:

```go
func LoadLocalStorage(tenantId, path string) *LocalStorage
func (s *LocalStorage) Delete(bucket, filePath string) (et.Item, error)
func (s *LocalStorage) UploaderB64(bucket, b64, fileName, contentType string) (et.Item, error)
func (s *LocalStorage) UploaderFile(r *http.Request, bucket, folder, fileName string) (et.Item, error)
func GetSender(tenantId string) (*Send, error)
func LoadSend(tenantId, name, email string) (*Send, error)
func (s *Send) AddSenderEmail(level Level, sender SenderEmail) *Send
func (s *Send) AddSenderPushNotification(level Level, sender SenderPushNotification) *Send
func (s *Send) AddSenderSMS(level Level, sender SenderSMS) *Send
func (s *Send) AddSenderWhatsapp(level Level, sender SenderWhatsapp) *Send
func (s *Send) OnSender(fn func(et.Item, error)) *Send
func (s *Send) SendEmail(from et.Json, to []et.Json, subject string, htmlContent string, params et.Json, tp TpMessage) (et.Item, error)
func (s *Send) SendOTPEmail(key, sender string, name, email string, length int, duration time.Duration, htmlContent string) (et.Item, error)
func (s *Send) SendOTPSMS(key, sender, countryCode, phoneNumber string, length int, duration time.Duration, content string) (et.Item, error)
func (s *Send) SendSMS(contactNumbers []string, content string, params et.Json, tpMessage TpMessage) (et.Item, error)
func (s *Send) SendWhatsapp(templateId string, contactNumbers []string, params []et.Json, tp TpMessage) (et.Item, error)
func (s *Send) VerifyOTP(key string, otp string) (bool, error)
func IntToTpMessage(i int) TpMessage
func (tp TpMessage) String() string
```

## `stdrout`

Salida coloreada y trazas usadas por `logs`.

Interfaces:

```go
type Stdout interface {
	Notify(kind string, message string)
}
```

Funciones y métodos:

```go
func CW(w io.Writer, color []byte, format string, args ...interface{})
func Color(s *string, color string, format string, args ...interface{}) *string
func ErrorTraces(err error) []string
func GetFunctionName(idx int) string
func Printl(kind string, color string, args ...any) string
func SetStdout(v Stdout)
func Traces(kind, color string, err error) error
```

## `stores`

Stores sobre `jsql`: instancias, autorización (ACL), config por tenant, estado de formularios.

Funciones y métodos:

```go
func DefineAuthorization(db *DB, tenantId, schema string) (*Authorization, error)
func (s *Authorization) Author(profileId, method, path string) (bool, error)
func (s *Authorization) Query(query et.Json) (et.Items, error)
func (s *Authorization) RemoveAuthor(profileId, method, path string) error
func (s *Authorization) SetAuthor(profileId, method, path string) error
func (s *Authorization) SetPath(method, path string) error
func DefineConfig(db *DB, tenantId, schema, stage, tag string) (*Config, error)
func (s *Config) DeleteConfig(tag, stage string) error
func (s *Config) Get(name string, def interface{}) interface{}
func (s *Config) SetConfig(tag, stage string, params et.Json) error
func DefineDbs(db *DB, schema string) (*Dbs, error)
func DefineFormState(db *DB, tenantId, schema string) (*FormState, error)
func (s *FormState) Delete(id string) error
func (s *FormState) Get(id string, dest et.Json) (bool, error)
func (s *FormState) Query(query et.Json) (et.Items, error)
func (s *FormState) Set(id, tag, ownerId string, data et.Json) error
func DefineInstance(db *DB, tenantId, schema string) (*Instance, error)
func (s *Instance) Delete(id string) error
func (s *Instance) Get(id string, dest any) (bool, error)
func (s *Instance) Query(query et.Json) (et.Items, error)
func (s *Instance) Set(tag, id, ownerId string, obj any) error
```

## `strs`

Utilidades de texto.

Funciones y métodos:

```go
func ApendAny(space string, args ...any) string
func Append(str1, str2, sp string) string
func AppendAny(val1, val2 interface{}, sp string) interface{}
func Change(str string, olds []string, news []string) string
func Contains(str string, substr string) bool
func DaskSpace(str string) string
func Format(format string, args ...any) string
func FormatDateTime(format string, value time.Time) string
func FormatLowCase(format string, args ...any) string
func FormatSerie(format string, num int64) string
func FormatUppCase(format string, args ...any) string
func GetSplitIndex(str, sep string, idx int) string
func HtmlToText(html string) string
func IsEmpty(str string) bool
func JoinQuoted(items []string, sep string) string
func Lowcase(s string) string
func MaskToken(token string, length int) string
func Name(str string) string
func NotSpace(str string) string
func Parse(str string, vars et.Json) string
func RemoveAcents(str string) string
func Replace(str string, old string, new string) string
func ReplaceAll(str string, olds []string, new string) string
func Same(str1, str2 string) bool
func Split(str, sep string) []string
func StrToBool(val string) (bool, error)
func StrToTime(val string) (time.Time, error)
func Titlecase(str string) string
func Trim(str string) string
func Uppcase(s string) string
```

## `timezone`

Hora local y formatos.

Funciones y métodos:

```go
func Add(d time.Duration) time.Time
func Format(t time.Time, layout Layout) string
func FormatMDYYYY(value string) string
func Location() *time.Location
func Now() time.Time
func NowStr() string
func Parse(layout Layout, value string) (time.Time, error)
```

## `units`

Cantidades y conversión de unidades.

Funciones y métodos:

```go
func Load(val any) (*Quantity, error)
func NewQuantity(val float64, unit TypeUnity) *Quantity
func (s *Quantity) Load(val interface{}) error
func (s *Quantity) To(un TypeUnity) error
func (s *Quantity) ToJson() et.Json
func (s *Quantity) ToStr() string
func StrToUnity(s string) TypeUnity
func (s TypeUnity) Str() string
```

## `utility`

Cripto, hashing, validaciones de formato, base64, algoritmos.

Funciones y métodos:

```go
func Add[T any](slice []T, item T) []T
func Address(host string, port int) string
func App() context.Context
func AppWait()
func BannerTitle(name string, size int) string
func BinarySearch(arr []string, target string) int
func Contains(v interface{}, vals ...any) bool
func DecryptAES(value, secret string) (string, error)
func Dijkstra(graph [][]int, start int) []int
func DivInt(a, b int64) int64
func DivNum(a, b float64) float64
func EncryptAES(value, secret string) (string, error)
func EncryptMD5(value string) (string, error)
func ExtractMencion(str string) []string
func FromBase64(data string) (string, error)
func FromBase64Raw(data string) string
func GenId(id string) string
func GenKey(id string) string
func GeneratePortNumber() int
func GetGoMod(atrib string) (string, error)
func GetOTP(length int) string
func GetPidByPort(port int) int
func GetRandom(charset string, length int) string
func GetRandomString(length int) string
func Hash(password string) (string, error)
func HashSHA1(value string) (string, error)
func HashSHA256(value string) (string, error)
func HashSHA512(value string) (string, error)
func InInt(val string, in []string) bool
func InStr(val string, in []string) bool
func Match(hashPassword, password string) bool
func Md5(password string) string
func MinDistance(dist []int, visited []bool) int
func More(tag string, expiration time.Duration) int64
func Normalize(input string) string
func Now() time.Time
func NowStr() string
func ParamQuote(str string, args ...any) string
func Params(str string, args ...any) string
func QuickSort(arr []int) []int
func Quote(val interface{}) any
func Serialize(v any) ([]byte, error)
func SetQuotedChar(char string)
func Sha256(password string) string
func TimeDifference(dateInt, dateEnd any) time.Duration
func ToBase64(data string) string
func ToBase64Raw(data string) string
func ToJson(v any) et.Json
func ToSerialize(v any) ([]byte, error)
func UUID() string
func UUIndex(tag string) int64
func Unquote(val interface{}) any
func ValidCode(val string) bool
func ValidEmail(val string) bool
func ValidId(val string) bool
func ValidIn(val string, min int, in []string) bool
func ValidInt(val int, notIn []int) bool
func ValidKey(val string) bool
func ValidName(val string) bool
func ValidNum(val float64, notIn []float64) bool
func ValidPhone(val string) bool
func ValidStr(val string, min int, notIn []string) bool
func ValidUUID(val string) bool
func ValidWord(word string) bool
func NewList() List
func (s *List) Add(val interface{})
func (s *List) Contains(val interface{}) bool
func (s *List) IndexOf(val interface{}) int
func (s *List) Remove(val interface{})
func (s *List) Size() int
```

## `validator`

Validación fluida de `et.Json` (mensajes según `LANG`).

Funciones y métodos:

```go
func (s *Condition) Between(min, max float64) *Condition
func (s *Condition) Field(name string) *Condition
func (s *Condition) IsEmail() *Condition
func (s *Condition) IsLetters() *Condition
func (s *Condition) IsNumbers() *Condition
func (s *Condition) IsPhone() *Condition
func (s *Condition) IsSpecialCharacters() *Condition
func (s *Condition) Max(max float64) *Condition
func (s *Condition) MaxLength(maxLength int) *Condition
func (s *Condition) Min(min float64) *Condition
func (s *Condition) MinLength(minLength int) *Condition
func (s *Condition) NotEmpty() *Condition
func (s *Condition) Pattern(pattern string) *Condition
func (s *Condition) Required() *Condition
func (s *Condition) Validate(value et.Json) (bool, error)
func New() *Validator
func (s *Validator) Field(name string) *Condition
func (s *Validator) NotEmpty() *Validator
func (s *Validator) Validate(value et.Json) (bool, error)
```

## `xls`

Leer y escribir hojas de cálculo / CSV.

Funciones y métodos:

```go
func IndexOf(list []string, value string) int
func NewXls(data []et.Json, nameSheet string, columns ...Column) *Xls
func (s *Xls) Add(data []et.Json, nameSheet string, columns []Column) *Xls
func (s *Xls) ToFile(path string) error
func (s *Xls) ToHttp(w http.ResponseWriter, filename string) error
func (s *Xls) ToWriter(w io.Writer) error
func ReadXls(data []byte) (*XlsReader, error)
func ReadXlsFile(path string) (*XlsReader, error)
func ReadXlsMultipart(fileData multipart.File) (*XlsReader, error)
func (s *XlsReader) Close() error
func (s *XlsReader) GetRows(nameSheet string) ([][]string, error)
func (s *XlsReader) GetSheet(nameSheet string, columns []string) ([]et.Json, error)
```

