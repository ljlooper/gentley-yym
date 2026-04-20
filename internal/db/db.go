package db

import (
	"fmt"
	"path/filepath"

	"power/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func Open(dataDir string) (*gorm.DB, error) {
	dsn := filepath.Join(dataDir, "scheduler.db")
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := database.AutoMigrate(
		&models.Group{},
		&models.RoleOption{},
		&models.Employee{},
		&models.SpecialtyOption{},
		&models.ShiftPost{},
		&models.PostDailyRequirement{},
		&models.PostWeekdayRequirement{},
		&models.SpecialRule{},
		&models.NightShiftRecord{},
		&models.MonthlyConstraint{},
		&models.ScheduleEntry{},
		&models.EmployeeRestPlan{},
		&models.RestDebtRecord{},
		&models.ScheduleRemark{},
	); err != nil {
		return nil, fmt.Errorf("migrate tables: %w", err)
	}
	return database, nil
}
