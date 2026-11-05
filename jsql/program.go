package jsql

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jwf"
	"github.com/dop251/goja"
)

/**
* Programs (spec §8.4): a "steps" descriptor runs a list of steps in one transaction. A step is any JSON descriptor
* of the DB (query, command, define, administration) or a control step: if, for, set, break, continue, return,
* error and script. Variables ({{name}}) are applied with Bind before each step runs; {"query": {...}} as a value
* is a subquery resolved before the step.
**/

const (
	programKey             = "steps"
	defaultProgramMaxSteps = 10000
	programMaxDepth        = 32
	defaultRecordLimit     = 1000
)

// controlKeys are the keys of the control steps, in the order they are checked.
var controlKeys = []string{"if", "for", "set", "break", "continue", "return", "error", "script"}

// flow is how a list of steps ended: normally, or by break, continue or return.
type flow int

const (
	flowNext flow = iota
	flowBreak
	flowContinue
	flowReturn
)

/**
* program: The state of one run of a steps descriptor.
**/
type program struct {
	db       *DB
	tx       *Tx
	consts   et.Json
	vars     et.Json
	maxSteps int
	count    int
	limit    int
	last     any
	result   any
	returned bool
}

/**
* runProgram: Runs a steps descriptor {const, vars, do, max_steps} in tx, or in a new transaction it commits at the
* end (rolled back on any error). Returns what return gives or, without return, the result of the last step.
* @param tx *Tx, value any
* @return et.Items, error
**/
func (s *DB) runProgram(tx *Tx, value any) (et.Items, error) {
	def, ok := jsonOf(value)
	if !ok {
		return et.Items{}, fmt.Errorf(MSG_STEPS_INVALID, "steps must be an object {const, vars, do}")
	}
	steps, ok := listOf(def["do"])
	if !ok {
		return et.Items{}, fmt.Errorf(MSG_STEPS_INVALID, "do must be a list of steps")
	}

	limit := s.RecordLimit
	if limit <= 0 {
		limit = defaultRecordLimit
	}
	p := &program{
		db:       s,
		consts:   def.Json("const"),
		vars:     def.Json("vars").Clone(),
		maxSteps: def.ValInt(defaultProgramMaxSteps, "max_steps"),
		limit:    limit,
	}
	if err := p.validate(steps, 1, false); err != nil {
		return et.Items{}, err
	}

	tx, commit := getTx(tx)
	p.tx = tx
	if _, err := p.run(steps, 1); err != nil {
		if commit {
			tx.Rollback()
		}
		return et.Items{}, err
	}
	if commit {
		if err := tx.Commit(); err != nil {
			return et.Items{}, err
		}
	}

	if p.returned {
		return itemsOf(p.result), nil
	}
	return itemsOf(p.last), nil
}

/**
* validate: Checks the program before running anything: every step is an object, nesting stays under
* programMaxDepth, no step assigns a constant, and break/continue are inside a for.
* @param steps []any, depth int, inLoop bool
* @return error
**/
func (p *program) validate(steps []any, depth int, inLoop bool) error {
	if depth > programMaxDepth {
		return fmt.Errorf(MSG_STEPS_DEPTH, programMaxDepth)
	}
	for _, raw := range steps {
		step, ok := jsonOf(raw)
		if !ok {
			return fmt.Errorf(MSG_STEPS_INVALID, fmt.Sprintf("a step must be an object: %v", raw))
		}
		names := []string{step.Str("as")}
		switch controlOf(step) {
		case "set":
			for name := range step.Json("set") {
				names = append(names, name)
			}
		case "break", "continue":
			if !inLoop {
				return fmt.Errorf(MSG_STEPS_INVALID, controlOf(step)+" outside a for")
			}
		case "if":
			for _, key := range []string{"then", "else"} {
				branch, _ := listOf(step[key])
				if err := p.validate(branch, depth+1, inLoop); err != nil {
					return err
				}
			}
		case "for":
			head := step.Json("for")
			names = append(names, head.Str("key"), head.Str("value"), head.Str("var"))
			body, _ := listOf(step["do"])
			if err := p.validate(body, depth+1, true); err != nil {
				return err
			}
		}
		for _, name := range names {
			if _, ok := p.consts[name]; ok && name != "" {
				return fmt.Errorf(MSG_STEPS_CONST, name)
			}
		}
	}
	return nil
}

/**
* run: Runs a list of steps in order; it stops at the first error, break, continue or return.
* @param steps []any, depth int
* @return flow, error
**/
func (p *program) run(steps []any, depth int) (flow, error) {
	for _, raw := range steps {
		step, ok := jsonOf(raw)
		if !ok {
			return flowNext, fmt.Errorf(MSG_STEPS_INVALID, fmt.Sprintf("a step must be an object: %v", raw))
		}
		result, err := p.step(step, depth)
		if err != nil || result != flowNext {
			return result, err
		}
	}
	return flowNext, nil
}

/**
* step: Runs one step: a control step, or a DB descriptor whose result is kept as the last one (and in as).
* @param step et.Json, depth int
* @return flow, error
**/
func (p *program) step(step et.Json, depth int) (flow, error) {
	p.count++
	if p.count > p.maxSteps {
		return flowNext, fmt.Errorf(MSG_STEPS_MAX, p.maxSteps)
	}

	switch controlOf(step) {
	case "if":
		return p.runIf(step, depth)
	case "for":
		return p.runFor(step, depth)
	case "set":
		return flowNext, p.runSet(step)
	case "break":
		return flowBreak, nil
	case "continue":
		return flowContinue, nil
	case "return":
		value, err := p.resolve(step["return"])
		if err != nil {
			return flowNext, err
		}
		p.result, p.returned = value, true
		return flowReturn, nil
	case "error":
		return flowNext, fmt.Errorf("%v", p.bind(step["error"]))
	case "script":
		value, err := p.runScript(step.Str("script"))
		if err != nil {
			return flowNext, err
		}
		p.last = value
		p.store(step, value)
		return flowNext, nil
	}

	desc := et.Json{}
	for key, value := range step {
		if key != "as" && key != "one" {
			desc[key] = value
		}
	}
	bound, err := p.resolveStep(desc)
	if err != nil {
		return flowNext, err
	}
	if bound == nil {
		return flowNext, fmt.Errorf(MSG_STEPS_INVALID, fmt.Sprintf("unknown step: %s", step.ToString()))
	}
	items, err := p.db.queryJsonTx(p.tx, bound)
	if err != nil {
		return flowNext, err
	}
	p.last = items
	p.store(step, items)
	return flowNext, nil
}

/**
* runSet: Assigns the variables of a set step, with their variables and subqueries resolved.
* @param step et.Json
* @return error
**/
func (p *program) runSet(step et.Json) error {
	values, err := p.resolve(step.Json("set"))
	if err != nil {
		return err
	}
	for name, value := range values.(et.Json) {
		p.vars[name] = value
	}
	return nil
}

/**
* runIf: Evaluates the condition of an if step and runs then or else. A single command written inside the if
* object (insert, update…) is its then when there is no then.
* @param step et.Json, depth int
* @return flow, error
**/
func (p *program) runIf(step et.Json, depth int) (flow, error) {
	head, ok := jsonOf(p.bind(step["if"]))
	if !ok {
		return flowNext, fmt.Errorf(MSG_IF_INVALID, "if must be an object {value|data|query, where}")
	}
	holds, err := p.condition(head)
	if err != nil {
		return flowNext, err
	}

	branch := step["else"]
	if holds {
		branch = step["then"]
		if branch == nil {
			for _, kind := range keywordsCommand {
				if value, ok := head[kind]; ok {
					branch = []any{et.Json{kind: value}}
					break
				}
			}
		}
	}
	steps, _ := listOf(branch)
	return p.run(steps, depth+1)
}

/**
* condition: Whether the head of an if holds: value with EvaluateValue (field references read from the
* variables), data with EvaluateObject (an object, or a list like the rows of a query) or the rows of query.
* @param head et.Json
* @return bool, error
**/
func (p *program) condition(head et.Json) (bool, error) {
	where := head.ArrayJson("where")
	conditions, err := et.ToConditions(where)
	if err != nil {
		return false, err
	}

	if value, ok := head["value"]; ok {
		value, err := p.subqueries(value)
		if err != nil {
			return false, err
		}
		return et.EvaluateValueWrapper()(value, where, p.scope())
	}

	if value, ok := head["data"]; ok {
		value, err := p.subqueries(value)
		if err != nil {
			return false, err
		}
		if obj, ok := jsonOf(value); ok {
			return et.EvaluateObject(obj, conditions), nil
		}
		rows, ok := rowsOf(value)
		if !ok {
			return false, fmt.Errorf(MSG_IF_INVALID, fmt.Sprintf("data must be an object or a list of objects: %v", value))
		}
		return matchRows(rows, conditions, head), nil
	}

	if value, ok := head["query"]; ok {
		query, ok := jsonOf(value)
		if !ok {
			return false, fmt.Errorf(MSG_IF_INVALID, "query must be a query descriptor")
		}
		resolved, err := p.subqueries(query)
		if err != nil {
			return false, err
		}
		items, err := p.db.queryJsonTx(p.tx, resolved.(et.Json))
		if err != nil {
			return false, err
		}
		return matchRows(items.Result, conditions, head), nil
	}

	return false, fmt.Errorf(MSG_IF_INVALID, "if needs value, data or query")
}

/**
* matchRows: Whether rows meet the conditions: with exists, whether some row meets them (all of them without
* conditions) equals exists; otherwise match "any" (default) or "all". No rows never meets "any" nor "all".
* @param rows []et.Json, conditions []*et.Condition, head et.Json
* @return bool
**/
func matchRows(rows []et.Json, conditions []*et.Condition, head et.Json) bool {
	meets := func(row et.Json) bool {
		return len(conditions) == 0 || et.EvaluateObject(row, conditions)
	}
	if _, ok := head["exists"]; ok {
		found := false
		for _, row := range rows {
			if meets(row) {
				found = true
				break
			}
		}
		return found == head.Bool("exists")
	}

	if len(rows) == 0 {
		return false
	}
	all := strings.EqualFold(head.Str("match"), "all")
	for _, row := range rows {
		if meets(row) != all {
			return !all
		}
	}
	return all
}

/**
* runFor: Runs a for step in one of its forms: range (a list, an object by sorted keys or a number n as 0..n-1),
* counter (var, from, to excluded, step) or while (conditions on the variables, checked before each turn). Going
* over max (DB_RECORD_LIMIT by default) is an error. key, value and var exist only inside do.
* @param step et.Json, depth int
* @return flow, error
**/
func (p *program) runFor(step et.Json, depth int) (flow, error) {
	raw, ok := jsonOf(step["for"])
	if !ok {
		return flowNext, fmt.Errorf(MSG_FOR_INVALID, "for must be an object {range|var|while, ...}")
	}
	body, _ := listOf(step["do"])
	head, ok := jsonOf(p.bind(raw))
	if !ok {
		return flowNext, fmt.Errorf(MSG_FOR_INVALID, "for must be an object")
	}
	max := head.ValInt(p.limit, "max")
	if max <= 0 {
		return flowNext, fmt.Errorf(MSG_FOR_INVALID, "max must be greater than 0")
	}
	keyName, valueName, varName := head.Str("key"), head.Str("value"), head.Str("var")
	defer p.keep(keyName, valueName, varName)()

	turn := func(key, value any) (flow, error) {
		if keyName != "" {
			p.vars[keyName] = key
		}
		if valueName != "" {
			p.vars[valueName] = value
		}
		if varName != "" {
			p.vars[varName] = value
		}
		return p.run(body, depth+1)
	}

	switch {
	case head["while"] != nil:
		for i := 0; ; i++ {
			current, _ := jsonOf(p.bind(raw))
			conditions, err := et.ToConditions(current.ArrayJson("while"))
			if err != nil {
				return flowNext, err
			}
			if !et.EvaluateObject(p.scope(), conditions) {
				return flowNext, nil
			}
			if i >= max {
				return flowNext, fmt.Errorf(MSG_FOR_MAX, max)
			}
			if result, stop, err := loopFlow(turn(i, i)); stop {
				return result, err
			}
		}
	case head["range"] != nil:
		value, err := p.subqueries(head["range"])
		if err != nil {
			return flowNext, err
		}
		keys, values, err := rangeOf(value)
		if err != nil {
			return flowNext, err
		}
		if len(keys) > max {
			return flowNext, fmt.Errorf(MSG_FOR_MAX, max)
		}
		for i := range keys {
			if result, stop, err := loopFlow(turn(keys[i], values[i])); stop {
				return result, err
			}
		}
		return flowNext, nil
	case varName != "":
		from, okFrom := integerOf(head.ValAny(0, "from"))
		to, okTo := integerOf(head["to"])
		inc, okStep := integerOf(head.ValAny(1, "step"))
		if !okFrom || !okTo || !okStep || inc == 0 {
			return flowNext, fmt.Errorf(MSG_FOR_INVALID, "from, to and step must be integers, step not 0")
		}
		count := 0
		if (inc > 0 && to > from) || (inc < 0 && to < from) {
			count = int(math.Ceil(float64(to-from) / float64(inc)))
		}
		if count > max {
			return flowNext, fmt.Errorf(MSG_FOR_MAX, max)
		}
		for i, n := 0, from; i < count; i, n = i+1, n+inc {
			if result, stop, err := loopFlow(turn(i, n)); stop {
				return result, err
			}
		}
		return flowNext, nil
	}
	return flowNext, fmt.Errorf(MSG_FOR_INVALID, "for needs range, var or while")
}

/**
* loopFlow: What a loop does after a turn: stop with an error or a return, stop normally after a break, or go on.
* @param result flow, err error
* @return flow, bool (stop the loop), error
**/
func loopFlow(result flow, err error) (flow, bool, error) {
	switch {
	case err != nil:
		return flowNext, true, err
	case result == flowReturn:
		return flowReturn, true, nil
	case result == flowBreak:
		return flowNext, true, nil
	}
	return flowNext, false, nil
}

/**
* keep: Saves the variables a for uses (key, value, var) and returns the function that restores them after it.
* @param names ...string
* @return func()
**/
func (p *program) keep(names ...string) func() {
	saved := et.Json{}
	missing := []string{}
	for _, name := range names {
		if name == "" {
			continue
		}
		if value, ok := p.vars[name]; ok {
			saved[name] = value
		} else {
			missing = append(missing, name)
		}
	}
	return func() {
		for name, value := range saved {
			p.vars[name] = value
		}
		for _, name := range missing {
			delete(p.vars, name)
		}
	}
}

/**
* runScript: Runs JavaScript (jwf) with the standard bindings, the jsql Wrapper, TX (the program's transaction),
* vars (the program variables: what the script writes stays) and consts (a copy). The code is not bound: values
* reach it through vars, never as code.
* @param code string
* @return any (the last value of the script), error
**/
func (p *program) runScript(code string) (any, error) {
	bindings := jwf.Bindings(nil)
	for name, binding := range Wrapper(p.db) {
		bindings[name] = binding
	}
	bindings["TX"] = wrapTx(p.tx)
	bindings["vars"] = jwf.Binder(func(vm *goja.Runtime) goja.Value { return et.ToJS(vm, p.vars) })
	consts := p.consts.Clone()
	bindings["consts"] = jwf.Binder(func(vm *goja.Runtime) goja.Value { return et.ToJS(vm, consts) })
	value, _, err := jwf.Run(code, bindings)
	return value, err
}

/**
* store: Keeps the result of a step in its as variable: the rows (a list) or, with one, the first row.
* @param step et.Json, value any
**/
func (p *program) store(step et.Json, value any) {
	name := step.Str("as")
	if name == "" {
		return
	}
	if step.Bool("one") {
		value = firstRowOf(value)
	} else if items, ok := value.(et.Items); ok {
		value = items.Result
	}
	p.vars[name] = value
}

/**
* scope: The values a step sees: the variables and, over them, the constants.
* @return et.Json
**/
func (p *program) scope() et.Json {
	result := make(et.Json, len(p.vars)+len(p.consts))
	for name, value := range p.vars {
		result[name] = value
	}
	for name, value := range p.consts {
		result[name] = value
	}
	return result
}

/**
* bind: Applies the variables and constants to a value (Bind).
* @param value any
* @return any
**/
func (p *program) bind(value any) any {
	return bindValue(value, p.scope())
}

/**
* resolve: Applies the variables and then resolves the subqueries of a value.
* @param value any
* @return any, error
**/
func (p *program) resolve(value any) (any, error) {
	return p.subqueries(p.bind(value))
}

/**
* resolveStep: resolve for a DB descriptor, leaving its scripts (before_*, after_*) as they are: values reach a
* script through NEW/OLD, never by being written into its code.
* @param desc et.Json
* @return et.Json, error
**/
func (p *program) resolveStep(desc et.Json) (et.Json, error) {
	result, err := p.resolve(withoutScripts(desc))
	if err != nil {
		return nil, err
	}
	bound, ok := result.(et.Json)
	if !ok {
		return nil, nil
	}
	for key, value := range desc {
		bound[key] = scriptsOf(key, value, bound[key])
	}
	return bound, nil
}

/**
* withoutScripts: A copy of a descriptor (and of the command inside it) without its script keys.
* @param desc et.Json
* @return et.Json
**/
func withoutScripts(desc et.Json) et.Json {
	result := et.Json{}
	for key, value := range desc {
		if isScriptKey(key) {
			continue
		}
		if inner, ok := jsonOf(value); ok {
			value = withoutScripts(inner)
		}
		result[key] = value
	}
	return result
}

/**
* scriptsOf: Puts back the script keys removed by withoutScripts into a bound value.
* @param key string, original, bound any
* @return any
**/
func scriptsOf(key string, original, bound any) any {
	if isScriptKey(key) {
		return original
	}
	inner, ok := jsonOf(original)
	if !ok {
		return bound
	}
	result, ok := jsonOf(bound)
	if !ok {
		return bound
	}
	for k, v := range inner {
		result[k] = scriptsOf(k, v, result[k])
	}
	return result
}

/**
* isScriptKey: Whether a descriptor key holds trigger scripts (before_insert, after_update…).
* @param key string
* @return bool
**/
func isScriptKey(key string) bool {
	key = strings.ToLower(key)
	return strings.HasPrefix(key, "before_") || strings.HasPrefix(key, "after_")
}

/**
* subqueries: Replaces every {"query": {...}} (optionally with "one": true) of a value by its result, run in the
* program's transaction: the rows, or the list of values when they have one column; with one, the first row (or its
* only value), nil without rows. Without its own limit, more than DB_RECORD_LIMIT rows is an error.
* @param value any
* @return any, error
**/
func (p *program) subqueries(value any) (any, error) {
	if obj, ok := jsonOf(value); ok {
		if query, one, ok := subqueryOf(obj); ok {
			return p.runSubquery(query, one)
		}
		result := et.Json{}
		for key, item := range obj {
			resolved, err := p.subqueries(item)
			if err != nil {
				return nil, err
			}
			result[key] = resolved
		}
		return result, nil
	}
	if list, ok := listOf(value); ok && value != nil {
		if _, isText := value.(string); !isText {
			result := make([]any, 0, len(list))
			for _, item := range list {
				resolved, err := p.subqueries(item)
				if err != nil {
					return nil, err
				}
				result = append(result, resolved)
			}
			return result, nil
		}
	}
	return value, nil
}

/**
* subqueryOf: Whether an object is a subquery: only the keys query (a descriptor with from) and one.
* @param obj et.Json
* @return et.Json, bool (one), bool (is a subquery)
**/
func subqueryOf(obj et.Json) (et.Json, bool, bool) {
	query, ok := jsonOf(obj["query"])
	if !ok || query["from"] == nil {
		return nil, false, false
	}
	for key := range obj {
		if key != "query" && key != "one" {
			return nil, false, false
		}
	}
	return query, obj.Bool("one"), true
}

/**
* runSubquery: Runs a subquery and gives its value (see subqueries).
* @param query et.Json, one bool
* @return any, error
**/
func (p *program) runSubquery(query et.Json, one bool) (any, error) {
	resolved, err := p.subqueries(query)
	if err != nil {
		return nil, err
	}
	query = resolved.(et.Json)
	_, limited := query["limit"]
	if one && !limited {
		query["limit"] = 1
	}
	items, err := p.db.queryJsonTx(p.tx, query)
	if err != nil {
		return nil, err
	}

	if !one && !limited && items.Count >= p.limit {
		probe := query.Clone()
		probe["limit"], probe["offset"] = 1, items.Count
		more, err := p.db.queryJsonTx(p.tx, probe)
		if err != nil {
			return nil, err
		}
		if more.Count > 0 {
			return nil, fmt.Errorf(MSG_SUBQUERY_MAX, p.limit)
		}
	}

	single := len(items.Result) > 0
	for _, row := range items.Result {
		if len(row) != 1 {
			single = false
			break
		}
	}
	valueOf := func(row et.Json) any {
		if !single {
			return row
		}
		for _, value := range row {
			return value
		}
		return nil
	}

	if one {
		if len(items.Result) == 0 {
			return nil, nil
		}
		return valueOf(items.Result[0]), nil
	}
	result := make([]any, 0, len(items.Result))
	for _, row := range items.Result {
		result = append(result, valueOf(row))
	}
	return result, nil
}

/**
* controlOf: The control key of a step (if, for, set…), or "" for a DB descriptor.
* @param step et.Json
* @return string
**/
func controlOf(step et.Json) string {
	for _, key := range controlKeys {
		if _, ok := step[key]; ok {
			return key
		}
	}
	return ""
}

/**
* listOf: The elements of any slice (a list of steps, the rows of a query…).
* @param value any
* @return []any, bool
**/
func listOf(value any) ([]any, bool) {
	switch v := value.(type) {
	case nil:
		return []any{}, false
	case []any:
		return v, true
	case []byte:
		return nil, false
	case et.Items:
		return listOf(v.Result)
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice {
		return nil, false
	}
	result := make([]any, rv.Len())
	for i := range result {
		result[i] = rv.Index(i).Interface()
	}
	return result, true
}

/**
* rowsOf: A list whose elements are all objects, as rows.
* @param value any
* @return []et.Json, bool
**/
func rowsOf(value any) ([]et.Json, bool) {
	list, ok := listOf(value)
	if !ok {
		return nil, false
	}
	result := make([]et.Json, 0, len(list))
	for _, item := range list {
		row, ok := jsonOf(item)
		if !ok {
			return nil, false
		}
		result = append(result, row)
	}
	return result, true
}

/**
* rangeOf: The keys and values a for range walks: a list (index, element), an object (its keys sorted, value)
* or a non-negative integer n (0..n-1, as key and value).
* @param value any
* @return []any (keys), []any (values), error
**/
func rangeOf(value any) ([]any, []any, error) {
	if n, ok := integerOf(value); ok {
		if n < 0 {
			return nil, nil, fmt.Errorf(MSG_FOR_RANGE, value)
		}
		keys := make([]any, n)
		for i := range keys {
			keys[i] = i
		}
		return keys, keys, nil
	}
	if obj, ok := jsonOf(value); ok {
		names := make([]string, 0, len(obj))
		for name := range obj {
			names = append(names, name)
		}
		sort.Strings(names)
		keys, values := make([]any, len(names)), make([]any, len(names))
		for i, name := range names {
			keys[i], values[i] = name, obj[name]
		}
		return keys, values, nil
	}
	if _, isText := value.(string); !isText {
		if list, ok := listOf(value); ok {
			keys := make([]any, len(list))
			for i := range list {
				keys[i] = i
			}
			return keys, list, nil
		}
	}
	return nil, nil, fmt.Errorf(MSG_FOR_RANGE, value)
}

/**
* integerOf: A whole number of any numeric type (a float like 3.0 counts; 3.5 does not).
* @param value any
* @return int, bool
**/
func integerOf(value any) (int, bool) {
	switch v := value.(type) {
	case json.Number:
		n, err := v.Int64()
		return int(n), err == nil
	case float64:
		return int(v), v == math.Trunc(v)
	case float32:
		return int(v), float64(v) == math.Trunc(float64(v))
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(rv.Uint()), true
	}
	return 0, false
}

/**
* firstRowOf: The first row of a result (Items or a list), or the value itself.
* @param value any
* @return any
**/
func firstRowOf(value any) any {
	if items, ok := value.(et.Items); ok {
		if len(items.Result) == 0 {
			return nil
		}
		return items.Result[0]
	}
	if _, isText := value.(string); !isText {
		if list, ok := listOf(value); ok {
			if len(list) == 0 {
				return nil
			}
			return list[0]
		}
	}
	return value
}

/**
* itemsOf: The result of a program as Items: Items as they are, an object as one row, a list of objects as rows;
* any other value goes in a row {value}.
* @param value any
* @return et.Items
**/
func itemsOf(value any) et.Items {
	switch v := value.(type) {
	case nil:
		return et.Items{Result: []et.Json{}}
	case et.Items:
		return v
	case et.Item:
		return et.Items{Ok: v.Ok, Count: 1, Result: []et.Json{v.Result}}
	}
	if obj, ok := jsonOf(value); ok {
		return et.Items{Ok: true, Count: 1, Result: []et.Json{obj}}
	}
	if _, isText := value.(string); !isText {
		if rows, ok := rowsOf(value); ok {
			return et.Items{Ok: len(rows) > 0, Count: len(rows), Result: rows}
		}
	}
	return et.Items{Ok: true, Count: 1, Result: []et.Json{{"value": value}}}
}
