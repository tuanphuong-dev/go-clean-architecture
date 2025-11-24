package utils

import "time"

// NowUnixMillis returns the current time in Unix milliseconds.
func NowUnixMillis() int64 {
	return time.Now().UnixMilli()
}

// MonthStart returns the start time of the month of the given time.
func MonthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// MonthStartUnixMilli returns the start time of the month in Unix milliseconds.
func MonthStartUnixMilli(t time.Time) int64 {
	return MonthStart(t).UnixMilli()
}

// MonthEnd returns the end time of the month (i.e., one millisecond before next month's start).
func MonthEnd(t time.Time) time.Time {
	return NextMonthStart(t).Add(-time.Millisecond)
}

// MonthEndUnixMilli returns the end time of the month in Unix milliseconds.
func MonthEndUnixMilli(t time.Time) int64 {
	return MonthEnd(t).UnixMilli()
}

// NextMonthStart returns the start time of the next month.
func NextMonthStart(t time.Time) time.Time {
	return MonthStart(t).AddDate(0, 1, 0)
}

// NextMonthStartUnixMilli returns the start time of the next month in Unix milliseconds.
func NextMonthStartUnixMilli(t time.Time) int64 {
	return NextMonthStart(t).UnixMilli()
}

// LastMonthStart returns the start time of the previous month.
func LastMonthStart(t time.Time) time.Time {
	return MonthStart(t).AddDate(0, -1, 0)
}

// LastMonthStartUnixMilli returns the start time of the previous month in Unix milliseconds.
func LastMonthStartUnixMilli(t time.Time) int64 {
	return LastMonthStart(t).UnixMilli()
}

// LastMonthEnd returns the end time of the previous month (i.e., one millisecond before this month's start).
func LastMonthEnd(t time.Time) time.Time {
	return MonthStart(t).Add(-time.Millisecond)
}

// LastMonthEndUnixMilli returns the end time of the previous month in Unix milliseconds.
func LastMonthEndUnixMilli(t time.Time) int64 {
	return LastMonthEnd(t).UnixMilli()
}

// TruncateTime truncates a time.Time to the specified groupBy unit
func TruncateTime(t time.Time, groupBy string) time.Time {
	switch groupBy {
	case "day":
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case "month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case "year":
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}

// GenerateTimeSeries creates a series of time points from start to end with intervals based on groupBy
func GenerateTimeSeries(fromTime, toTime time.Time, groupBy string) []time.Time {
	var result []time.Time

	current := fromTime
	for current.Before(toTime) || current.Equal(toTime) {
		result = append(result, current)

		// Increment current by one unit based on groupBy
		switch groupBy {
		case "day":
			current = current.AddDate(0, 0, 1)
		case "month":
			current = current.AddDate(0, 1, 0)
		case "year":
			current = current.AddDate(1, 0, 0)
		}
	}

	return result
}

// GetTimeFormat returns the appropriate time format based on groupBy
func GetTimeFormat(groupBy string) string {
	formats := map[string]string{
		"day":   "2006-01-02",
		"month": "2006-01",
		"year":  "2006",
	}
	return formats[groupBy]
}
