package models

import (
	"strings"
	"time"
)

type EmployeeRole string

const (
	ShiftRest     = "rest"
	ShiftOffNight = "off_after_night"
)

type Group struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Name       string    `json:"name" gorm:"size:64;uniqueIndex:idx_dept_group;not null"`
	Department string    `json:"department" gorm:"size:64;uniqueIndex:idx_dept_group;not null"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Employee struct {
	ID           uint         `json:"id" gorm:"primaryKey"`
	Name         string       `json:"name" gorm:"size:64;not null"`
	Role         EmployeeRole `json:"role" gorm:"size:32;not null"`
	Roles        string       `json:"roles" gorm:"size:255;default:''"` // comma-separated role names
	Category     string       `json:"category" gorm:"size:32;default:''"`
	GroupID      uint         `json:"groupId" gorm:"index;not null"`
	CanNight     bool         `json:"canNight"`
	Active       bool         `json:"active" gorm:"default:true"`
	SortPriority int          `json:"sortPriority" gorm:"default:0"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

func ParseRoleList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func JoinRoleList(roles []string) string {
	cleaned := make([]string, 0, len(roles))
	seen := map[string]bool{}
	for _, role := range roles {
		name := strings.TrimSpace(role)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		cleaned = append(cleaned, name)
	}
	return strings.Join(cleaned, ",")
}

func (e Employee) RoleList() []string {
	roles := ParseRoleList(e.Roles)
	if len(roles) > 0 {
		return roles
	}
	if strings.TrimSpace(string(e.Role)) == "" {
		return nil
	}
	return []string{string(e.Role)}
}

type RoleOption struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	GroupID       uint      `json:"groupId" gorm:"index;not null"`
	Name          string    `json:"name" gorm:"size:64;not null"`
	AllowLessRest bool      `json:"allowLessRest" gorm:"default:false"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type SpecialtyOption struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	GroupID   uint      `json:"groupId" gorm:"index;not null"`
	Name      string    `json:"name" gorm:"size:64;not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ShiftPost struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	GroupID     uint      `json:"groupId" gorm:"index;not null"`
	Name        string    `json:"name" gorm:"size:64;not null"`
	Required    int       `json:"required" gorm:"not null"`
	Priority    int       `json:"priority" gorm:"default:100"` // lower value scheduled first
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	Description string    `json:"description" gorm:"size:255;default:''"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PostDailyRequirement struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	GroupID   uint      `json:"groupId" gorm:"index:idx_pdr_group_month_day_post;not null"`
	Month     string    `json:"month" gorm:"size:7;index:idx_pdr_group_month_day_post;not null"`
	Day       int       `json:"day" gorm:"index:idx_pdr_group_month_day_post;not null"`
	PostName  string    `json:"postName" gorm:"size:64;index:idx_pdr_group_month_day_post;not null"`
	Required  int       `json:"required" gorm:"not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PostWeekdayRequirement struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	GroupID   uint      `json:"groupId" gorm:"index:idx_pwr_group_month_weekday_post;not null"`
	Month     string    `json:"month" gorm:"size:7;index:idx_pwr_group_month_weekday_post;not null"`
	Weekday   int       `json:"weekday" gorm:"index:idx_pwr_group_month_weekday_post;not null"` // 0=Sunday
	PostName  string    `json:"postName" gorm:"size:64;index:idx_pwr_group_month_weekday_post;not null"`
	Required  int       `json:"required" gorm:"not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SpecialRule struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	GroupID      uint      `json:"groupId" gorm:"index;not null"`
	Month        string    `json:"month" gorm:"size:7;index;default:''"` // YYYY-MM, empty means legacy recurring rule
	Name         string    `json:"name" gorm:"size:64;not null"`
	RuleType     string    `json:"ruleType" gorm:"size:16;not null"` // date | weekday
	DayOfMonth   int       `json:"dayOfMonth"`
	Weekday      int       `json:"weekday"` // 0=Sunday
	PostName     string    `json:"postName" gorm:"size:64;not null"`
	Required     int       `json:"required" gorm:"not null"`
	EmployeeID   uint      `json:"employeeId" gorm:"index"`
	EmployeeName string    `json:"employeeName" gorm:"size:64;default:''"`
	Enabled      bool      `json:"enabled" gorm:"default:true"`
	Description  string    `json:"description" gorm:"size:255;default:''"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type NightShiftRecord struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Month     string    `json:"month" gorm:"size:7;index:idx_month_day;not null"` // YYYY-MM
	Day       int       `json:"day" gorm:"index:idx_month_day;not null"`
	StaffA    string    `json:"staffA" gorm:"size:64;not null"`
	StaffB    string    `json:"staffB" gorm:"size:64;not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type MonthlyConstraint struct {
	ID           uint         `json:"id" gorm:"primaryKey"`
	GroupID      uint         `json:"groupId" gorm:"index:idx_group_month_role;not null"`
	Month        string       `json:"month" gorm:"size:7;index:idx_group_month_role;not null"`
	Role         EmployeeRole `json:"role" gorm:"size:32;index:idx_group_month_role;not null"`
	RestDaysGoal int          `json:"restDaysGoal" gorm:"not null"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type ScheduleEntry struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	GroupID    uint      `json:"groupId" gorm:"index:idx_schedule_month_day;not null"`
	Month      string    `json:"month" gorm:"size:7;index:idx_schedule_month_day;not null"`
	Day        int       `json:"day" gorm:"index:idx_schedule_month_day;not null"`
	EmployeeID uint      `json:"employeeId" gorm:"index;not null"`
	Employee   string    `json:"employee" gorm:"size:64;not null"`
	ShiftName  string    `json:"shiftName" gorm:"size:64;not null"` // post name/rest/off_after_night
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// EmployeeRestPlan 个人每月休息配置：固定休息日列表 + 浮动休息天数覆盖
// FixedDays 存储逗号分隔的日期字符串，如 "3,15,22"
// FloatDays 为 -1 表示未覆盖（使用角色默认），>=0 表示手动覆盖
type EmployeeRestPlan struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	GroupID      uint      `json:"groupId" gorm:"index:idx_erp_group_month_emp;not null"`
	Month        string    `json:"month" gorm:"size:7;index:idx_erp_group_month_emp;not null"`
	EmployeeID   uint      `json:"employeeId" gorm:"index:idx_erp_group_month_emp;not null"`
	EmployeeName string    `json:"employeeName" gorm:"size:64;not null"`
	FixedDays    string    `json:"fixedDays" gorm:"size:255;default:''"` // comma-separated day numbers
	FloatDays    int       `json:"floatDays" gorm:"default:-1"`          // -1 = use role default
	Note         string    `json:"note" gorm:"size:255;default:''"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// RestDebtRecord 休息欠账记录（最多保留12个月）
type RestDebtRecord struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	GroupID      uint      `json:"groupId" gorm:"index:idx_rdr_group_month_emp;not null"`
	Month        string    `json:"month" gorm:"size:7;index:idx_rdr_group_month_emp;not null"` // 欠账发生月
	EmployeeID   uint      `json:"employeeId" gorm:"index:idx_rdr_group_month_emp;not null"`
	EmployeeName string    `json:"employeeName" gorm:"size:64;not null"`
	DebtDays     int       `json:"debtDays" gorm:"not null"` // 欠了几天（正数）
	Reason       string    `json:"reason" gorm:"size:255;default:''"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ScheduleRemark 排班备注（每月生成后写入，导出时附加）
type ScheduleRemark struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	GroupID   uint      `json:"groupId" gorm:"index:idx_sr_group_month;not null"`
	Month     string    `json:"month" gorm:"size:7;index:idx_sr_group_month;not null"`
	Tag       string    `json:"tag" gorm:"size:32;not null"` // debt | crossmonth | makeup
	Content   string    `json:"content" gorm:"size:512;not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
