package timezone

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/cgalvisleon/et/config"
)

type Layout string

const (
	RFC3339Nano        Layout = "RFC3339Nano"
	RFC3339            Layout = "RFC3339"
	YYYYMMDDTHHMMSSZ   Layout = "2006-01-02T15:04:05Z"
	YYYYMMDDTHHMMSSSSZ Layout = "2006-01-02T15:04:05.000Z"
)

var loc *time.Location
var layouts = map[Layout]string{
	RFC3339Nano:        time.RFC3339Nano,
	RFC3339:            time.RFC3339,
	YYYYMMDDTHHMMSSZ:   "2006-01-02T15:04:05Z",
	YYYYMMDDTHHMMSSSSZ: "2006-01-02T15:04:05.000Z",
}
var layout = layouts[RFC3339]

func init() {
	timezone := config.GetStr("TIMEZONE", "America/Bogota")
	var err error
	loc, err = time.LoadLocation(timezone)
	if err != nil {
		panic(err)
	}

	layoutTime := config.GetStr("LAYOUT_TIME", "RFC3339")
	var ok bool
	layout, ok = layouts[Layout(layoutTime)]
	if !ok {
		layout = layouts[RFC3339]
	}
}

/**
* Now
* @return time.Time
**/
func Now() time.Time {
	result := time.Now().In(loc)
	return result
}

/**
* NowStr
* @return string
**/
func NowStr() string {
	return Now().Format(layout)
}

/**
* Add
* @param d time.Duration
* @return time.Time
**/
func Add(d time.Duration) time.Time {
	return time.Now().In(loc).Add(d)
}

/**
* Location
* @return *time.Location
* Remember to this function use ZONEINFO variable
**/
func Location() *time.Location {
	return loc
}

/**
* Format
* @param t time.Time, layout Layout
* @return string
**/
func Format(t time.Time, layout Layout) string {
	return t.Format(layouts[layout])
}

/**
* Parse
* @param layout, value string
* @return time.Time, error
**/
func Parse(layout Layout, value string) (time.Time, error) {
	current, err := time.ParseInLocation(layouts[layout], value, loc)
	if err != nil {
		if strings.Count(value, "+") == 2 || strings.Count(value, "-") == 2 {
			layout = YYYYMMDDTHHMMSSZ
		} else {
			layout = YYYYMMDDTHHMMSSSSZ
		}
		return time.ParseInLocation(layouts[layout], value, loc)
	}

	return current, nil
}

/**
* FormatMDYYYY
* @param layout, value string
* @return string
**/
func FormatMDYYYY(value string) string {
	t, err := Parse("2006-01-02T15:04:05", value)
	if err != nil {
		return value
	}

	months := map[time.Month]string{
		time.January:   "Ene",
		time.February:  "Feb",
		time.March:     "Mar",
		time.April:     "Abr",
		time.May:       "May",
		time.June:      "Jun",
		time.July:      "Jul",
		time.August:    "Ago",
		time.September: "Sep",
		time.October:   "Oct",
		time.November:  "Nov",
		time.December:  "Dic",
	}

	return fmt.Sprintf("%s %02d %d", months[t.Month()], t.Day(), t.Year())
}
