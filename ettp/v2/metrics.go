package ettp

import (
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
)

/**
* Usage metrics of the gateway, in memory and per replica: per route (METHOD:path) the requests, the count by status
* class, the latency (average, approximate p95 from a fixed histogram, max), the last request and the last error; the
* totals of the gateway and a window of the last 60 minutes. The gateway's own routes (its package, s.Name: /metrics,
* /routes…) are counted per route with own = true but stay out of the totals and the window, so polling /metrics does
* not count as use. GET <parent>/metrics reads them and DELETE <parent>/metrics starts over.
**/

// metricsWindow: minutes kept by the window of the totals
const metricsWindow = 60

// latencyBounds: upper bounds (ms) of the buckets of the latency histogram; one more bucket takes what is above
var latencyBounds = [...]float64{1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000}

/**
* usage: Counters of a set of requests (a route, a package or the gateway)
**/
type usage struct {
	requests int64
	classes  [5]int64 // 1xx, 2xx, 3xx, 4xx, 5xx
	totalMs  float64
	maxMs    float64
	buckets  [len(latencyBounds) + 1]int64
}

/**
* add: Counts one request with its status and latency
* @param status int, ms float64
**/
func (u *usage) add(status int, ms float64) {
	u.requests++
	if class := status/100 - 1; class >= 0 && class < len(u.classes) {
		u.classes[class]++
	}
	u.totalMs += ms
	u.maxMs = math.Max(u.maxMs, ms)
	idx := sort.SearchFloat64s(latencyBounds[:], ms)
	u.buckets[idx]++
}

/**
* merge: Adds the counters of other
* @param other *usage
**/
func (u *usage) merge(other *usage) {
	u.requests += other.requests
	for i := range u.classes {
		u.classes[i] += other.classes[i]
	}
	u.totalMs += other.totalMs
	u.maxMs = math.Max(u.maxMs, other.maxMs)
	for i := range u.buckets {
		u.buckets[i] += other.buckets[i]
	}
}

/**
* p95: The upper bound of the bucket that holds the 95th percentile (the max when it is in the last one or below it)
* @return float64
**/
func (u *usage) p95() float64 {
	target := int64(math.Ceil(float64(u.requests) * 0.95))
	var seen int64
	for i, n := range u.buckets {
		seen += n
		if seen >= target && i < len(latencyBounds) {
			return math.Min(latencyBounds[i], u.maxMs)
		}
	}
	return u.maxMs
}

/**
* toJson
* @return et.Json
**/
func (u *usage) toJson() et.Json {
	avg := 0.0
	p95 := 0.0
	if u.requests > 0 {
		avg = u.totalMs / float64(u.requests)
		p95 = u.p95()
	}

	return et.Json{
		"requests": u.requests,
		"errors":   u.classes[3] + u.classes[4],
		"status": et.Json{
			"1xx": u.classes[0],
			"2xx": u.classes[1],
			"3xx": u.classes[2],
			"4xx": u.classes[3],
			"5xx": u.classes[4],
		},
		"latency": et.Json{
			"avg_ms": math.Round(avg*100) / 100,
			"p95_ms": math.Round(p95*100) / 100,
			"max_ms": math.Round(u.maxMs*100) / 100,
		},
	}
}

/**
* hit: One request as the metrics keep it (the last one and the last error of a route)
**/
type hit struct {
	At     time.Time `json:"at"`
	Method string    `json:"method"`
	Path   string    `json:"path"`
	Status int       `json:"status"`
	Ms     float64   `json:"ms"`
}

/**
* routeUsage: The usage of one route, under its own lock
**/
type routeUsage struct {
	mu        sync.Mutex
	id        string
	method    string
	path      string
	pkg       string
	kind      TypeRouter
	own       bool
	usage     usage
	last      *hit
	lastError *hit
}

/**
* toJson
* @return et.Json
**/
func (r *routeUsage) toJson() et.Json {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := r.usage.toJson()
	result["id"] = r.id
	result["method"] = r.method
	result["path"] = r.path
	result["package"] = r.pkg
	result["kind"] = r.kind.String()
	result["own"] = r.own
	result["last"] = r.last
	result["last_error"] = r.lastError
	return result
}

/**
* minute: One minute of the window
**/
type minute struct {
	at     int64 // unix minute
	usage  usage
	errors int64
}

/**
* Metrics: The usage of a gateway (see the comment at the top of the file)
**/
type Metrics struct {
	mu        sync.RWMutex // guards routes (the map); each route has its own lock
	routes    map[string]*routeUsage
	muTotals  sync.Mutex // guards since, totals, unmatched and minutes
	since     time.Time
	totals    usage
	unmatched int64
	minutes   [metricsWindow]minute
}

/**
* newMetrics
* @return *Metrics
**/
func newMetrics() *Metrics {
	return &Metrics{
		routes: make(map[string]*routeUsage),
		since:  time.Now(),
	}
}

/**
* route: The usage of the route of the resolver, created on its first request
* @param resolver *Resolver, method string, own bool
* @return *routeUsage
**/
func (m *Metrics) route(resolver *Resolver, method string, own bool) *routeUsage {
	id := method + ":" + resolver.Path
	m.mu.RLock()
	result, ok := m.routes[id]
	m.mu.RUnlock()
	if ok {
		return result
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if result, ok = m.routes[id]; ok {
		return result
	}
	result = &routeUsage{id: id, method: method, path: resolver.Path, pkg: resolver.packageName, kind: resolver.Kind, own: own}
	m.routes[id] = result
	return result
}

/**
* record: Counts a request resolved to a route; the gateway's own routes (own) stay out of the totals
* @param resolver *Resolver, r *http.Request, status int, start time.Time, own bool
**/
func (m *Metrics) record(resolver *Resolver, r *http.Request, status int, start time.Time, own bool) {
	now := time.Now()
	ms := float64(now.Sub(start).Microseconds()) / 1000
	if status == 0 {
		status = http.StatusOK
	}
	last := &hit{At: now, Method: r.Method, Path: r.URL.Path, Status: status, Ms: ms}

	route := m.route(resolver, r.Method, own)
	route.mu.Lock()
	route.usage.add(status, ms)
	route.last = last
	if status >= http.StatusBadRequest {
		route.lastError = last
	}
	route.mu.Unlock()

	if own {
		return
	}

	m.muTotals.Lock()
	m.totals.add(status, ms)
	at := now.Unix() / 60
	slot := &m.minutes[at%metricsWindow]
	if slot.at != at {
		*slot = minute{at: at}
	}
	slot.usage.add(status, ms)
	m.muTotals.Unlock()
}

/**
* recordUnmatched: Counts a request that resolved to no route (404 of the gateway)
**/
func (m *Metrics) recordUnmatched() {
	m.muTotals.Lock()
	m.unmatched++
	m.muTotals.Unlock()
}

/**
* Reset: Starts every counter over
**/
func (m *Metrics) Reset() {
	m.mu.Lock()
	m.routes = make(map[string]*routeUsage)
	m.mu.Unlock()

	m.muTotals.Lock()
	m.since = time.Now()
	m.totals = usage{}
	m.unmatched = 0
	m.minutes = [metricsWindow]minute{}
	m.muTotals.Unlock()
}

/**
* window: The last 60 minutes, oldest first, the ones without requests in zero
* @param now time.Time
* @return []et.Json
**/
func (m *Metrics) window(now time.Time) []et.Json {
	result := make([]et.Json, 0, metricsWindow)
	current := now.Unix() / 60
	for at := current - metricsWindow + 1; at <= current; at++ {
		slot := minute{at: at}
		if m.minutes[at%metricsWindow].at == at {
			slot = m.minutes[at%metricsWindow]
		}
		item := slot.usage.toJson()
		item["minute"] = time.Unix(at*60, 0).UTC().Format(time.RFC3339)
		result = append(result, item)
	}
	return result
}

/**
* ToJson: The usage of the gateway: since, uptime, totals (with unmatched), the window of minutes, each route and each
* package (the sum of its routes)
* @return et.Json
**/
func (m *Metrics) ToJson() et.Json {
	m.mu.RLock()
	routes := make([]*routeUsage, 0, len(m.routes))
	for _, route := range m.routes {
		routes = append(routes, route)
	}
	m.mu.RUnlock()

	routeItems := make([]et.Json, 0, len(routes))
	packages := map[string]*usage{}
	packageOwn := map[string]bool{}
	for _, route := range routes {
		routeItems = append(routeItems, route.toJson())
		route.mu.Lock()
		pkg, ok := packages[route.pkg]
		if !ok {
			pkg = &usage{}
			packages[route.pkg] = pkg
		}
		pkg.merge(&route.usage)
		packageOwn[route.pkg] = route.own
		route.mu.Unlock()
	}
	sort.Slice(routeItems, func(i, j int) bool { return routeItems[i].Str("id") < routeItems[j].Str("id") })

	packageItems := make([]et.Json, 0, len(packages))
	for name, pkg := range packages {
		item := pkg.toJson()
		item["package"] = name
		item["own"] = packageOwn[name]
		packageItems = append(packageItems, item)
	}
	sort.Slice(packageItems, func(i, j int) bool { return packageItems[i].Str("package") < packageItems[j].Str("package") })

	now := time.Now()
	m.muTotals.Lock()
	defer m.muTotals.Unlock()
	totals := m.totals.toJson()
	totals["unmatched"] = m.unmatched

	return et.Json{
		"since":    m.since.UTC().Format(time.RFC3339),
		"uptime_s": int64(now.Sub(m.since).Seconds()),
		"totals":   totals,
		"minutes":  m.window(now),
		"routes":   routeItems,
		"packages": packageItems,
	}
}
