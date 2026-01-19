package timezone

import (
	"fmt"
	"strings"
	"time"

	"github.com/cgalvisleon/et/envar"
)

var loc *time.Location

/**
* Now
* @return time.Time
* Remember to this function use ZONEINFO variable
**/
func Now() time.Time {
	timezone := envar.GetStr("TIMEZONE", "America/Bogota")

	if loc == nil {
		loc = time.FixedZone(timezone, -5*60*60)
	}

	now := time.Now().UTC()

	return now.In(loc)
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
* Parse
* @param layout, value string
* @return time.Time, error
**/
func Parse(layout string, value string) (time.Time, error) {
	current, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		if strings.Count(value, "+") == 2 || strings.Count(value, "-") == 2 {
			layout = "2006-01-02 15:04:05 -0700 -0700"
		} else {
			layout = "2006-01-02 15:04:05 -0700"
		}

		return time.ParseInLocation(layout, value, loc)
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

/**
* Add
* @param d time.Duration
* @return time.Time
**/
func Add(d time.Duration) time.Time {
	timezone := envar.GetStr("TIMEZONE", "America/Bogota")
	if loc == nil {
		loc = time.FixedZone(timezone, -5*60*60)
	}

	now := time.Now().UTC().Add(d)

	return now.In(loc)
}

/**
* NowStr
* @return string
**/
func NowStr() string {
	return Now().Format("2006/01/02 15:04:05")
}
