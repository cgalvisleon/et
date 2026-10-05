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
type JsqlTypeColumn = "column" | "atrib" | "detail" | "master" | "rollup" | "calc_func" | "calc";

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
type JsqlRollupOperation = "count" | "sum" | "avg" | "min" | "max" | "row" | "object";

/** A where condition, built with the jsql.* functions (eq, more, like…). Opaque. */
interface JsqlCondition {
  readonly __condition: unique symbol;
}

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

/** Trigger function: change `newRecord` to change what is saved; throw to abort the command. */
type JsqlTriggerFunction = (tx: any, oldRecord: JsqlRecord, newRecord: JsqlRecord) => void;

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
  columns?: { name: string; type_column?: JsqlTypeColumn; type_data: JsqlTypeData; default?: any }[];
  hiddens?: string[];
  omit_updates?: string[];
  details?: {
    name: string;
    keys: JsqlKeys;
    rows?: number;
    select?: string[];
    columns?: { name: string; type_column?: JsqlTypeColumn; type_data: JsqlTypeData; default?: any }[];
    primary_keys?: { name: string; sorted?: boolean }[];
    indexes?: { name: string; sorted?: boolean }[];
    idx_field?: string;
    idt_field?: string;
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
  newModel(schema: string, name: string, version: number, userId: string): JsqlModel;
  /** Drops the model's table and removes the model. */
  removeModel(schema: string, name: string): void;
  getModel(schema: string, name: string): JsqlModel;
  define(define: JsqlDefine): JsqlModel;
  defineModel(schema: string, name: string, version: number, userId: string): JsqlModel;
  defineTenantModel(schema: string, name: string, version: number, userId: string): JsqlModel;
  defineProjectModel(schema: string, name: string, version: number, userId: string): JsqlModel;
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
  defineIndex(name: string, type: JsqlTypeData, defaultValue: any): JsqlIndex | null;
  definePrimaryKey(name: string, type: JsqlTypeData, defaultValue: any): JsqlIndex | null;
  defineUnique(name: string, type: JsqlTypeData, defaultValue: any): JsqlIndex | null;
  defineRequired(name: string, type: JsqlTypeData, defaultValue: any): JsqlIndex | null;
  defineColumn(name: string, type: JsqlTypeData, defaultValue: any): JsqlColumn | null;
  defineAttrib(name: string, type: JsqlTypeData, defaultValue: any): JsqlColumn | null;
  defineForeignKeys(to: JsqlModel, keys: JsqlKeys, onDeleteCascade: boolean, onUpdateCascade: boolean): JsqlRecord | null;
  defineOmitUpdate(...names: string[]): JsqlModel;
  defineHidden(...names: string[]): JsqlModel;
  defineModel(): JsqlModel;
  defineRollup(name: string, to: JsqlModel, keys: JsqlKeys, selects: string[], operation: JsqlRollupOperation): any;
  defineDetail(name: string, keys: JsqlKeys, rows: number, ...selects: string[]): JsqlModel;
  defineMaster(name: string, to: JsqlModel, keys: JsqlKeys, toKeys: JsqlKeys, selects: string[], rows?: number): JsqlModel;
  defineCalcFunc(name: string, calc: JsqlCalcFunction): JsqlModel;
  /** Calc field computed by a script that sets `item.<name>`. */
  defineCalc(name: string, script: string): JsqlModel;
  defineBeforeInsert(name: string, code: string, version: number): JsqlModel;
  defineBeforeUpdate(name: string, code: string, version: number): JsqlModel;
  defineBeforeDelete(name: string, code: string, version: number): JsqlModel;
  defineAfterInsert(name: string, code: string, version: number): JsqlModel;
  defineAfterUpdate(name: string, code: string, version: number): JsqlModel;
  defineAfterDelete(name: string, code: string, version: number): JsqlModel;

  // Triggers
  beforeInsert(fn: JsqlTriggerFunction): JsqlModel;
  beforeUpdate(fn: JsqlTriggerFunction): JsqlModel;
  beforeDelete(fn: JsqlTriggerFunction): JsqlModel;
  beforeInsertOrUpdate(fn: JsqlTriggerFunction): JsqlModel;
  afterInsert(fn: JsqlTriggerFunction): JsqlModel;
  afterUpdate(fn: JsqlTriggerFunction): JsqlModel;
  afterDelete(fn: JsqlTriggerFunction): JsqlModel;
  afterInsertOrUpdate(fn: JsqlTriggerFunction): JsqlModel;

  // Lookups (null when missing)
  getCalcFunc(name: string): JsqlCalcFunction | null;
  detail(name: string): JsqlModel | null;
  master(name: string): JsqlQuery | null;
  bridge(name: string): JsqlModel | null;
  getColumn(name: string): JsqlColumn | null;
  getField(name: string): any;
  getFrom(): JsqlRecord | null;

  // Queries
  as(...as: string[]): JsqlQuery;
  join(to: JsqlModel, as: string, on: JsqlCondition[]): JsqlQuery;
  select(...fields: string[]): JsqlQuery;
  calc(...fields: string[]): JsqlQuery;
  where(cond: JsqlCondition): JsqlQuery;
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
  where(cond: JsqlCondition): JsqlQuery;
  and(cond: JsqlCondition): JsqlQuery;
  or(cond: JsqlCondition): JsqlQuery;
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
  where(cond: JsqlCondition): JsqlCommand;
  and(cond: JsqlCondition): JsqlCommand;
  or(cond: JsqlCondition): JsqlCommand;
  /** Fields returned by the command (RETURNING). */
  return(...fields: string[]): JsqlCommand;
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
  /** Next code of the series, formatted. */
  genSerie(tag: string): string;
  /** Next value of the series. */
  genValue(tag: string): number;
}

interface JsqlPackage {
  /** Database configured by the DB_* environment variables. */
  load(): JsqlDB;
  loadTo(dbName: string, ...hostName: string[]): JsqlDB;
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
  addAuditLog(auditLog: JsqlRecord[], userId: string, action: string): JsqlRecord[];
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
/** New transaction (same as jsql.newTx()). */
declare function newTx(): JsqlTx;
/** Constructors, conditions and helpers of jsql. */
declare const jsql: JsqlPackage;

/** Trigger scripts: the record before the change (empty in an insert). */
declare var OLD: JsqlRecord;
/** Trigger scripts: the record being saved; changes made in a before trigger are saved. */
declare var NEW: JsqlRecord;
/** Trigger scripts: lowercase alias of OLD. */
declare var old: JsqlRecord;
/** Calc scripts (defineCalc): the record; set item.<name> to calculate the field. */
declare var item: JsqlRecord;
