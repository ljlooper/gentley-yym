package service

import (
	"power/internal/models"

	"gorm.io/gorm"
)

type GormSchedulerStore struct {
	db *gorm.DB
}

func NewGormSchedulerStore(db *gorm.DB) *GormSchedulerStore {
	return &GormSchedulerStore{db: db}
}

func (s *GormSchedulerStore) ListNightShifts(month string) ([]models.NightShiftRecord, error) {
	var items []models.NightShiftRecord
	err := s.db.Where("month = ?", month).Order("day asc, id asc").Find(&items).Error
	return items, err
}

func (s *GormSchedulerStore) ListEmployees(groupID uint) ([]models.Employee, error) {
	var items []models.Employee
	err := s.db.Where("group_id = ? AND active = 1", groupID).Order("sort_priority, id").Find(&items).Error
	return items, err
}

func (s *GormSchedulerStore) ListPosts(groupID uint) ([]models.ShiftPost, error) {
	var items []models.ShiftPost
	err := s.db.Where("group_id = ? AND enabled = 1", groupID).Order("priority asc, id asc").Find(&items).Error
	return items, err
}

func (s *GormSchedulerStore) ListRoleOptions(groupID uint) ([]models.RoleOption, error) {
	var items []models.RoleOption
	err := s.db.Where("group_id = ?", groupID).Order("id asc").Find(&items).Error
	return items, err
}

func (s *GormSchedulerStore) ListConstraints(groupID uint, month string) ([]models.MonthlyConstraint, error) {
	var items []models.MonthlyConstraint
	err := s.db.Where("group_id = ? AND month = ?", groupID, month).Find(&items).Error
	return items, err
}

func (s *GormSchedulerStore) ListRestPlans(groupID uint, month string) ([]models.EmployeeRestPlan, error) {
	var items []models.EmployeeRestPlan
	err := s.db.Where("group_id = ? AND month = ?", groupID, month).Find(&items).Error
	return items, err
}

func (s *GormSchedulerStore) ListSpecialRules(groupID uint, month string) ([]models.SpecialRule, error) {
	var items []models.SpecialRule
	err := s.db.Where("group_id = ? AND enabled = 1 AND (month = ? OR month = '')", groupID, month).Find(&items).Error
	return items, err
}

func (s *GormSchedulerStore) ListPostDailyRequirements(groupID uint, month string) ([]models.PostDailyRequirement, error) {
	var items []models.PostDailyRequirement
	err := s.db.Where("group_id = ? AND month = ?", groupID, month).Find(&items).Error
	return items, err
}

func (s *GormSchedulerStore) ListPostWeekdayRequirements(groupID uint, month string) ([]models.PostWeekdayRequirement, error) {
	var items []models.PostWeekdayRequirement
	err := s.db.Where("group_id = ? AND month = ?", groupID, month).Find(&items).Error
	return items, err
}

func (s *GormSchedulerStore) SaveSchedule(groupID uint, month string, results []models.ScheduleEntry, remarks []models.ScheduleRemark) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ? AND month = ?", groupID, month).Delete(&models.ScheduleEntry{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ? AND month = ?", groupID, month).Delete(&models.ScheduleRemark{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ? AND month = ?", groupID, month).Delete(&models.RestDebtRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&results).Error; err != nil {
			return err
		}
		if len(remarks) > 0 {
			if err := tx.Create(&remarks).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
