package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/jsql/drivers/oracle"
	"github.com/cgalvisleon/et/jsql/drivers/postgres"
	"github.com/cgalvisleon/et/jsql/drivers/sqlite"
)

// sections are the 11 groups of catalog.go, in the same order.
var sections = []string{
	"1. Conexión y base de datos",
	"2. Definición de modelos (DDL)",
	"3. Relaciones y campos calculados",
	"4. Consultas",
	"5. Condiciones",
	"6. Comandos",
	"7. Triggers",
	"8. Transacciones",
	"9. Series",
	"10. Auditoría",
	"11. Utilidades",
}

/**
* recorder: Wraps a real driver and keeps every SQL statement it generates (DDL, queries and
* commands), so each test case can report the SQL it produced.
**/
type recorder struct {
	jsql.Driver
	mu  sync.Mutex
	sql []string
}

func (s *recorder) add(kind, sql string, err error) {
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sql = append(s.sql, fmt.Sprintf("-- %s\n%s", kind, sql))
}

func (s *recorder) take() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := s.sql
	s.sql = nil
	return result
}

func (s *recorder) Load(model *jsql.Model) (string, error) {
	sql, err := s.Driver.Load(model)
	s.add("DDL", sql, err)
	return sql, err
}

func (s *recorder) Query(query *jsql.Query) (string, error) {
	sql, err := s.Driver.Query(query)
	s.add("QUERY", sql, err)
	return sql, err
}

func (s *recorder) Command(command *jsql.Command) (string, error) {
	sql, err := s.Driver.Command(command)
	s.add(strings.ToUpper(string(command.Type)), sql, err)
	return sql, err
}

/**
* caseResult: Outcome of one functionality check. Status is pass, fail, gap (a documented
* limitation of jsql) or skip (not applicable in the test environment).
**/
type caseResult struct {
	Section string   `json:"section"`
	Case    string   `json:"case"`
	Status  string   `json:"status"`
	Note    string   `json:"note,omitempty"`
	Output  any      `json:"output,omitempty"`
	SQL     []string `json:"sql,omitempty"`
}

/**
* suite: State shared by the cases of one driver.
**/
type suite struct {
	t       *testing.T
	tg      target
	db      *jsql.DB
	rec     *recorder
	schema  string
	results []caseResult
	models  map[string]*jsql.Model
	events  []string
}

/**
* run: Executes one case, recovering panics, and records its status, output and generated SQL.
* @param section int, name string, fn func() (any, error)
**/
func (s *suite) run(section int, name string, fn func() (any, error)) {
	s.rec.take()
	out, err := func() (out any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		return fn()
	}()
	res := caseResult{Section: sections[section-1], Case: name, Status: "pass", Output: out, SQL: s.rec.take()}
	if err != nil {
		res.Status = "fail"
		res.Note = err.Error()
		s.t.Errorf("%s / %s: %v", res.Section, name, err)
	}
	s.results = append(s.results, res)
}

/**
* mark: Records a case that is not executed as a normal check (gap or skip) with its reason.
* @param section int, name, status, note string, fn func() (any, error)
**/
func (s *suite) mark(section int, name, status, note string, fn func() (any, error)) {
	s.rec.take()
	var out any
	if fn != nil {
		out, _ = fn()
	}
	s.results = append(s.results, caseResult{Section: sections[section-1], Case: name, Status: status, Note: note, Output: out, SQL: s.rec.take()})
}

/**
* expect: Returns an error when the normalized JSON content of got differs from want.
* @param label string, want, got any
* @return error
**/
func expect(label string, want, got any) error {
	w := normalize(want, false)
	g := normalize(got, false)
	if !reflect.DeepEqual(w, g) {
		a, _ := json.Marshal(w)
		b, _ := json.Marshal(g)
		return fmt.Errorf("%s: want %s, got %s", label, a, b)
	}
	return nil
}

/**
* ids: Returns the "id" of each row, in order.
* @param items et.Items
* @return []string
**/
func ids(items et.Items) []string {
	result := make([]string, 0, len(items.Result))
	for _, row := range items.Result {
		result = append(result, row.Str("id"))
	}
	return result
}

/**
* rawSelectOne: A driver-specific "SELECT 1" statement.
* @param driver string
* @return string
**/
func rawSelectOne(driver string) string {
	if driver == "oracle" {
		return `SELECT 1 AS "n" FROM DUAL`
	}
	return `SELECT 1 AS n`
}

/**
* recorded: Registers a recorder around the real driver of tg and returns target params that use it.
* @param tg target
* @return target, *recorder
**/
func recorded(tg target) (target, *recorder) {
	var drv jsql.Driver
	switch tg.name {
	case "postgres":
		drv = &postgres.Postgres{}
	case "oracle":
		drv = &oracle.Oracle{}
	default:
		drv = &sqlite.Sqlite{}
	}
	rec := &recorder{Driver: drv}
	name := "rec_" + tg.name
	jsql.Register(name, rec)
	tg.params.Driver = name
	return tg, rec
}

/**
* catalogTargets: The same databases as targets(), in their own schema / file / tables.
* @return []target
**/
func catalogTargets() []target {
	result := []target{}
	for _, tg := range targets() {
		switch tg.name {
		case "sqlite":
			file := filepath.Join(os.TempDir(), "jsql_catalog.db")
			tg.file = file
			tg.params.Name = file
			tg.params.Connection = &jsql.SqliteConection{Name: file, RecordLimit: 1000}
			tg.schema = "jsql_catalog"
		case "postgres":
			tg.schema = "jsql_catalog"
			tg.cleanup = []string{`DROP SCHEMA IF EXISTS jsql_catalog CASCADE`}
		case "oracle":
			tg.cleanup = oracleDrops(tg.schema, "f_users_f_roles", "f_orders_items", "f_orders", "f_users", "f_roles", "f_doc_types",
				"f_products", "f_events", "f_tenant", "f_project", "f_required", "f_strict", "f_child", "f_parent", "series")
		}
		result = append(result, tg)
	}
	return result
}

/**
* TestCatalog: Exercises every section of the jsql catalog against sqlite, postgres and oracle.
* The results (status, output and generated SQL per case) are written to result/<driver>.json and
* result/<driver>.sql, and a comparison of the three drivers to result/summary.md.
**/
func TestCatalog(t *testing.T) {
	all := map[string][]caseResult{}
	for _, base := range catalogTargets() {
		tg, rec := recorded(base)
		t.Run(base.name, func(t *testing.T) {
			db := connect(t, tg)
			s := &suite{t: t, tg: base, db: db, rec: rec, schema: tg.schema, models: map[string]*jsql.Model{}}
			s.connection()
			s.ddl()
			s.relations()
			s.queries()
			s.conditions()
			s.commands()
			s.triggers()
			s.transactions()
			s.series()
			s.audit()
			s.utilities()
			all[base.name] = s.results
			writeResults(t, base.name, s.results)
		})
	}
	writeSummary(t, all)
}

/**
* connection: Section 1 — connection and database.
**/
func (s *suite) connection() {
	db := s.db
	s.run(1, "Register + ConnectTo", func() (any, error) {
		return db.ToJson().Str("driver"), nil
	})
	s.run(1, "Init (idempotente)", func() (any, error) {
		return nil, db.Init()
	})
	s.run(1, "NewDB", func() (any, error) {
		other, err := jsql.NewDB(s.tgParams())
		if err != nil {
			return nil, err
		}
		return other.ToJson().Str("name"), nil
	})
	s.run(1, "LoadDb (desde ToJson)", func() (any, error) {
		other, err := jsql.LoadDb(db.ToJson())
		if err != nil {
			return nil, err
		}
		return other.ToJson().Str("driver"), nil
	})
	s.mark(1, "Load / LoadTo", "skip", "leen la configuración de DB_* con envar, que la guarda en caché; se cubren con ConnectTo", nil)
	s.run(1, "Sql", func() (any, error) {
		items, err := db.Sql(rawSelectOne(s.tg.name))
		if err != nil {
			return nil, err
		}
		return items.Result, expect("n", "1", fmt.Sprint(items.Result[0]["n"]))
	})
	s.run(1, "SqlTx + Tx.Commit", func() (any, error) {
		tx := jsql.NewTx()
		items, err := db.SqlTx(tx, rawSelectOne(s.tg.name))
		if err != nil {
			return nil, err
		}
		return items.Count, tx.Commit()
	})
	s.run(1, "Connection.GetParams / SetDatabase / GetDatabase", func() (any, error) {
		conn := s.tgParams().Connection
		name := conn.GetDatabase()
		conn.SetDatabase("other")
		changed := conn.GetDatabase()
		conn.SetDatabase(name)
		if changed != "other" {
			return nil, fmt.Errorf("SetDatabase did not change the database: %q", changed)
		}
		return conn.GetParams().Str("driver"), nil
	})
	s.run(1, "SetDebug / Debug", func() (any, error) {
		db.SetDebug(false)
		return db.ToJson().Str("name"), nil
	})
	s.mark(1, "DB.Query / DB.QueryTx (SQL libre en JSON)", "gap", "jquery.go no genera SQL todavía (brecha #1 de spec.md)", func() (any, error) {
		_, err := db.Query(et.Json{"select": []any{"id"}})
		return fmt.Sprint(err), nil
	})
}

/**
* tgParams: The connection params of the target (with the recorder driver name).
* @return jsql.ConnectParams
**/
func (s *suite) tgParams() jsql.ConnectParams {
	params := s.tg.params
	params.Driver = "rec_" + s.tg.name
	return params
}

/**
* model: Defines and initializes a model with the standard columns plus KEY columns.
* @param name string, columns ...string
* @return *jsql.Model
**/
func (s *suite) model(name string, columns ...string) (*jsql.Model, error) {
	model, err := s.db.DefineModel(s.schema, name, 1, "test")
	if err != nil {
		return nil, err
	}
	for _, column := range columns {
		model.DefineColumn(column, et.KEY, "")
	}
	s.models[name] = model
	return model, nil
}

/**
* ddl: Section 2 — model definition. Creates the models and data used by the next sections.
**/
func (s *suite) ddl() {
	db := s.db
	s.run(2, "Define (declarativo)", func() (any, error) {
		model, err := db.Define(jsql.Define{
			Schema: s.schema, Name: "f_products", Version: 1, SourceField: jsql.SOURCE,
			Columns: []jsql.Column{
				{Name: "id", TypeColumn: jsql.COLUMN, TypeData: et.KEY, Default: ""},
				{Name: "name", TypeColumn: jsql.COLUMN, TypeData: et.TEXT, Default: ""},
				{Name: "category", TypeColumn: jsql.COLUMN, TypeData: et.KEY, Default: ""},
				{Name: "price", TypeColumn: jsql.ATTRIB, TypeData: et.FLOAT, Default: 0},
			},
			PrimaryKeys: []jsql.DefIndex{{Name: "id", Sorted: true}},
			Indexes:     []jsql.DefIndex{{Name: "category", Sorted: true}},
		})
		if err != nil {
			return nil, err
		}
		s.models["f_products"] = model
		return model.Name, model.Init()
	})
	s.run(2, "DefineModel + DefineColumn / DefineAttrib / DefineUnique / DefineHidden", func() (any, error) {
		roles, err := s.model("f_roles", "name")
		if err != nil {
			return nil, err
		}
		orders, err := s.model("f_orders", "user_id")
		if err != nil {
			return nil, err
		}
		orders.DefineAttrib("amount", et.FLOAT, 0)
		items, err := orders.DefineDetail("items", map[string]string{"id": "order_id"}, 30, "product")
		if err != nil {
			return nil, err
		}
		items.DefinePrimaryKey("id", et.KEY, "")
		items.DefineColumn("product", et.TEXT, "")
		s.models["f_orders_items"] = items

		docTypes, err := s.model("f_doc_types", "title")
		if err != nil {
			return nil, err
		}

		users, err := s.model("f_users", "name")
		if err != nil {
			return nil, err
		}
		users.DefineUnique("email", et.TEXT, "")
		users.DefineAttrib("age", et.INT, 0)
		users.DefineHidden("password")
		if _, err := users.DefineMaster("roles", roles, map[string]string{"id": "user_id"}, map[string]string{"id": "role_id"}, []string{"name"}); err != nil {
			return nil, err
		}
		if _, err := users.DefineMaster("main_role", roles, map[string]string{"id": "user_id"}, map[string]string{"id": "role_id"}, []string{"name"}, 1); err != nil {
			return nil, err
		}
		if _, err := users.DefineRollup("tp_doc_title", docTypes, map[string]string{"tp_doc": "id"}, []string{"title"}, jsql.RollupRow); err != nil {
			return nil, err
		}
		if _, err := users.DefineRollup("orders_count", orders, map[string]string{"id": "user_id"}, nil, jsql.RollupCount); err != nil {
			return nil, err
		}
		if _, err := users.DefineRollup("orders_total", orders, map[string]string{"id": "user_id"}, []string{"amount"}, jsql.RollupSum); err != nil {
			return nil, err
		}
		if _, err := users.DefineRollup("last_order", orders, map[string]string{"id": "user_id"}, []string{"id", "amount"}, jsql.RollupObject); err != nil {
			return nil, err
		}
		users.DefineCalcFunc("label", func(tx *jsql.Tx, data et.Json) {
			data["label"] = fmt.Sprintf("%s <%s>", data.Str("name"), data.Str("email"))
		})
		users.DefineCalc("initial", `item.initial = item.name.substring(0, 1);`)

		for _, m := range []*jsql.Model{roles, docTypes, orders, users} {
			if err := m.Init(); err != nil {
				return nil, fmt.Errorf("init %s: %w", m.Name, err)
			}
		}
		return []string{roles.Name, orders.Name, items.Name, users.Name}, nil
	})
	s.run(2, "DefineTenantModel / DefineProjectModel", func() (any, error) {
		tenant, err := db.DefineTenantModel(s.schema, "f_tenant", 1, "test")
		if err != nil {
			return nil, err
		}
		project, err := db.DefineProjectModel(s.schema, "f_project", 1, "test")
		if err != nil {
			return nil, err
		}
		if err := tenant.Init(); err != nil {
			return nil, err
		}
		if err := project.Init(); err != nil {
			return nil, err
		}
		_, okT := tenant.GetColumn(jsql.TENANT_ID)
		_, okP := project.GetColumn(jsql.PROJECT_ID)
		if !okT || !okP {
			return nil, errors.New("tenant_id / project_id not defined")
		}
		return []string{jsql.TENANT_ID, jsql.PROJECT_ID}, nil
	})
	s.run(2, "DefineRequired (rechaza el insert sin el campo)", func() (any, error) {
		model, err := s.model("f_required")
		if err != nil {
			return nil, err
		}
		model.DefineRequired("name", et.TEXT, "")
		if err := model.Init(); err != nil {
			return nil, err
		}
		_, err = model.Insert(et.Json{"id": "r1"}).Exec()
		if err == nil {
			return nil, errors.New("insert without the required field was accepted")
		}
		return err.Error(), nil
	})
	s.run(2, "DefineForeignKeys (rechaza un hijo sin padre)", func() (any, error) {
		parent, err := s.model("f_parent")
		if err != nil {
			return nil, err
		}
		child, err := s.model("f_child", "parent_id")
		if err != nil {
			return nil, err
		}
		child.DefineForeignKeys(parent, map[string]string{"parent_id": "id"}, true, false)
		if err := parent.Init(); err != nil {
			return nil, err
		}
		if err := child.Init(); err != nil {
			return nil, err
		}
		if _, err := parent.Insert(et.Json{"id": "p1"}).Exec(); err != nil {
			return nil, err
		}
		if _, err := child.Insert(et.Json{"id": "c1", "parent_id": "p1"}).Exec(); err != nil {
			return nil, err
		}
		_, err = child.Insert(et.Json{"id": "c2", "parent_id": "missing"}).Exec()
		if err == nil {
			return nil, errors.New("child without parent was accepted")
		}
		return "fk violation", nil
	})
	s.run(2, "Stricted (ignora campos desconocidos)", func() (any, error) {
		model, err := s.model("f_strict", "name")
		if err != nil {
			return nil, err
		}
		model.Stricted()
		if err := model.Init(); err != nil {
			return nil, err
		}
		if _, err := model.Insert(et.Json{"id": "s1", "name": "x", "extra": "ignored"}).Exec(); err != nil {
			return nil, err
		}
		row, err := model.Where(jsql.Eq("id", "s1")).One()
		if err != nil {
			return nil, err
		}
		if _, ok := row.Result["extra"]; ok {
			return row.Result, errors.New("unknown field was stored")
		}
		return row.Result, nil
	})
	s.run(2, "GetModel / RemoveModel / NewModel", func() (any, error) {
		if _, err := db.GetModel(s.schema, "f_users"); err != nil {
			return nil, err
		}
		tmp := db.NewModel(s.schema, "f_tmp", 1, "test")
		if _, err := db.GetModel(s.schema, tmp.Name); err != nil {
			return nil, err
		}
		if err := db.RemoveModel(s.schema, tmp.Name); err != nil {
			return nil, err
		}
		if _, err := db.GetModel(s.schema, tmp.Name); err == nil {
			return nil, errors.New("model still registered after RemoveModel")
		}
		return "ok", nil
	})
	s.run(2, "Insert (datos base)", func() (any, error) {
		m := s.models
		rows := []struct {
			model string
			data  et.Json
		}{
			{"f_doc_types", et.Json{"id": "CC", "title": "Cédula de ciudadanía"}},
			{"f_doc_types", et.Json{"id": "NIT", "title": "Número de identificación tributaria"}},
			{"f_users", et.Json{"id": "u1", "name": "Ana", "email": "ana@example.com", "age": 30, "password": "secret", "tp_doc": "CC"}},
			{"f_users", et.Json{"id": "u2", "name": "Luis", "email": "luis@example.com", "age": 17}},
			{"f_users", et.Json{"id": "u3", "name": "Marta O'Neil", "email": "marta@example.com", "age": 45, "tp_doc": "NIT"}},
			{"f_roles", et.Json{"id": "r1", "name": "admin"}},
			{"f_roles", et.Json{"id": "r2", "name": "editor"}},
			{"f_orders", et.Json{"id": "o1", "user_id": "u1", "amount": 100.5}},
			{"f_orders", et.Json{"id": "o2", "user_id": "u1", "amount": 200}},
			{"f_orders", et.Json{"id": "o3", "user_id": "u3", "amount": 50}},
			{"f_orders_items", et.Json{"id": "i1", "order_id": "o1", "product": "router"}},
			{"f_orders_items", et.Json{"id": "i2", "order_id": "o1", "product": "cable"}},
			{"f_products", et.Json{"id": "p1", "name": "Plan 200", "category": "internet", "price": 90000}},
			{"f_products", et.Json{"id": "p2", "name": "Plan 500", "category": "internet", "price": 150000}},
			{"f_products", et.Json{"id": "p3", "name": "Decoder", "category": "tv", "price": 20000}},
		}
		for _, row := range rows {
			if _, err := m[row.model].Insert(row.data).Exec(); err != nil {
				return nil, fmt.Errorf("%s: %w", row.model, err)
			}
		}
		bridge, ok := m["f_users"].Bridge("roles")
		if !ok {
			return nil, errors.New("Bridge(roles) not found")
		}
		for _, link := range []et.Json{{"user_id": "u1", "role_id": "r1"}, {"user_id": "u1", "role_id": "r2"}, {"user_id": "u2", "role_id": "r2"}} {
			if _, err := bridge.Insert(link).Exec(); err != nil {
				return nil, fmt.Errorf("bridge: %w", err)
			}
		}
		return len(rows) + 3, nil
	})
}

/**
* relations: Section 3 — details, masters, rollups and calculated fields.
**/
func (s *suite) relations() {
	users, orders := s.models["f_users"], s.models["f_orders"]
	s.run(3, "Detail en el select (DefineDetail)", func() (any, error) {
		row, err := orders.Select("id", "items").Where(jsql.Eq("id", "o1")).One()
		if err != nil {
			return nil, err
		}
		items := row.Result.ArrayJson("items")
		if len(items) != 2 {
			return row.Result, fmt.Errorf("want 2 items, got %d", len(items))
		}
		products := []string{}
		for _, item := range items {
			if len(item) != 1 {
				return row.Result, fmt.Errorf("detail shows only the defined fields (product), got %v", item)
			}
			products = append(products, item.Str("product"))
		}
		sort.Strings(products)
		return row.Result, expect("products", []string{"cable", "router"}, products)
	})
	s.run(3, "Model.Detail + Query.Detail", func() (any, error) {
		if _, ok := orders.Detail("items"); !ok {
			return nil, errors.New("Detail(items) not found")
		}
		row, err := orders.Where(jsql.Eq("id", "o1")).Detail("items").One()
		if err != nil {
			return nil, err
		}
		if n := len(row.Result.ArrayJson("items")); n != 2 {
			return row.Result, fmt.Errorf("want 2 items, got %d", n)
		}
		return row.Result, nil
	})
	s.run(3, "Master en el select (DefineMaster)", func() (any, error) {
		row, err := users.Select("id", "roles").Where(jsql.Eq("id", "u1")).One()
		if err != nil {
			return nil, err
		}
		names := []string{}
		for _, role := range row.Result.ArrayJson("roles") {
			if len(role) != 1 {
				return row.Result, fmt.Errorf("master shows only the defined fields (name), got %v", role)
			}
			names = append(names, role.Str("name"))
		}
		sort.Strings(names)
		return row.Result, expect("roles", []string{"admin", "editor"}, names)
	})
	s.run(3, "Master 1 a 1 (rows = 1)", func() (any, error) {
		row, err := users.Select("id", "main_role").Where(jsql.Eq("id", "u2")).One()
		if err != nil {
			return nil, err
		}
		return row.Result, expect("main_role", et.Json{"name": "editor"}, row.Result["main_role"])
	})
	s.run(3, "Model.Master + Model.Bridge", func() (any, error) {
		query, ok := users.Master("roles")
		if !ok {
			return nil, errors.New("Master(roles) not found")
		}
		items, err := query.Where(jsql.Eq("B.user_id", "u2")).All()
		if err != nil {
			return nil, err
		}
		return items.Result, expect("roles of u2", []string{"r2"}, ids(items))
	})
	s.run(3, "DefineRollup row (tp_doc → title)", func() (any, error) {
		items, err := users.Select("id", "tp_doc_title").OrderBy("id", true).All()
		if err != nil {
			return nil, err
		}
		got := []any{}
		for _, row := range items.Result {
			got = append(got, row["tp_doc_title"])
		}
		return items.Result, expect("titles", []any{"Cédula de ciudadanía", nil, "Número de identificación tributaria"}, got)
	})
	s.run(3, "DefineRollup (count, sum, object)", func() (any, error) {
		row, err := users.Select("id", "orders_count", "orders_total", "last_order").Where(jsql.Eq("id", "u3")).One()
		if err != nil {
			return nil, err
		}
		r := row.Result
		got := et.Json{"orders_count": r["orders_count"], "orders_total": r["orders_total"], "last_order": r.Json("last_order").Str("id")}
		return r, expect("rollups", et.Json{"orders_count": 1, "orders_total": 50, "last_order": "o3"}, got)
	})
	s.run(3, "DefineCalcFunc + Model.Calc", func() (any, error) {
		row, err := users.Calc("label").Where(jsql.Eq("id", "u1")).One()
		if err != nil {
			return nil, err
		}
		return row.Result, expect("label", "Ana <ana@example.com>", row.Result["label"])
	})
	s.run(3, "DefineCalc (script JS)", func() (any, error) {
		row, err := users.Select("id", "name", "initial").Where(jsql.Eq("id", "u3")).One()
		if err != nil {
			return nil, err
		}
		return row.Result, expect("initial", "M", row.Result["initial"])
	})
}

/**
* queries: Section 4 — queries.
**/
func (s *suite) queries() {
	users, orders := s.models["f_users"], s.models["f_orders"]
	s.run(4, "Select + Where + OrderBy + All", func() (any, error) {
		items, err := users.Select("id", "name", "age").Where(jsql.More("age", 18)).OrderBy("age", false).All()
		if err != nil {
			return nil, err
		}
		return items.Result, expect("ids", []string{"u3", "u1"}, ids(items))
	})
	s.run(4, "One / First / Count / Exists", func() (any, error) {
		one, err := users.Where(jsql.Eq("id", "u2")).One()
		if err != nil {
			return nil, err
		}
		first, err := users.As("A").OrderBy("id", true).First(2)
		if err != nil {
			return nil, err
		}
		count, err := users.Count()
		if err != nil {
			return nil, err
		}
		exists, err := users.Where(jsql.Eq("id", "u9")).Exists()
		if err != nil {
			return nil, err
		}
		got := et.Json{"one": one.Result.Str("name"), "first": ids(first), "count": count, "exists": exists}
		return got, expect("results", et.Json{"one": "Luis", "first": []string{"u1", "u2"}, "count": 3, "exists": false}, got)
	})
	s.run(4, "Limit (paginación) / Page", func() (any, error) {
		page2, err := users.As("A").OrderBy("id", true).Limit(2, 1)
		if err != nil {
			return nil, err
		}
		return page2.Result, expect("page 2", []string{"u2"}, ids(page2))
	})
	s.run(4, "Hidden (campo oculto en la consulta y en el modelo)", func() (any, error) {
		row, err := users.Where(jsql.Eq("id", "u1")).Hidden("email").One()
		if err != nil {
			return nil, err
		}
		if _, ok := row.Result["email"]; ok {
			return row.Result, errors.New("email should be hidden")
		}
		if _, ok := row.Result["password"]; ok {
			return row.Result, errors.New("password (model hidden) should be hidden")
		}
		return row.Result, nil
	})
	s.run(4, "Join (fluido)", func() (any, error) {
		items, err := orders.As("A").
			Join(users, "U", []*et.Condition{jsql.Eq("A.user_id", "U.id")}).
			Select("A.id", "U.name:user").
			OrderBy("A.id", true).
			All()
		if err != nil {
			return nil, err
		}
		return items.Result, expect("rows", []et.Json{{"id": "o1", "user": "Ana"}, {"id": "o2", "user": "Ana"}, {"id": "o3", "user": "Marta O'Neil"}}, items.Result)
	})
	s.run(4, "LeftJoin + GroupBy", func() (any, error) {
		items, err := users.As("A").
			LeftJoin(orders, "O", []*et.Condition{jsql.Eq("O.user_id", "A.id")}).
			Select("A.id", "count(O.id):n").
			GroupBy("A.id").
			OrderBy("A.id", true).
			All()
		if err != nil {
			return nil, err
		}
		return items.Result, expect("rows", []et.Json{{"id": "u1", "n": 2}, {"id": "u2", "n": 0}, {"id": "u3", "n": 1}}, items.Result)
	})
	s.run(4, "GroupBy + Having (fluido)", func() (any, error) {
		items, err := orders.Select("user_id", "count(id):n", "sum(amount):total").
			GroupBy("user_id").
			Having(jsql.More("count(id)", 1)).
			All()
		if err != nil {
			return nil, err
		}
		return items.Result, expect("rows", []et.Json{{"user_id": "u1", "n": 2, "total": 300.5}}, items.Result)
	})
	s.run(4, "Model.Query (JSON)", func() (any, error) {
		items, err := users.Query(et.Json{
			"selects": []any{"id", "name"},
			"where":   []any{et.Json{"age": et.Json{"more_eq": 18}}},
			"orders":  []any{et.Json{"name": false}},
			"limit":   10,
		}).All()
		if err != nil {
			return nil, err
		}
		return items.Result, expect("ids", []string{"u3", "u1"}, ids(items))
	})
	s.run(4, "Model.Query con from (otro modelo y alias)", func() (any, error) {
		items, err := orders.Query(et.Json{
			"from":    s.schema + ".f_users:U",
			"selects": []any{"U.id", "U.name"},
			"where":   []any{et.Json{"U.age": et.Json{"more": 18}}},
			"orders":  []any{et.Json{"U.id": true}},
		}).All()
		if err != nil {
			return nil, err
		}
		return items.Result, expect("ids", []string{"u1", "u3"}, ids(items))
	})
	s.run(4, "Model.Query con from sin esquema", func() (any, error) {
		items, err := users.Query(et.Json{
			"from":    "f_roles:R",
			"selects": []any{"R.name"},
			"orders":  []any{et.Json{"R.name": true}},
		}).All()
		if err != nil {
			return nil, err
		}
		return items.Result, expect("roles", []et.Json{{"name": "admin"}, {"name": "editor"}}, items.Result)
	})
	s.run(4, "Model.Query con from + join + groups", func() (any, error) {
		items, err := orders.Query(et.Json{
			"from":    "f_users:U",
			"join":    []any{et.Json{"to": s.schema + ".f_orders:O", "on": []any{et.Json{"O.user_id": et.Json{"eq": "U.id"}}}}},
			"selects": []any{"U.name", "count(O.id):n"},
			"groups":  []any{"U.name"},
			"orders":  []any{et.Json{"U.name": true}},
		}).All()
		if err != nil {
			return nil, err
		}
		return items.Result, expect("rows", []et.Json{{"name": "Ana", "n": 2}, {"name": "Marta O'Neil", "n": 1}}, items.Result)
	})
	s.run(4, "Model.Query con from inválido (devuelve error)", func() (any, error) {
		_, err := users.Query(et.Json{"from": "no_existe:X"}).All()
		if err == nil {
			return nil, errors.New("an unknown from was accepted")
		}
		return err.Error(), nil
	})
	s.run(4, "NewQuery / GetField / GetColumn / GetFrom / ToJson", func() (any, error) {
		query := jsql.NewQuery(users, "X")
		fld, ok := query.GetField("X.name:nombre")
		if !ok {
			return nil, errors.New("GetField failed")
		}
		col, ok := users.GetColumn("age")
		if !ok {
			return nil, errors.New("GetColumn failed")
		}
		got := et.Json{"field": fld.As, "column": string(col.TypeColumn), "from": users.GetFrom().Name, "json": len(query.ToJson()) > 0}
		return got, expect("values", et.Json{"field": "nombre", "column": "atrib", "from": "f_users", "json": true}, got)
	})
	s.run(4, "Test (genera el SQL sin ejecutarlo) / Debug", func() (any, error) {
		items, err := users.Where(jsql.Eq("id", "u1")).Test().All()
		if err != nil {
			return nil, err
		}
		if items.Count != 0 {
			return nil, errors.New("Test() executed the query")
		}
		return "sql generated, not executed", nil
	})
}

/**
* conditions: Section 5 — every condition constructor against the users model.
**/
func (s *suite) conditions() {
	users := s.models["f_users"]
	cases := []struct {
		name string
		cond *et.Condition
		want []string
	}{
		{"Eq", jsql.Eq("name", "Ana"), []string{"u1"}},
		{"Neg", jsql.Neg("name", "Ana"), []string{"u2", "u3"}},
		{"Less", jsql.Less("age", 30), []string{"u2"}},
		{"LessEq", jsql.LessEq("age", 30), []string{"u1", "u2"}},
		{"More", jsql.More("age", 30), []string{"u3"}},
		{"MoreEq", jsql.MoreEq("age", 30), []string{"u1", "u3"}},
		{"Like", jsql.Like("name", "%an%"), []string{"u1"}},
		{"In", jsql.In("id", []any{"u1", "u3"}), []string{"u1", "u3"}},
		{"NotIn", jsql.NotIn("id", []any{"u1", "u3"}), []string{"u2"}},
		{"Is (NULL)", jsql.Is("nickname", nil), []string{"u1", "u2", "u3"}},
		{"IsNot (NULL)", jsql.IsNot("age", nil), []string{"u1", "u2", "u3"}},
		{"Null", jsql.Null("nickname"), []string{"u1", "u2", "u3"}},
		{"NotNull", jsql.NotNull("email"), []string{"u1", "u2", "u3"}},
		{"Between", jsql.Between("age", 18, 40), []string{"u1"}},
		{"NotBetween", jsql.NotBetween("age", 18, 40), []string{"u2", "u3"}},
		{"Where (genérico)", jsql.Where("name", "Luis"), []string{"u2"}},
	}
	for _, c := range cases {
		c := c
		s.run(5, c.name, func() (any, error) {
			items, err := users.Where(c.cond).OrderBy("id", true).All()
			if err != nil {
				return nil, err
			}
			return ids(items), expect(c.name, c.want, ids(items))
		})
	}
	s.run(5, "And / Or (conectores)", func() (any, error) {
		items, err := users.Where(jsql.Eq("name", "Ana")).Or(jsql.Eq("name", "Luis")).OrderBy("id", true).All()
		if err != nil {
			return nil, err
		}
		and, err := users.Where(jsql.More("age", 18)).And(jsql.Like("name", "%neil%")).All()
		if err != nil {
			return nil, err
		}
		got := et.Json{"or": ids(items), "and": ids(and)}
		return got, expect("ids", et.Json{"or": []string{"u1", "u2"}, "and": []string{"u3"}}, got)
	})
}

/**
* commands: Section 6 — commands, on the products model.
**/
func (s *suite) commands() {
	products := s.models["f_products"]
	s.run(6, "Insert (RETURNING)", func() (any, error) {
		row, err := products.Insert(et.Json{"id": "p4", "name": "Router Wi-Fi 6", "category": "equipos", "price": 350000}).One()
		if err != nil {
			return nil, err
		}
		return row.Result, expect("price", 350000, row.Result["price"])
	})
	s.run(6, "Bulk", func() (any, error) {
		items, err := products.Bulk([]et.Json{
			{"id": "p5", "name": "Cable HDMI", "category": "tv", "price": 15000},
			{"id": "p6", "name": "Control", "category": "tv", "price": 10000},
		}).Exec()
		if err != nil {
			return nil, err
		}
		return items.Result, expect("ids", []string{"p5", "p6"}, ids(items))
	})
	s.run(6, "Update + Where", func() (any, error) {
		row, err := products.Update(et.Json{"price": 99000, "promo": true}).Where(jsql.Eq("id", "p1")).One()
		if err != nil {
			return nil, err
		}
		return row.Result, expect("updated", et.Json{"price": 99000, "promo": true}, et.Json{"price": row.Result["price"], "promo": row.Result["promo"]})
	})
	s.run(6, "Update + Where + Or (varias filas)", func() (any, error) {
		items, err := products.Update(et.Json{"stock": 5}).Where(jsql.Eq("id", "p5")).Or(jsql.Eq("id", "p6")).Exec()
		if err != nil {
			return nil, err
		}
		sort.Strings(ids(items))
		return items.Count, expect("count", 2, items.Count)
	})
	s.run(6, "Upsert (inserta y luego actualiza)", func() (any, error) {
		first, err := products.Upsert(et.Json{"id": "p7", "name": "Antena", "category": "tv"}).Where(jsql.Eq("id", "p7")).One()
		if err != nil {
			return nil, err
		}
		second, err := products.Upsert(et.Json{"id": "p7", "name": "Antena HD"}).Where(jsql.Eq("id", "p7")).One()
		if err != nil {
			return nil, err
		}
		got := et.Json{"insert": first.Result.Str("name"), "update": second.Result.Str("name")}
		return got, expect("names", et.Json{"insert": "Antena", "update": "Antena HD"}, got)
	})
	s.run(6, "Return (campos del RETURNING)", func() (any, error) {
		row, err := products.Update(et.Json{"stock": 7}).Where(jsql.Eq("id", "p4")).Return("id").One()
		if err != nil {
			return nil, err
		}
		return row.Result, expect("id", "p4", row.Result["id"])
	})
	s.run(6, "Delete (devuelve la fila borrada)", func() (any, error) {
		row, err := products.Delete().Where(jsql.Eq("id", "p6")).One()
		if err != nil {
			return nil, err
		}
		exists, err := products.Where(jsql.Eq("id", "p6")).Exists()
		if err != nil {
			return nil, err
		}
		got := et.Json{"deleted": row.Result.Str("name"), "exists": exists}
		return got, expect("delete", et.Json{"deleted": "Control", "exists": false}, got)
	})
	s.run(6, "Test (no ejecuta) + ToJson", func() (any, error) {
		cmd := products.Insert(et.Json{"id": "p8", "name": "No se guarda"}).Test()
		if _, err := cmd.Exec(); err != nil {
			return nil, err
		}
		exists, err := products.Where(jsql.Eq("id", "p8")).Exists()
		if err != nil {
			return nil, err
		}
		got := et.Json{"exists": exists, "type": cmd.ToJson().Str("type")}
		return got, expect("test", et.Json{"exists": false, "type": "insert"}, got)
	})
}

/**
* triggers: Section 7 — Go and JS triggers on the events model.
**/
func (s *suite) triggers() {
	var events *jsql.Model
	log := func(name string) jsql.TriggerFunction {
		return func(tx *jsql.Tx, old, new et.Json) error {
			s.events = append(s.events, name)
			return nil
		}
	}
	s.run(7, "Before/After Insert, Update, Delete e InsertOrUpdate", func() (any, error) {
		var err error
		events, err = s.model("f_events", "name")
		if err != nil {
			return nil, err
		}
		events.BeforeInsert(func(tx *jsql.Tx, old, new et.Json) error {
			new["stage"] = "before_insert"
			return nil
		})
		events.AfterInsert(log("after_insert"))
		events.BeforeUpdate(func(tx *jsql.Tx, old, new et.Json) error {
			new["stage"] = "before_update"
			return nil
		})
		events.AfterUpdate(log("after_update"))
		events.BeforeDelete(log("before_delete"))
		events.AfterDelete(log("after_delete"))
		events.BeforeInsertOrUpdate(func(tx *jsql.Tx, old, new et.Json) error {
			new["touched"] = true
			return nil
		})
		events.AfterInsertOrUpdate(log("after_insert_or_update"))
		if err := events.Init(); err != nil {
			return nil, err
		}
		inserted, err := events.Insert(et.Json{"id": "e1", "name": "alta"}).One()
		if err != nil {
			return nil, err
		}
		updated, err := events.Update(et.Json{"name": "cambio"}).Where(jsql.Eq("id", "e1")).One()
		if err != nil {
			return nil, err
		}
		if _, err := events.Delete().Where(jsql.Eq("id", "e1")).Exec(); err != nil {
			return nil, err
		}
		got := et.Json{
			"insert": et.Json{"stage": inserted.Result["stage"], "touched": inserted.Result["touched"]},
			"update": et.Json{"stage": updated.Result["stage"], "touched": updated.Result["touched"]},
			"events": s.events,
		}
		return got, expect("triggers", et.Json{
			"insert": et.Json{"stage": "before_insert", "touched": true},
			"update": et.Json{"stage": "before_update", "touched": true},
			"events": []string{"after_insert", "after_insert_or_update", "after_update", "after_insert_or_update", "before_delete", "after_delete"},
		}, got)
	})
	s.run(7, "Trigger del comando + error que aborta", func() (any, error) {
		row, err := events.Insert(et.Json{"id": "e2", "name": "cmd"}).
			BeforeInsert(func(tx *jsql.Tx, old, new et.Json) error {
				new["source"] = "command"
				return nil
			}).One()
		if err != nil {
			return nil, err
		}
		_, err = events.Insert(et.Json{"id": "e3", "name": "rechazado"}).
			BeforeInsert(func(tx *jsql.Tx, old, new et.Json) error {
				return errors.New("rejected by trigger")
			}).Exec()
		if err == nil {
			return nil, errors.New("the failing trigger did not abort the insert")
		}
		exists, err := events.Where(jsql.Eq("id", "e3")).Exists()
		if err != nil {
			return nil, err
		}
		got := et.Json{"source": row.Result["source"], "aborted_exists": exists}
		return got, expect("command trigger", et.Json{"source": "command", "aborted_exists": false}, got)
	})
	s.run(7, "Triggers JS (DefineBeforeInsert / DefineAfterUpdate…)", func() (any, error) {
		events.DefineBeforeInsert(`NEW.js = "before_insert";`)
		events.DefineBeforeUpdate("js_update", `NEW.js = "before_update";`)
		row, err := events.Insert(et.Json{"id": "e4", "name": "js"}).One()
		if err != nil {
			return nil, err
		}
		updated, err := events.Update(et.Json{"name": "js2"}).Where(jsql.Eq("id", "e4")).One()
		if err != nil {
			return nil, err
		}
		got := et.Json{"insert": row.Result["js"], "update": updated.Result["js"]}
		return got, expect("js", et.Json{"insert": "before_insert", "update": "before_update"}, got)
	})
}

/**
* transactions: Section 8 — commit and rollback.
**/
func (s *suite) transactions() {
	products := s.models["f_products"]
	s.run(8, "NewTx + ExecTx + Rollback", func() (any, error) {
		tx := jsql.NewTx()
		if _, err := products.Insert(et.Json{"id": "t1", "name": "rollback"}).ExecTx(tx); err != nil {
			return nil, err
		}
		if err := tx.Rollback(); err != nil {
			return nil, err
		}
		exists, err := products.Where(jsql.Eq("id", "t1")).Exists()
		if err != nil {
			return nil, err
		}
		return exists, expect("exists after rollback", false, exists)
	})
	s.run(8, "NewTx + ExecTx + Commit", func() (any, error) {
		tx := jsql.NewTx()
		if _, err := products.Insert(et.Json{"id": "t2", "name": "commit"}).ExecTx(tx); err != nil {
			return nil, err
		}
		if _, err := products.Update(et.Json{"name": "commit 2"}).Where(jsql.Eq("id", "t2")).ExecTx(tx); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		row, err := products.Where(jsql.Eq("id", "t2")).One()
		if err != nil {
			return nil, err
		}
		return row.Result, expect("name after commit", "commit 2", row.Result["name"])
	})
}

/**
* series: Section 9 — series (consecutive codes).
**/
func (s *suite) series() {
	var series *jsql.Series
	s.run(9, "DefineSeries", func() (any, error) {
		var err error
		series, err = jsql.DefineSeries(s.db, s.schema)
		return "series", err
	})
	s.run(9, "SetSeries + GetSeries", func() (any, error) {
		if err := series.SetSeries("invoice", "FAC-%05d", 10); err != nil {
			return nil, err
		}
		item, err := series.GetSeries("invoice")
		if err != nil {
			return nil, err
		}
		got := et.Json{"format": item.Result.Str("format"), "value": item.Result["value"]}
		return got, expect("series", et.Json{"format": "FAC-%05d", "value": 10}, got)
	})
	s.run(9, "GenValue + GenSerie", func() (any, error) {
		value, err := series.GenValue("invoice")
		if err != nil {
			return nil, err
		}
		code, err := series.GenSerie("invoice")
		if err != nil {
			return nil, err
		}
		got := et.Json{"value": value, "code": code}
		return got, expect("next", et.Json{"value": 11, "code": "FAC-00012"}, got)
	})
	s.run(9, "DeleteSeries", func() (any, error) {
		if err := series.DeleteSeries("invoice"); err != nil {
			return nil, err
		}
		item, err := series.GetSeries("invoice")
		if err != nil {
			return nil, err
		}
		return item.Ok, expect("exists", false, item.Ok)
	})
}

/**
* audit: Section 10 — audit log.
**/
func (s *suite) audit() {
	s.run(10, "OnAuditLog + Model.AddAuditLog + AddAuditLog", func() (any, error) {
		calls := []string{}
		s.db.OnAuditLog(func(userId, action string) error {
			calls = append(calls, userId+":"+action)
			return nil
		})
		s.models["f_users"].AddAuditLog("tester", "review")
		list := jsql.AddAuditLog(nil, "tester", "export")
		got := et.Json{"callback": calls, "list": len(list), "model": len(s.models["f_users"].ToJson().Array("audit_log")) > 0}
		return got, expect("audit", et.Json{"callback": []string{"tester:review"}, "list": 1, "model": true}, got)
	})
}

/**
* utilities: Section 11 — helpers and accessors.
**/
func (s *suite) utilities() {
	users := s.models["f_users"]
	s.run(11, "Quoted / EscapeSQLString / SQLParse / JsonString", func() (any, error) {
		js, err := jsql.JsonString(et.Json{"html": "<b>&"})
		if err != nil {
			return nil, err
		}
		got := et.Json{
			"quoted": jsql.Quoted("O'Brien"),
			"escape": jsql.EscapeSQLString("it's"),
			"parse":  jsql.SQLParse("name = $1 AND age = $2", "Ana", 30),
			"json":   js,
		}
		return got, expect("helpers", et.Json{
			"quoted": "'O''Brien'",
			"escape": "it''s",
			"parse":  "name = 'Ana' AND age = 30",
			"json":   `{"html":"<b>&"}`,
		}, got)
	})
	s.run(11, "ArgWhitAs / ArgWhitSchema / StatusList / TypeColumn.Str", func() (any, error) {
		as, _ := jsql.ArgWhitAs("public.users:U")
		schema, _ := jsql.ArgWhitSchema("public.users")
		got := et.Json{"as": as, "schema": schema, "status": len(jsql.StatusList()) > 0, "type": jsql.ATTRIB.Str()}
		return got, expect("helpers", et.Json{"as": []string{"public.users", "U"}, "schema": []string{"public", "users"}, "status": true, "type": "atrib"}, got)
	})
	s.run(11, "RowsToItems", func() (any, error) {
		rows, err := users.SqlDB().Query(rawSelectOne(s.tg.name))
		if err != nil {
			return nil, err
		}
		items := jsql.RowsToItems(rows)
		return items.Result, expect("n", "1", fmt.Sprint(items.Result[0]["n"]))
	})
	s.run(11, "Model.Db / SetDb / GetModel / ToJson, Column.ToJson, Schema.ToJson", func() (any, error) {
		if users.Db() != s.db {
			return nil, errors.New("Db() is not the connected DB")
		}
		users.SetDb(s.db)
		other, err := users.GetModel(s.schema, "f_orders")
		if err != nil {
			return nil, err
		}
		col, _ := users.GetColumn("name")
		schemas := 0
		for _, sch := range s.db.Schemas {
			if len(sch.ToJson()) > 0 {
				schemas++
			}
		}
		got := et.Json{"model": users.ToJson().Str("name"), "other": other.Name, "column": col.ToJson().Str("name"), "schemas": schemas > 0}
		return got, expect("accessors", et.Json{"model": "f_users", "other": "f_orders", "column": "name", "schemas": true}, got)
	})
}

/**
* writeResults: Writes result/<driver>.json (every case) and result/<driver>.sql (the SQL of each case).
* @param t *testing.T, driver string, results []caseResult
**/
func writeResults(t *testing.T, driver string, results []caseResult) {
	t.Helper()
	if err := os.MkdirAll("result", 0o755); err != nil {
		t.Fatal(err)
	}
	bt, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("result", driver+".json"), bt, 0o644); err != nil {
		t.Fatal(err)
	}

	var sb strings.Builder
	for _, r := range results {
		if len(r.SQL) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("-- ========== %s · %s [%s]\n\n", r.Section, r.Case, r.Status))
		for _, sql := range r.SQL {
			sb.WriteString(sql)
			sb.WriteString("\n\n")
		}
	}
	if err := os.WriteFile(filepath.Join("result", driver+".sql"), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

/**
* writeSummary: Writes result/summary.md with the status of every case for each driver.
* @param t *testing.T, all map[string][]caseResult
**/
func writeSummary(t *testing.T, all map[string][]caseResult) {
	t.Helper()
	drivers := []string{"sqlite", "postgres", "oracle"}
	status := map[string]map[string]string{}
	order := []string{}
	sectionOf := map[string]string{}
	totals := map[string]map[string]int{}
	for _, d := range drivers {
		totals[d] = map[string]int{}
		for _, r := range all[d] {
			key := r.Section + " · " + r.Case
			if _, ok := status[key]; !ok {
				status[key] = map[string]string{}
				order = append(order, key)
				sectionOf[key] = r.Section
			}
			status[key][d] = r.Status
			totals[d][r.Status]++
		}
	}
	sort.SliceStable(order, func(i, j int) bool {
		return sectionIndex(sectionOf[order[i]]) < sectionIndex(sectionOf[order[j]])
	})
	icon := map[string]string{"pass": "✅", "fail": "❌", "gap": "⚠️ brecha", "skip": "➖ omitido", "": "—"}

	var sb strings.Builder
	sb.WriteString("# Resultado de TestCatalog\n\n")
	sb.WriteString("Generado por `go test -run TestCatalog` en `jsql/drivers/test`. El detalle de cada caso (salida y SQL) está en `<driver>.json` y `<driver>.sql`.\n\n")
	sb.WriteString("| Driver | Pasan | Fallan | Brechas | Omitidos |\n|---|---|---|---|---|\n")
	for _, d := range drivers {
		if len(all[d]) == 0 {
			sb.WriteString(fmt.Sprintf("| %s | no disponible | | | |\n", d))
			continue
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d |\n", d, totals[d]["pass"], totals[d]["fail"], totals[d]["gap"], totals[d]["skip"]))
	}
	current := ""
	for _, key := range order {
		if sectionOf[key] != current {
			current = sectionOf[key]
			sb.WriteString(fmt.Sprintf("\n## %s\n\n| Caso | sqlite | postgres | oracle |\n|---|---|---|---|\n", current))
		}
		name := strings.TrimPrefix(key, current+" · ")
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", name, icon[status[key]["sqlite"]], icon[status[key]["postgres"]], icon[status[key]["oracle"]]))
	}
	if err := os.WriteFile(filepath.Join("result", "summary.md"), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

/**
* sectionIndex: Position of a section name in sections.
* @param name string
* @return int
**/
func sectionIndex(name string) int {
	for i, s := range sections {
		if s == name {
			return i
		}
	}
	return len(sections)
}
