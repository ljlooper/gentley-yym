package models

import "time"

type EmployeeRole string

const (
	ShiftRest                  = "rest"
	ShiftOffNight              = "off_after_night"
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
	Category     string       `json:"category" gorm:"size:32;default:''"`
	GroupID      uint         `json:"groupId" gorm:"index;not null"`
	CanNight     bool         `json:"canNight"`
	Active       bool         `json:"active" gorm:"default:true"`
	SortPriority int          `json:"sortPriority" gorm:"default:0"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type RoleOption struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	GroupID   uint      `json:"groupId" gorm:"index;not null"`
	Name      string    `json:"name" gorm:"size:64;not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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

type SpecialRule struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	GroupID     uint      `json:"groupId" gorm:"index;not null"`
	Name        string    `json:"name" gorm:"size:64;not null"`
	RuleType    string    `json:"ruleType" gorm:"size:16;not null"` // date | weekday
	DayOfMonth  int       `json:"dayOfMonth"`
	Weekday     int       `json:"weekday"` // 0=Sunday
	PostName    string    `json:"postName" gorm:"size:64;not null"`
	Required    int       `json:"required" gorm:"not null"`
	EmployeeID  uint      `json:"employeeId" gorm:"index"`
	EmployeeName string   `json:"employeeName" gorm:"size:64;default:''"`
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	Description string    `json:"description" gorm:"size:255;default:''"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
