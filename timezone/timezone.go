package timezone

import (
	"time"

	"github.com/cgalvisleon/et/envar"
)

var loc *time.Location
var timezone = envar.GetStr("America/Bogota", "ZONEINFO")

/**
* NowTime
* @return time.Time
* Remember to this function use ZONEINFO variable
**/
func NowTime() time.Time {
	if loc == nil {
		loc = time.FixedZone(timezone, -5*60*60)
	}

	now := time.Now().UTC()

	return now.In(loc)
}

/**
* Add
* @param d time.Duration
* @return time.Time
**/
func Add(d time.Duration) time.Time {
	if loc == nil {
		loc = time.FixedZone(timezone, -5*60*60)
	}

	now := time.Now().UTC().Add(d)

	return now.In(loc)
}

/**
* Now
* @return string
**/
func Now() string {
	return NowTime().Format("2006/01/02 15:04:05")
}
