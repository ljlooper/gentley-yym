package main

import (
	"fmt"
	"path/filepath"

	"power/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	dbPath := filepath.Join(".", "data", "scheduler.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("open db: %w", err))
	}

	var groups []models.Group
	if err := db.Find(&groups).Error; err != nil {
		panic(fmt.Errorf("list groups: %w", err))
	}

	for _, g := range groups {
		if err := ensureRole(db, g.ID, "正式员工"); err != nil {
			panic(err)
		}
		if err := ensureSpecialty(db, g.ID, "生化"); err != nil {
			panic(err)
		}
		if err := normalizeEmployees(db, g.ID); err != nil {
			panic(err)
		}
	}

	fmt.Printf("done: updated %d groups\n", len(groups))
}

func ensureRole(db *gorm.DB, groupID uint, name string) error {
	var count int64
	if err := db.Model(&models.RoleOption{}).Where("group_id = ? AND name = ?", groupID, name).Count(&count).Error; err != nil {
		return fmt.Errorf("check role: %w", err)
	}
	if count > 0 {
		return nil
	}
	return db.Create(&models.RoleOption{GroupID: groupID, Name: name}).Error
}

func ensureSpecialty(db *gorm.DB, groupID uint, name string) error {
	var count int64
	if err := db.Model(&models.SpecialtyOption{}).Where("group_id = ? AND name = ?", groupID, name).Count(&count).Error; err != nil {
		return fmt.Errorf("check specialty: %w", err)
	}
	if count > 0 {
		return nil
	}
	return db.Create(&models.SpecialtyOption{GroupID: groupID, Name: name}).Error
}

func normalizeEmployees(db *gorm.DB, groupID uint) error {
	return db.Model(&models.Employee{}).
		Where("group_id = ?", groupID).
		Updates(map[string]interface{}{
			"role":     "正式员工",
			"roles":    "正式员工",
			"category": "生化",
		}).Error
}
