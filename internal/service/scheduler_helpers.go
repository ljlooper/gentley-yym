package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseFixedDays(s string) []int {
	var days []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		d, err := strconv.Atoi(part)
		if err == nil && d > 0 {
			days = append(days, d)
		}
	}
	return days
}

func weekdayOf(month string, day int) time.Weekday {
	t, err := time.Parse("2006-01-02", fmt.Sprintf("%s-%02d", month, day))
	if err != nil {
		return time.Monday
	}
	return t.Weekday()
}

func previousMonth(month string) (string, int, error) {
	current, err := time.Parse("2006-01", month)
	if err != nil {
		return "", 0, errInvalidMonthFormat
	}
	prev := current.AddDate(0, -1, 0)
	prevTotalDays := time.Date(prev.Year(), prev.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return prev.Format("2006-01"), prevTotalDays, nil
}
