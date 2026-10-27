/**
 * jsql — bindings available to JavaScript (goja) inside jsql scripts: triggers (DefineBeforeInsert…,
 * before_insert… in a command descriptor) and calc scripts (DefineCalc). Generated from jsql/wrapper.go.
 *
 * Monaco:
 *   monaco.languages.typescript.javascriptDefaults.addExtraLib(source, "file:///jsql.d.ts");
 *
 * A Go error is thrown as a JavaScript exception; wrap calls in try/catch to handle it.
 */

/** A record: column name → value. */
type JsqlRecord = { [key: string]: any };

/** Result of a query or command that returns many records. */
interface JsqlItems {
  ok: boolean;
  count: number;
  result: JsqlRecord[];
}

/** Result of a query or command that returns one record (`ok` is false when there was none). */
interface JsqlItem {
  ok: boolean;
  result: JsqlRecord;
}

/** Data type of a column. */
type JsqlTypeData =
  | "any"
  | "byte"
  | "key"
  | "text"
  | "memo"
  | "int"
  | "float"
  | "bool"
  | "datetime"
  | "json"
  | "array"
  | "array_json"
  | "array_string"
  | "array_int"
  | "array_float"
  | "array_bool"
  | "array_datetime";

/** Kind of a column. */
type JsqlTypeColumn =
  | "column"
  | "atrib"
  | "detail"
  | "master"
  | "rollup"
  | "calc_func"
  | "calc";

/** Comparison operator of a condition. */
type JsqlOperator =
  | "eq"
  | "neg"
  | "less"
  | "less_eq"
  | "more"
  | "more_eq"
  | "like"
  | "in"
  | "not_in"
  | "is"
  | "is_not"
  | "null"
  | "not_null"
  | "between"
  | "not_between";

/** Aggregate of a rollup. */
type JsqlRollupOperation =
  | "count"
  | "sum"
  | "avg"
  | "min"
  | "max"
  | "row"
  | "object";

/** A where condition, built with the jsql.* functions (eq, more, like…). Opaque. */
interface JsqlCondition {
  readonly __condition: unique symbol;
}

/** A where condition as JSON: {field: {operator: value}}; with where/and/or in front, its connector. */
type JsqlWhere = { [field: string]: { [op in JsqlOperator]?: any } };
type JsqlWhereItem =
  | JsqlWhere
  | { where: JsqlWhere }
  | { and: JsqlWhere }
  | { or: JsqlWhere };

/** Map of local field → foreign field. */
type JsqlKeys = { [field: string]: string };

interface JsqlIndex {
  name: string;
  sorted: boolean;
}

interface JsqlColumn {
  name: string;
  type_column: JsqlTypeColumn;
  type_data: JsqlTypeData;
  default: any;
}

/** Reference of a model in a relation or a query. */
interface JsqlFromRef {
  database: string;
  schema: string;
  name: string;
  table: string;
  as: string;
}

/** A model in a relation or a query. */
interface JsqlFrom {
  ref(): JsqlFromRef;
  /** The model, or null. */
  model(): JsqlModel | null;
}

/** Foreign key relation of a model (defineForeignKeys, defineDetail). */
interface JsqlDetail {
  ref(): { to: JsqlFromRef | null };
  /** to, keys, select, on_delete_cascade, on_update_cascade, rows. */
  toJson(): JsqlRecord;
  to(): JsqlFrom | null;
  /** Query of the detail rows of item (a record of the model). */
  getQuery(item: JsqlRecord, page: number, rows: number): JsqlQuery;
}

/** Relation through a bridge model (defineMaster). */
interface JsqlMaster {
  ref(): { to: JsqlFromRef | null; bridge: JsqlFromRef | null };
  /** from, to, bridge, keys, to_keys, select, rows. */
  toJson(): JsqlRecord;
  from(): JsqlFrom | null;
  to(): JsqlFrom | null;
  bridge(): JsqlFrom | null;
}

/** Aggregate over a related model (defineRollup). */
interface JsqlRollups {
  /** to, keys, select, operation, where. */
  toJson(): JsqlRecord;
  to(): JsqlFrom | null;
}

/** Detail or master resolved for a query (query.detail(...), query.master(...)). */
interface JsqlQueryDetail {
  /** to, bridge, to_keys, keys, select, page, rows. */
  toJson(): JsqlRecord;
  to(): JsqlFrom | null;
  bridge(): JsqlFrom | null;
  /** Query of the related rows of item (a row of the query). */
  getQuery(item: JsqlRecord): JsqlQuery;
}

/** Rollup resolved for a query (when the query runs). */
interface JsqlQueryRollups {
  /** to, keys, select, operation, where. */
  toJson(): JsqlRecord;
  to(): JsqlFrom | null;
  /** Query of the rollup for item, or null when item lacks any of the keys or of the fields its where references. */
  getQuery(item: JsqlRecord): JsqlQuery | null;
}

/** Trigger function: change `newRecord` to change what is saved; throw to abort the command. */
type JsqlTriggerFunction = (
  tx: any,
  oldRecord: JsqlRecord,
  newRecord: JsqlRecord,
) => void;

/** Calc function: set the calculated fields on `data`. */
type JsqlCalcFunction = (tx: any, data: JsqlRecord) => void;

/** A transaction. Pass it to the *Tx methods; without one each call runs on its own. */
interface JsqlTx {
  commit(): void;
  rollback(): void;
}

/** Query descriptor of db.query / model.query (see jsql/feature.md). */
interface JsqlQueryDescriptor {
  [key: string]: any;
}

/** Connection of a driver: the keys of its GetParams. Missing keys are read from DB_*. */
interface JsqlConnection {
  /** postgres, mysql, mssql */
  database?: string;
  host?: string;
  port?: number;
  /** postgres, mysql, mssql */
  user?: string;
  password?: string;
  /** postgres */
  sslmode?: string;
  /** postgres, sqlite */
  app_name?: string;
  /** postgres, sqlite */
  record_limit?: number;
  /** sqlite: path of the database file */
  file?: string;
  /** sqlite */
  pool_max_open?: number;
  pool_max_idle?: number;
  pool_lifetime?: number;
  pool_idle_time?: number;
  /** oracle, mysql, mssql */
  id?: string;
  /** oracle */
  username?: string;
  service_name?: string;
  ssl?: boolean;
  ssl_verify?: boolean;
  /** milliseconds */
  timeout?: number;
}

/** Connection object of a driver (jsql.newConnection). */
interface JsqlConnectionObject {
  /** The connection params (timeout in milliseconds). */
  getParams(): JsqlConnection & { driver: string };
  setDatabase(name: string): void;
  /** database (postgres, mysql, mssql), file (sqlite) or service_name (oracle). */
  getDatabase(): string;
  /** The id of the connection (every driver has one). */
  id(): string;
}

/** Params of jsql.newDB / connectTo / createDB. Missing values are read from DB_* (as in loadTo). */
interface JsqlConnectParams {
  driver?: "postgres" | "sqlite" | "mysql" | "mssql" | "oracle" | (string & {});
  host?: string;
  /** Database to connect to, unless connection names it (database, file or service_name). */
  name?: string;
  /** Its JSON, or a connection object from jsql.newConnection. */
  connection?: JsqlConnection | JsqlConnectionObject;
  record_limit?: number;
  /** milliseconds; 0 or none never fails by timeout */
  timeout?: number;
  is_debug?: boolean;
}

/** Define descriptor of db.define: the Define structure in JSON. */
interface JsqlDefine {
  schema: string;
  id?: string;
  name: string;
  version?: number;
  source_field?: string;
  idx_field?: string;
  primary_keys?: { name: string; sorted?: boolean }[];
  foreign_keys?: {
    to: { schema: string; name: string };
    keys: JsqlKeys;
    on_delete_cascade?: boolean;
    on_update_cascade?: boolean;
  }[];
  indexes?: { name: string; sorted?: boolean }[];
  unique?: { name: string; sorted?: boolean }[];
  required?: { name: string; sorted?: boolean }[];
  columns?: {
    name: string;
    type_column?: JsqlTypeColumn;
    type_data: JsqlTypeData;
    default?: any;
  }[];
  hiddens?: string[];
  omit_updates?: string[];
  details?: {
    name: string;
    keys: JsqlKeys;
    rows?: number;
    select?: string[];
    columns?: {
      name: string;
      type_column?: JsqlTypeColumn;
      type_data: JsqlTypeData;
      default?: any;
    }[];
    primary_keys?: { name: string; sorted?: boolean }[];
    indexes?: { name: string; sorted?: boolean }[];
    idx_field?: string;
  }[];
  master?: {
    name: string;
    to: { schema: string; name: string };
    keys: JsqlKeys;
    to_keys: JsqlKeys;
    select?: string[];
    rows?: number;
  }[];
  rollups?: {
    name: string;
    to: { schema: string; name: string };
    keys: JsqlKeys;
    select?: string[];
    operation: JsqlRollupOperation;
    /** Filter on `to`, joined to the keys with AND (no `or`); `{field: "x"}` reads x from the row. */
    where?: JsqlWhereItem[];
  }[];
  user_id?: string;
}

interface JsqlDB {
  toJson(): JsqlRecord;
  init(): void;
  close(): void;
  setDebug(debug: boolean): void;
  debug(): void;
  /** Runs raw SQL; `$1`, `$2`… are replaced by args. */
  sql(query: string, ...args: any[]): JsqlItems;
  sqlTx(tx: JsqlTx | null, query: string, ...args: any[]): JsqlItems;
  /** Runs a JSON descriptor (query, command or define). */
  query(query: JsqlQueryDescriptor): JsqlItems;
  queryTx(tx: JsqlTx | null, query: JsqlQueryDescriptor): JsqlItems;
  newModel(
    schema: string,
    name: string,
    version: number,
    userId: string,
  ): JsqlModel;
  /** Drops the model's table and removes the model. */
  removeModel(schema: string, name: string): void;
  getModel(schema: string, name: string): JsqlModel;
  /** The series of a schema of this database (defined once, then reused). In a trigger, pass TX to genSerie/genValue:
   * NEW.code = db.series("app").genSerie("users", TX) */
  series(schema: string): JsqlSeries;
  define(define: JsqlDefine): JsqlModel;
  defineModel(
    schema: string,
    name: string,
    version: number,
    userId: string,
  ): JsqlModel;
  defineTenantModel(
    schema: string,
    name: string,
    version: number,
    userId: string,
  ): JsqlModel;
  defineProjectModel(
    schema: string,
    name: string,
    version: number,
    userId: string,
  ): JsqlModel;
  onAuditLog(fn: (userId: string, action: string) => void): void;
}

interface JsqlModel {
  toJson(): JsqlRecord;
  debug(): JsqlModel;
  init(): void;
  stricted(): JsqlModel;
  db(): JsqlDB;
  setDb(db: JsqlDB): JsqlModel;
  getModel(schema: string, name: string): JsqlModel;
  addAuditLog(userId: string, action: string): void;

  // Definition
  defineSource(): JsqlColumn | null;
  defineIdxField(): JsqlIndex | null;
  defineIndex(
    name: string,
    type: JsqlTypeData,
    defaultValue: any,
  ): JsqlIndex | null;
  definePrimaryKey(
    name: string,
    type: JsqlTypeData,
    defaultValue: any,
  ): JsqlIndex | null;
  defineUnique(
    name: string,
    type: JsqlTypeData,
    defaultValue: any,
  ): JsqlIndex | null;
  defineRequired(
    name: string,
    type: JsqlTypeData,
    defaultValue: any,
  ): JsqlIndex | null;
  defineColumn(
    name: string,
    type: JsqlTypeData,
    defaultValue: any,
  ): JsqlColumn | null;
  defineAttrib(
    name: string,
    type: JsqlTypeData,
    defaultValue: any,
  ): JsqlColumn | null;
  defineForeignKeys(
    to: JsqlModel,
    keys: JsqlKeys,
    onDeleteCascade: boolean,
    onUpdateCascade: boolean,
  ): JsqlDetail | null;
  defineOmitUpdate(...names: string[]): JsqlModel;
  defineHidden(...names: string[]): JsqlModel;
  defineModel(): JsqlModel;
  /** where filters `to`, joined to the keys with AND (no `or`); `{field: "x"}` (field("x")) reads x from the row. */
  defineRollup(
    name: string,
    to: JsqlModel,
    keys: JsqlKeys,
    selects: string[],
    operation: JsqlRollupOperation,
    where?: JsqlWhereItem | JsqlWhereItem[] | null,
  ): JsqlRollups;
  defineDetail(
    name: string,
    keys: JsqlKeys,
    rows: number,
    ...selects: string[]
  ): JsqlModel;
  defineMaster(
    name: string,
    to: JsqlModel,
    keys: JsqlKeys,
    toKeys: JsqlKeys,
    selects: string[],
    rows?: number,
  ): JsqlModel;
  defineCalcFunc(name: string, calc: JsqlCalcFunction): JsqlModel;
  /** Calc field computed by a script that sets `item.<name>`. */
  defineCalc(name: string, script: string): JsqlModel;

  // Triggers: a script the model keeps (replaced by name) and runs on every insert, update or delete, with OLD and NEW;
  // a before trigger can change NEW and a throw cancels the operation.
  // users.beforeInsert("normalize", "NEW.name = NEW.name.toUpperCase();", 1)
  beforeInsert(name: string, code: string, version: number): JsqlModel;
  beforeUpdate(name: string, code: string, version: number): JsqlModel;
  beforeDelete(name: string, code: string, version: number): JsqlModel;
  /** The same script as before insert and before update. */
  beforeInsertOrUpdate(name: string, code: string, version: number): JsqlModel;
  afterInsert(name: string, code: string, version: number): JsqlModel;
  afterUpdate(name: string, code: string, version: number): JsqlModel;
  afterDelete(name: string, code: string, version: number): JsqlModel;
  /** The same script as after insert and after update. */
  afterInsertOrUpdate(name: string, code: string, version: number): JsqlModel;

  // Lookups (null when missing)
  getCalcFunc(name: string): JsqlCalcFunction | null;
  detail(name: string): JsqlModel | null;
  master(name: string): JsqlQuery | null;
  bridge(name: string): JsqlModel | null;
  getColumn(name: string): JsqlColumn | null;
  getField(name: string): any;
  getFrom(): JsqlFrom | null;
  /** Relations of the model, by name. */
  getDetails(): { [name: string]: JsqlDetail };
  getMasters(): { [name: string]: JsqlMaster };
  getRollups(): { [name: string]: JsqlRollups };

  // Queries
  as(...as: string[]): JsqlQuery;
  join(to: JsqlModel, as: string, on: JsqlCondition[]): JsqlQuery;
  select(...fields: string[]): JsqlQuery;
  calc(...fields: string[]): JsqlQuery;
  /** A query with this where: one condition or a list, as in a query. */
  where(cond: JsqlWhereItem | JsqlWhereItem[]): JsqlQuery;
  count(): number;
  first(n: number): JsqlItems;
  queryTx(tx: JsqlTx | null, query: JsqlQueryDescriptor): JsqlQuery;
  query(query: JsqlQueryDescriptor): JsqlQuery;

  // Commands
  insert(data: JsqlRecord): JsqlCommand;
  bulk(data: JsqlRecord[]): JsqlCommand;
  update(data: JsqlRecord): JsqlCommand;
  delete(): JsqlCommand;
  upsert(data: JsqlRecord): JsqlCommand;
}

interface JsqlQuery {
  toJson(): JsqlRecord;
  /** Logs the SQL and does not run it. */
  debug(): JsqlQuery;
  /** Builds the SQL without running it. */
  test(): JsqlQuery;
  getField(name: string): any;
  getSelectField(name: string): any;
  /** Models of the query. */
  getFroms(): JsqlFrom[];
  /** Details resolved by detail(...), by name. */
  getDetails(): { [name: string]: JsqlQueryDetail };
  /** Masters resolved by master(...), by name. */
  getMasters(): { [name: string]: JsqlQueryDetail };
  /** Rollups resolved when the query runs, by name. */
  getRollups(): { [name: string]: JsqlQueryRollups };
  join(model: JsqlModel, as: string, on: JsqlCondition[]): JsqlQuery;
  leftJoin(model: JsqlModel, as: string, on: JsqlCondition[]): JsqlQuery;
  rightJoin(model: JsqlModel, as: string, on: JsqlCondition[]): JsqlQuery;
  fullJoin(model: JsqlModel, as: string, on: JsqlCondition[]): JsqlQuery;
  select(...fields: string[]): JsqlQuery;
  calc(...fields: string[]): JsqlQuery;
  detail(...fields: string[]): JsqlQuery;
  master(...fields: string[]): JsqlQuery;
  hidden(...fields: string[]): JsqlQuery;
  addCondition(cond: JsqlCondition): JsqlQuery;
  /** One condition or a list: where([{ status: { eq: "active" } }, { and: { age: { more: 18 } } }]). */
  where(cond: JsqlWhereItem | JsqlWhereItem[]): JsqlQuery;
  groupBy(...fields: string[]): JsqlQuery;
  having(cond: JsqlCondition): JsqlQuery;
  page(page: number): JsqlQuery;
  /** sorted: true (default) ascending, false descending. */
  orderBy(field: string, sorted?: boolean): JsqlQuery;
  allTx(tx: JsqlTx | null): JsqlItems;
  all(): JsqlItems;
  oneTx(tx: JsqlTx | null): JsqlItem;
  one(): JsqlItem;
  firstTx(tx: JsqlTx | null, n: number): JsqlItems;
  first(n: number): JsqlItems;
  limitTx(tx: JsqlTx | null, page: number, rows: number): JsqlItems;
  limit(page: number, rows: number): JsqlItems;
  existsTx(tx: JsqlTx | null): boolean;
  exists(): boolean;
  countTx(tx: JsqlTx | null): number;
  count(): number;
}

interface JsqlCommand {
  toJson(): JsqlRecord;
  /** Logs the SQL and does not run it. */
  debug(): JsqlCommand;
  /** Builds the SQL without running it. */
  test(): JsqlCommand;
  /** One condition or a list, as in a query. */
  where(cond: JsqlWhereItem | JsqlWhereItem[]): JsqlCommand;
  /** Fields returned by the command (RETURNING). */
  returning(...fields: string[]): JsqlCommand;
  /** Rows an update or delete works on; 0 = every row that matches the where. */
  limit(rows: number): JsqlCommand;
  execTx(tx: JsqlTx | null): JsqlItems;
  exec(): JsqlItems;
  oneTx(tx: JsqlTx | null): JsqlItem;
  one(): JsqlItem;
  beforeInsert(fn: JsqlTriggerFunction): JsqlCommand;
  beforeUpdate(fn: JsqlTriggerFunction): JsqlCommand;
  beforeDelete(fn: JsqlTriggerFunction): JsqlCommand;
  beforeInsertOrUpdate(fn: JsqlTriggerFunction): JsqlCommand;
  afterInsert(fn: JsqlTriggerFunction): JsqlCommand;
  afterUpdate(fn: JsqlTriggerFunction): JsqlCommand;
  afterInsertOrUpdate(fn: JsqlTriggerFunction): JsqlCommand;
  afterDelete(fn: JsqlTriggerFunction): JsqlCommand;
}

interface JsqlSeries {
  model(): JsqlModel;
  newSeries(tag: string, format: string): void;
  setSeries(tag: string, format: string, value: number): void;
  getSeries(tag: string): JsqlItem;
  deleteSeries(tag: string): void;
  /** Next code of the series, formatted; with tx (in a trigger, TX) it runs in that transaction. */
  genSerie(tag: string, tx?: JsqlTx): string;
  /** Next value of the series; with tx (in a trigger, TX) it runs in that transaction. */
  genValue(tag: string, tx?: JsqlTx): number;
}

interface JsqlPackage {
  /** A copy of source with its {{name}} variables applied from vars: "{{name}}" alone takes the value as is (number,
   * list, object), inside a text it is replaced by its text; in a list, a list value is expanded; {{user.id}} is a path.
   * Unknown variables are left as they are. Not for raw SQL text (values are sent as parameters). */
  bind<T = Record<string, any>>(source: T, vars: Record<string, any>): T;
  /** Database configured by the DB_* environment variables. */
  load(): JsqlDB;
  loadTo(dbName: string, ...hostName: string[]): JsqlDB;
  /** Connection of a driver from the keys of its GetParams; missing ones are read from DB_*. */
  newConnection(
    driver:
      | "postgres"
      | "sqlite"
      | "mysql"
      | "mssql"
      | "oracle"
      | (string & {}),
    params?: JsqlConnection,
  ): JsqlConnectionObject;
  /** Database from its params, without connecting (call init()). */
  newDB(params: JsqlConnectParams): JsqlDB;
  /** Database from its params, connected. */
  connectTo(params: JsqlConnectParams): JsqlDB;
  /** Creates the database of the params when it does not exist. timeoutMs: 0 or none never fails by timeout. */
  createDB(params: JsqlConnectParams, timeoutMs?: number): void;
  /** Database from its JSON (db.toJson()). */
  loadDb(params: JsqlRecord): JsqlDB;
  /** Drops the whole database. timeoutMs: 0 or none never fails by timeout. */
  dropDB(db: JsqlDB, timeoutMs?: number): void;
  newTx(): JsqlTx;
  newQuery(model: JsqlModel, ...as: string[]): JsqlQuery;
  defineSeries(db: JsqlDB, schema: string): JsqlSeries;
  detailKeys(key: string, foreignKey: string): JsqlKeys;
  rollupKeys(key: string, foreignKey: string): JsqlKeys;

  // Conditions
  where(field: any, value: any): JsqlCondition;
  and(field: any, operator: JsqlOperator, value: any): JsqlCondition;
  or(field: any, operator: JsqlOperator, value: any): JsqlCondition;
  eq(field: string, value: any): JsqlCondition;
  neg(field: string, value: any): JsqlCondition;
  less(field: string, value: any): JsqlCondition;
  lessEq(field: string, value: any): JsqlCondition;
  more(field: string, value: any): JsqlCondition;
  moreEq(field: string, value: any): JsqlCondition;
  like(field: string, value: any): JsqlCondition;
  in(field: string, values: any[]): JsqlCondition;
  notIn(field: string, values: any[]): JsqlCondition;
  is(field: string, value: any): JsqlCondition;
  isNot(field: string, value: any): JsqlCondition;
  null(field: string): JsqlCondition;
  notNull(field: string): JsqlCondition;
  between(field: string, min: any, max: any): JsqlCondition;
  notBetween(field: string, min: any, max: any): JsqlCondition;

  // Helpers
  addAuditLog(
    auditLog: JsqlRecord[],
    userId: string,
    action: string,
  ): JsqlRecord[];
  statusList(): string[];
  /** Replaces $1, $2… in sql by the quoted args. */
  sqlParse(sql: string, ...args: any[]): string;
  quoted(value: any): any;
  escapeSQLString(value: string): string;
  jsonString(value: any): string;
  /** Splits "name as alias"; returns [parts, ok]. */
  argWhitAs(arg: string): [string[], boolean];
  /** Splits "schema.name"; returns [parts, ok]. */
  argWhitSchema(arg: string): [string[], boolean];
}

/** Database of the model that runs the script. */
declare const db: JsqlDB;
/** Only in a model trigger (beforeInsert…): the transaction of the command that runs it. What the trigger writes goes in
 * it: db.getModel(…).insert(…).oneTx(TX), series.genSerie(tag, TX) (with SQLite, outside it waits forever). */
declare const TX: JsqlTx;
/** New transaction (same as jsql.newTx()). */
declare function newTx(): JsqlTx;
/** Constructors, conditions and helpers of jsql. */
declare const jsql: JsqlPackage;
/** Same as jsql.bind. */
declare function bind<T = Record<string, any>>(
  source: T,
  vars: Record<string, any>,
): T;
/** A plain copy of a Go json (keys readable, JSON.stringify works): const query = toJsMap(params.query). */
declare function toJsMap(source: Record<string, any>): Record<string, any>;

/** A reference to a field of the query (or of the evaluated object), as a condition value: {field: "name"}.
 * where([{ "A.total": { more: field("A.min_total") } }]) compares two columns; the name must be a field of the query
 * origins (one that does not resolve makes the condition false, never a text). */
type JsqlFieldRef = { field: string };
declare function field(name: string): JsqlFieldRef;
/** Aggregates over a field: {field: "sum(name)"}. In a query they are SQL aggregates; in evaluateObject they apply to
 * the attribute (a list: items->price over [{price: 2}, {price: 3}] is [2, 3]). */
declare function count(name: string): JsqlFieldRef;
declare function sum(name: string): JsqlFieldRef;
declare function avg(name: string): JsqlFieldRef;
declare function min(name: string): JsqlFieldRef;
declare function max(name: string): JsqlFieldRef;
/** Parts of a date: {field: "extract_year(name)"}… */
declare function year(name: string): JsqlFieldRef;
declare function month(name: string): JsqlFieldRef;
declare function day(name: string): JsqlFieldRef;
declare function hour(name: string): JsqlFieldRef;
declare function minute(name: string): JsqlFieldRef;
declare function second(name: string): JsqlFieldRef;
declare function intValue(name: string): JsqlFieldRef;
declare function floatValue(name: string): JsqlFieldRef;

/** Trigger scripts: the record before the change (empty in an insert). */
declare var OLD: JsqlRecord;
/** Trigger scripts: the record being saved; changes made in a before trigger are saved. */
declare var NEW: JsqlRecord;
/** Trigger scripts: lowercase alias of OLD. */
declare var old: JsqlRecord;
/** Calc scripts (defineCalc): the record; set item.<name> to calculate the field. */
declare var item: JsqlRecord;
