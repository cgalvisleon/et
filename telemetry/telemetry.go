package telemetry

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/shirou/gopsutil/v3/mem"
)

var DefaultTelemetry func(next http.Handler) http.Handler

type Metrics struct {
	ReqID            string
	Channel          string
	TimeBegin        time.Time
	TimeEnd          time.Time
	TimeExec         time.Time
	SearchTime       time.Duration
	ResponseTime     time.Duration
	Downtime         time.Duration
	Latency          time.Duration
	NotFount         bool
	EndPoint         string
	Method           string
	RemoteAddr       string
	HostName         string
	HostRequest      string
	Proto            string
	Status           int
	ContentLength    int64
	MTotal           uint64
	MUsed            uint64
	MFree            uint64
	PFree            float64
	RequestsHost     *Request
	RequestsEndpoint *Request
	Scheme           string
}

type Request struct {
	Tag     string
	Day     int
	Hour    int
	Minute  int
	Seccond int
	Limit   int
}

func NewMetric(r *http.Request) *Metrics {
	result := &Metrics{}
	result.TimeBegin = time.Now()
	result.ReqID = utility.NewId()
	result.EndPoint = r.URL.Path
	result.Method = r.Method
	result.Proto = r.Proto
	result.RemoteAddr = r.RemoteAddr
	result.HostName, _ = os.Hostname()
	result.HostRequest = r.Host
	memory, err := mem.VirtualMemory()
	if err != nil {
		result.MFree = 0
		result.MTotal = 0
		result.MUsed = 0
		result.PFree = 0
	} else {
		result.MTotal = memory.Total
		result.MUsed = memory.Used
		result.MFree = memory.Total - memory.Used
		result.PFree = float64(result.MFree) / float64(result.MTotal) * 100
	}
	result.RequestsHost = result.CallRequests(result.HostName)
	result.RequestsEndpoint = result.CallRequests(result.EndPoint)
	result.Scheme = "http"
	if r.TLS != nil {
		result.Scheme = "https"
	}

	return result
}

func (m *Metrics) CallRequests(tag string) *Request {
	return &Request{
		Tag:     tag,
		Day:     cache.Count(strs.Format(`%s-%d`, tag, time.Now().Unix()/86400), 86400),
		Hour:    cache.Count(strs.Format(`%s-%d`, tag, time.Now().Unix()/3600), 3600),
		Minute:  cache.Count(strs.Format(`%s-%d`, tag, time.Now().Unix()/60), 60),
		Seccond: cache.Count(strs.Format(`%s-%d`, tag, time.Now().Unix()/1), 1),
		Limit:   envar.GetInt(400, "REQUESTS_LIMIT"),
	}
}

func (m *Metrics) CallExecute() {
	m.SearchTime = time.Since(m.TimeBegin)
	m.TimeExec = time.Now()
}

func (m *Metrics) printLn() et.Json {
	w := logs.Color(logs.NMagenta, fmt.Sprintf(" [%s]:", "TELEMETRY"))
	logs.CW(w, logs.NCyan, fmt.Sprintf(" [%s]:", strs.Uppcase(m.Channel)))
	logs.CW(w, logs.NCyan, fmt.Sprintf(" [%s]:", m.Method))
	logs.CW(w, logs.NCyan, fmt.Sprintf("%s %s", m.EndPoint, m.Proto))
	logs.CW(w, logs.NWhite, fmt.Sprintf(" from %s", m.RemoteAddr))
	status := fmt.Sprintf(" - %d %s", m.Status, getStatusDescription(m.Status))
	if m.Status >= 500 {
		logs.CW(w, logs.NRed, status)
	} else if m.Status >= 400 {
		logs.CW(w, logs.NYellow, status)
	} else if m.Status >= 300 {
		logs.CW(w, logs.NCyan, status)
	} else {
		logs.CW(w, logs.NGreen, status)
	}
	if m.NotFount {
		logs.CW(w, logs.NWhite, " Not Found ")
	} else {
		logs.CW(w, logs.NWhite, " Found ")
	}
	logs.CW(w, logs.NCyan, fmt.Sprintf(" %v%s", m.ContentLength, "KB"))
	logs.CW(w, logs.NWhite, " in ")
	if m.Latency < 500*time.Millisecond {
		logs.CW(w, logs.NGreen, "%s", m.Latency)
	} else if m.Latency < 5*time.Second {
		logs.CW(w, logs.NYellow, "%s", m.Latency)
	} else {
		logs.CW(w, logs.NRed, "%s", m.Latency)
	}
	logs.CW(w, logs.NRed, " Downtime:%s", m.Downtime)
	if m.RequestsHost.Seccond > m.RequestsHost.Limit {
		logs.CW(w, logs.NRed, " - Request S:%vM:%vH:%vD:%vL:%v", m.RequestsHost.Seccond, m.RequestsHost.Minute, m.RequestsHost.Hour, m.RequestsHost.Day, m.RequestsHost.Limit)
	} else {
		logs.CW(w, logs.NYellow, " - Request S:%vM:%vH:%vD:%vL:%v", m.RequestsHost.Seccond, m.RequestsHost.Minute, m.RequestsHost.Hour, m.RequestsHost.Day, m.RequestsHost.Limit)
	}

	logs.Println(w)

	result := et.Json{
		"reqID":         m.ReqID,
		"time_begin":    m.TimeBegin,
		"time_end":      m.TimeEnd,
		"time_exec":     m.TimeExec,
		"latency":       m.Latency,
		"search_time":   m.SearchTime,
		"response_time": m.ResponseTime,
		"host_name":     m.HostName,
		"remote_addr":   m.RemoteAddr,
		"request": et.Json{
			"end_point": m.EndPoint,
			"method":    m.Method,
			"status":    m.Status,
			"bytes":     m.ContentLength,
			"scheme":    m.Scheme,
			"host":      m.HostRequest,
		},
		"memory": et.Json{
			"unity":        "MB",
			"total":        m.MTotal / 1024 / 1024,
			"used":         m.MUsed / 1024 / 1024,
			"free":         m.MFree / 1024 / 1024,
			"percent_free": math.Floor(m.PFree*100) / 100,
		},
		"request_host": et.Json{
			"host":   m.RequestsHost.Tag,
			"day":    m.RequestsHost.Day,
			"hour":   m.RequestsHost.Hour,
			"minute": m.RequestsHost.Minute,
			"second": m.RequestsHost.Seccond,
			"limit":  m.RequestsHost.Limit,
		},
		"requests_endpoint": et.Json{
			"endpoint": m.RequestsEndpoint.Tag,
			"day":      m.RequestsEndpoint.Day,
			"hour":     m.RequestsEndpoint.Hour,
			"minute":   m.RequestsEndpoint.Minute,
			"second":   m.RequestsEndpoint.Seccond,
			"limit":    m.RequestsEndpoint.Limit,
		},
	}

	go event.Telemetry(result)

	if m.RequestsHost.Seccond > m.RequestsHost.Limit {
		go event.Overflow(result)
	}

	return result
}

func (m *Metrics) Done(res *http.Response) et.Json {
	m.TimeEnd = time.Now()
	m.Channel = "Done"
	m.ResponseTime = time.Since(m.TimeExec)
	m.Latency = time.Since(m.TimeBegin)
	m.Status = res.StatusCode
	m.ContentLength = res.ContentLength
	result := m.printLn()

	return result
}

func (m *Metrics) DoneResponse(status int, contentLength int64) et.Json {
	m.TimeEnd = time.Now()
	m.Channel = "Done-Reponse"
	m.ResponseTime = time.Since(m.TimeExec)
	m.Latency = time.Since(m.TimeBegin)
	m.Status = status
	m.ContentLength = contentLength
	result := m.printLn()

	return result
}

func (m *Metrics) DoneHandler() et.Json {
	m.TimeEnd = time.Now()
	m.Channel = "Done-Handler"
	m.ResponseTime = time.Since(m.TimeExec)
	m.Latency = time.Since(m.TimeBegin)
	m.Status = http.StatusOK
	m.ContentLength = 0
	result := m.printLn()

	return result
}

func (m *Metrics) NotFound(r *http.Request) et.Json {
	m.NotFount = true
	m.Channel = "NotFound"
	m.TimeEnd = time.Now()
	m.ResponseTime = time.Since(m.TimeExec)
	m.Latency = time.Since(m.TimeBegin)
	m.EndPoint = r.RequestURI
	m.Status = http.StatusNotFound
	m.ContentLength = 0
	result := m.printLn()

	return result
}
