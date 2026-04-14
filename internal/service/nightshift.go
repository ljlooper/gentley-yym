package service

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"power/internal/models"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type NightShiftService struct {
	db *gorm.DB
}

func NewNightShiftService(db *gorm.DB) *NightShiftService {
	return &NightShiftService{db: db}
}

func (s *NightShiftService) Import(month string, reader io.Reader) error {
	if _, err := time.Parse("2006-01", month); err != nil {
		return fmt.Errorf("月份格式应为YYYY-MM")
	}

	f, err := excelize.OpenReader(reader)
	if err != nil {
		return fmt.Errorf("读取Excel失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	if sheet == "" {
		return fmt.Errorf("Excel没有工作表")
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("读取行失败: %w", err)
	}
	if len(rows) < 2 {
		return fmt.Errorf("夜班表至少需要表头和1行数据")
	}

	var records []models.NightShiftRecord
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 2 {
			continue
		}
		dayText := strings.TrimSpace(row[0])
		if dayText == "" {
			continue
		}
		day, err := strconv.Atoi(dayText)
		if err != nil {
			return fmt.Errorf("第%d行日期不是数字", i+1)
		}
		a := strings.TrimSpace(row[1])
		b := ""
		if len(row) >= 3 {
			b = strings.TrimSpace(row[2])
		}
		if b == "" {
			parts := strings.FieldsFunc(a, func(r rune) bool {
				return r == '|' || r == '｜' || r == '/' || r == '、' || r == ',' || r == '，'
			})
			if len(parts) >= 2 {
				a = strings.TrimSpace(parts[0])
				b = strings.TrimSpace(parts[1])
			}
		}
		if a == "" || b == "" {
			return fmt.Errorf("第%d行夜班人员必须为两人", i+1)
		}
		records = append(records, models.NightShiftRecord{
			Month: month,
			Day:   day,
			StaffA: a,
			StaffB: b,
		})
	}
	if len(records) == 0 {
		return fmt.Errorf("未解析到夜班记录")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("month = ?", month).Delete(&models.NightShiftRecord{}).Error; err != nil {
			return err
		}
		return tx.Create(&records).Error
	})
}
