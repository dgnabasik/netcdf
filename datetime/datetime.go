package datetime
// ISO 8601 format: use package iso8601 since The built-in RFC3333 time layout in Go is too restrictive to support any ISO8601 date-time.

import (
	"strconv"
	"time"
)

const (
	TimeFormat     = "2006-01-02T15:04:05Z" // yyyy-MM-ddThh:mm:ssZ UTC RFC3339 format. Do not save timezone.
	TimeFormat1    = "2006-01-02 15:04:05"
	TimeFormatNano = "2006-01-02T15:04:05.000Z07:00" // this is the preferred milliseconds version.
)

// someTime can be either a long or a readable dateTime string.
func GetStartTimeFromLongint(someTime string) (time.Time, error) {
	isLong, err := strconv.ParseInt(someTime, 10, 64)
	if err == nil {
		return time.Unix(isLong, 0), err
	}
	t, err := time.Parse(TimeFormat, someTime)
	if err != nil {
		return time.Unix(isLong, 0), err
	}
	return t, nil
}

// Process: 2017-01-18 10:40:00
func GetStartTimeFromString(someTime string) (time.Time, error) {
	t, err := time.Parse(TimeFormat1, someTime)
	if err != nil {
		return t, err
	}
	return t, nil
}

// Return yyyy-MM-dd
func GetDateStr(t time.Time) string {
	return t.Format("2006-01-02")
}

func StandardDate(dt time.Time) string {
	var a [20]byte
	var b = a[:0]                        // Using the a[:0] notation converts the fixed-size array to a slice type represented by b that is backed by this array.
	b = dt.AppendFormat(b, time.RFC3339) // AppendFormat() accepts type []byte. The allocated memory a is passed to AppendFormat().
	return string(b[0:10])
}

