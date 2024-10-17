package timezone

import (
	"time"
)

var loc *time.Location

/**
* NowTime
* @return time.Time
* Remember to this function use ZONEINFO variable
**/
func NowTime() time.Time {
	if loc == nil {
		loc = time.FixedZone("America/Bogota", -5*60*60)
	}

	now := time.Now().UTC()

	return now.In(loc)
}

/**
* Now
* @return string
**/
func Now() string {
	return NowTime().Format("2006/01/02 15:04:05")
}
