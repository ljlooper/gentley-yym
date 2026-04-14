package service

import (
	"fmt"
	"io"
	"sort"

	"power/internal/models"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ExportService struct {
	db *gorm.DB
}

func NewExportService(db *gorm.DB) *ExportService {
	return &ExportService{db: db}
}

func (s *ExportService) ExportMonth(groupID uint, month string, out io.Writer) error {
	var items []models.ScheduleEntry
	if err := s.db.Where("group_id = ? AND month = ?", groupID, month).Find(&items).Error; err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("没有可导出的排班结果")
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Day == items[j].Day {
			return items[i].Employee < items[j].Employee
		}
		return items[i].Day < items[j].Day
	})

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	f.SetSheetName(sheet, "排班")
	sheet = "排班"

	_ = f.SetCellValue(sheet, "A1", "月份")
	_ = f.SetCellValue(sheet, "B1", month)
	_ = f.SetCellValue(sheet, "A2", "日期")
	_ = f.SetCellValue(sheet, "B2", "员工")
	_ = f.SetCellValue(sheet, "C2", "班次")

	for i, item := range items {
		row := i + 3
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.Day)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.Employee)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.ShiftName)
	}

	_, err := f.WriteTo(out)
	return err
}
