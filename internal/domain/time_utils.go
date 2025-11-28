package domain

import "time"

const (
	DatetimeLayout     = "2006-01-02T15:04:05Z"
	MonthlyLayout      = "2006-01"
	OnlyDateTimeLayout = "2006-01-02 15:04:05"
	OnlyDate           = "2006-01-02"
	ShotMonth          = "Jan"
	DatetimeZoneLayout = "2006-01-02 15:04:05.000 -0700"
)

// DateTimeLayout returns the datetime layout
func DateTimeLayout() string {
	return DatetimeLayout
}

// EndOfDay returns the end of the day (23:59:59) of the given date.
func EndOfDay(date time.Time) time.Time {
	location, _ := time.LoadLocation("Asia/Bangkok")
	date = date.In(location)
	y, m, d := date.Date()
	return time.Date(y, m, d, 23, 59, 59, 0, location)
}

// BeginningOfMonth beginning of month
func BeginningOfMonth(date time.Time) time.Time {
	location, _ := time.LoadLocation("Asia/Bangkok")
	date = date.In(location)
	y, m, _ := date.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, location)
}

// EndOfMonth end of month
func EndOfMonth(date time.Time) time.Time {
	date = BeginningOfMonth(date)
	return date.AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// BeginningOfYear beginning of year
func BeginningOfYear(date time.Time) time.Time {
	location, _ := time.LoadLocation("Asia/Bangkok")
	date = date.In(location)
	y, _, _ := date.Date()
	return time.Date(y, time.January, 1, 0, 0, 0, 0, location)
}

// EndOfYear end of year
func EndOfYear(date time.Time) time.Time {
	date = BeginningOfYear(date)
	return date.AddDate(1, 0, 0).Add(-time.Nanosecond)
}
