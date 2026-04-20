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

	// 备注 sheet
	var remarks []models.ScheduleRemark
	_ = s.db.Where("group_id = ? AND month = ?", groupID, month).Order("id asc").Find(&remarks)

	if len(remarks) > 0 {
		_, _ = f.NewSheet("备注说明")
		rsheet := "备注说明"
		_ = f.SetCellValue(rsheet, "A1", "类型")
		_ = f.SetCellValue(rsheet, "B1", "说明")
		tagLabel := map[string]string{
			"debt":       "休息欠账",
			"crossmonth": "跨月处理",
			"makeup":     "上月补偿",
		}
		for i, rem := range remarks {
			row := i + 2
			label := tagLabel[rem.Tag]
			if label == "" {
				label = rem.Tag
			}
			_ = f.SetCellValue(rsheet, fmt.Sprintf("A%d", row), label)
			_ = f.SetCellValue(rsheet, fmt.Sprintf("B%d", row), rem.Content)
		}
	}

	_, err := f.WriteTo(out)
	return err
}
