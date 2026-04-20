package service

import "errors"

var (
	errInvalidMonthFormat = errors.New("月份格式应为 YYYY-MM")
	errNightShiftRequired = errors.New("生成排班前请先导入当月夜班表")
	errNoActiveEmployees  = errors.New("当前小组没有可用员工")
	errNoEnabledPosts     = errors.New("当前小组没有启用中的岗位")
)
