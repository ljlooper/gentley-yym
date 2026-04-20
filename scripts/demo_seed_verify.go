//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"power/internal/db"
	"power/internal/models"
	"power/internal/service"
)

const (
	demoDataDir = "data/demo"
	demoMonth   = "2026-04"
	groupName   = "生化免疫组-演示"
	deptName    = "检验科"
)

type nightPair struct {
	Day int
	A   string
	B   string
}

func main() {
	if err := os.MkdirAll(filepath.Join(demoDataDir), 0o755); err != nil {
		panic(err)
	}
	database, err := db.Open(demoDataDir)
	if err != nil {
		panic(err)
	}

	if err := resetDemoData(database); err != nil {
		panic(err)
	}

	groupID, err := seedDemoData(database)
	if err != nil {
		panic(err)
	}

	scheduler := service.NewSchedulerService(database)
	items, err := scheduler.Generate(service.GenerateRequest{GroupID: groupID, Month: demoMonth})
	if err != nil {
		panic(err)
	}

	if err := verifyGeneratedSchedule(database, groupID, items); err != nil {
		panic(err)
	}

	fmt.Printf("Demo data seeded into %s\\scheduler.db\n", demoDataDir)
	fmt.Printf("Generated %d schedule entries for group %d, month %s\n", len(items), groupID, demoMonth)
	fmt.Println("Verification passed: daily post demand matched and night-rest constraints held.")
}

func resetDemoData(database interface {
	Exec(string, ...interface{}) interface{ Error() error }
}) error {
	// use raw SQL for a clean demo reset without touching the main database
	statements := []string{
		"DELETE FROM schedule_remarks",
		"DELETE FROM rest_debt_records",
		"DELETE FROM employee_rest_plans",
		"DELETE FROM schedule_entries",
		"DELETE FROM night_shift_records",
		"DELETE FROM monthly_constraints",
		"DELETE FROM special_rules",
		"DELETE FROM post_weekday_requirements",
		"DELETE FROM post_daily_requirements",
		"DELETE FROM shift_posts",
		"DELETE FROM specialty_options",
		"DELETE FROM employees",
		"DELETE FROM role_options",
		"DELETE FROM groups",
	}
	for _, stmt := range statements {
		if err := database.Exec(stmt).Error(); err != nil {
			return err
		}
	}
	return nil
}

func seedDemoData(database *db.GormDB) (uint, error) {
	panic("placeholder")
}

func verifyGeneratedSchedule(database *db.GormDB, groupID uint, items []models.ScheduleEntry) error {
	panic("placeholder")
}
