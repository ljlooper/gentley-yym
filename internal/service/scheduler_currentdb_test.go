package service

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"power/internal/db"
	"power/internal/models"

	"gorm.io/gorm"
)

func TestCurrentDatabaseAprilNightShiftSheetAudit(t *testing.T) {
	sourceDB := filepath.Join("..", "..", "data", "scheduler.db")
	if _, err := os.Stat(sourceDB); err != nil {
		t.Skip("current database not found")
	}

	tempDir := t.TempDir()
	targetDB := filepath.Join(tempDir, "scheduler.db")

	raw, err := os.ReadFile(sourceDB)
	if err != nil {
		t.Fatalf("read source db: %v", err)
	}
	if err := os.WriteFile(targetDB, raw, 0o644); err != nil {
		t.Fatalf("write temp db: %v", err)
	}

	database, err := db.Open(tempDir)
	if err != nil {
		t.Fatalf("open copied db: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("open sql db handle: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	records := aprilNightShiftSheetRecords()
	unknown, ineligible, err := auditNightShiftSheet(database, 1, records)
	if err != nil {
		t.Fatalf("audit night shift sheet: %v", err)
	}

	expectedUnknown := []string{
		"\u4ed8\u5c0f\u6e05",
		"\u4e07\u5c0f\u5a1f",
		"\u5eb7\u7d20\u6021",
		"\u5b8b\u79cb\u82b3",
		"\u66fe\u4fca\u5347",
		"\u6797\u826f\u946b",
		"\u738b\u957f\u6625",
		"\u9093\u8fbe\u6210",
		"\u90ed\u5e73",
		"\u8521\u660e",
		"\u9ea6\u9759\u83b9",
	}
	expectedIneligible := []string{"\u6797\u6b22"}

	sort.Strings(expectedUnknown)
	sort.Strings(expectedIneligible)

	if !equalStringSlices(unknown, expectedUnknown) {
		t.Fatalf("unexpected unknown names: got %v want %v", unknown, expectedUnknown)
	}
	if !equalStringSlices(ineligible, expectedIneligible) {
		t.Fatalf("unexpected ineligible names: got %v want %v", ineligible, expectedIneligible)
	}

	if err := database.Where("month = ?", "2026-04").Delete(&models.NightShiftRecord{}).Error; err != nil {
		t.Fatalf("clear temp night shifts: %v", err)
	}
	if err := database.Create(&records).Error; err != nil {
		t.Fatalf("insert temp night shifts: %v", err)
	}

	scheduler := NewSchedulerService(database)
	_, err = scheduler.Generate(GenerateRequest{GroupID: 1, Month: "2026-04"})
	if err == nil {
		t.Fatal("expected schedule generation to fail with current db and provided night shift sheet")
	}
	if !strings.Contains(err.Error(), "\u5eb7\u7d20\u6021") {
		t.Fatalf("expected first failure to mention 康素怡, got %v", err)
	}
}

func auditNightShiftSheet(database *gorm.DB, groupID uint, records []models.NightShiftRecord) ([]string, []string, error) {
	var employees []models.Employee
	if err := database.Where("group_id = ? AND active = 1", groupID).Find(&employees).Error; err != nil {
		return nil, nil, err
	}

	knownEmployees := map[string]models.Employee{}
	for _, employee := range employees {
		knownEmployees[strings.TrimSpace(employee.Name)] = employee
	}

	unknownSet := map[string]bool{}
	ineligibleSet := map[string]bool{}

	for _, record := range records {
		for _, name := range []string{record.StaffA, record.StaffB} {
			normalized := strings.TrimSpace(name)
			employee, ok := knownEmployees[normalized]
			if !ok {
				unknownSet[normalized] = true
				continue
			}
			if !employee.CanNight {
				ineligibleSet[normalized] = true
			}
		}
	}

	unknown := setKeys(unknownSet)
	ineligible := setKeys(ineligibleSet)
	sort.Strings(unknown)
	sort.Strings(ineligible)

	return unknown, ineligible, nil
}

func setKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func aprilNightShiftSheetRecords() []models.NightShiftRecord {
	rows := []struct {
		day int
		a   string
		b   string
	}{
		{1, "\u59da\u96e8\u6726", "\u5eb7\u7d20\u6021"},
		{2, "\u5b8b\u79cb\u82b3", "\u8521\u660e"},
		{3, "\u4f55\u590f\u683e", "\u66fe\u4fca\u5347"},
		{4, "\u6797\u826f\u946b", "\u9093\u8fbe\u6210"},
		{5, "\u4ed8\u5c0f\u6e05", "\u674e\u7ae0\u7fbd"},
		{6, "\u4e07\u5c0f\u5a1f", "\u5f20\u94f6\u971e"},
		{7, "\u5218\u7ec3\u658c", "\u738b\u957f\u6625"},
		{8, "\u90ed\u5e73", "\u9ea6\u9759\u83b9"},
		{9, "\u6797\u6b22", "\u5eb7\u7d20\u6021"},
		{10, "\u59da\u96e8\u6726", "\u5b8b\u79cb\u82b3"},
		{11, "\u8521\u660e", "\u4f55\u590f\u683e"},
		{12, "\u66fe\u4fca\u5347", "\u6797\u826f\u946b"},
		{13, "\u9093\u8fbe\u6210", "\u4ed8\u5c0f\u6e05"},
		{14, "\u674e\u7ae0\u7fbd", "\u4e07\u5c0f\u5a1f"},
		{15, "\u5f20\u94f6\u971e", "\u738b\u957f\u6625"},
		{16, "\u5218\u7ec3\u658c", "\u90ed\u5e73"},
		{17, "\u9ea6\u9759\u83b9", "\u6797\u6b22"},
		{18, "\u59da\u96e8\u6726", "\u5eb7\u7d20\u6021"},
		{19, "\u5b8b\u79cb\u82b3", "\u8521\u660e"},
		{20, "\u4f55\u590f\u683e", "\u66fe\u4fca\u5347"},
		{21, "\u6797\u826f\u946b", "\u9093\u8fbe\u6210"},
		{22, "\u4ed8\u5c0f\u6e05", "\u674e\u7ae0\u7fbd"},
		{23, "\u4e07\u5c0f\u5a1f", "\u5f20\u94f6\u971e"},
		{24, "\u5218\u7ec3\u658c", "\u738b\u957f\u6625"},
		{25, "\u90ed\u5e73", "\u9ea6\u9759\u83b9"},
		{26, "\u6797\u6b22", "\u5eb7\u7d20\u6021"},
		{27, "\u59da\u96e8\u6726", "\u5b8b\u79cb\u82b3"},
		{28, "\u8521\u660e", "\u4f55\u590f\u683e"},
		{29, "\u66fe\u4fca\u5347", "\u6797\u826f\u946b"},
		{30, "\u9093\u8fbe\u6210", "\u4ed8\u5c0f\u6e05"},
	}

	records := make([]models.NightShiftRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, models.NightShiftRecord{
			Month:  "2026-04",
			Day:    row.day,
			StaffA: row.a,
			StaffB: row.b,
		})
	}
	return records
}
