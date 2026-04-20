package service

import "errors"

var (
	errInvalidMonthFormat = errors.New("month format must be YYYY-MM")
	errNightShiftRequired = errors.New("night shift records are required before generating schedule")
	errNoActiveEmployees  = errors.New("no active employees found in group")
	errNoEnabledPosts     = errors.New("no enabled posts found in group")
)
