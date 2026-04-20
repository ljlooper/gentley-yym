package service

import (
	"time"

	"power/internal/models"

	"gorm.io/gorm"
)

type SchedulerStore interface {
	ListNightShifts(month string) ([]models.NightShiftRecord, error)
	ListEmployees(groupID uint) ([]models.Employee, error)
	ListPosts(groupID uint) ([]models.ShiftPost, error)
	ListRoleOptions(groupID uint) ([]models.RoleOption, error)
	ListConstraints(groupID uint, month string) ([]models.MonthlyConstraint, error)
	ListRestPlans(groupID uint, month string) ([]models.EmployeeRestPlan, error)
	ListSpecialRules(groupID uint, month string) ([]models.SpecialRule, error)
	ListPostDailyRequirements(groupID uint, month string) ([]models.PostDailyRequirement, error)
	ListPostWeekdayRequirements(groupID uint, month string) ([]models.PostWeekdayRequirement, error)
	SaveSchedule(groupID uint, month string, results []models.ScheduleEntry, remarks []models.ScheduleRemark) error
}

type CandidateScorer interface {
	Score(ctx CandidateContext) int
}

type SchedulerService struct {
	store  SchedulerStore
	scorer CandidateScorer
}

type GenerateRequest struct {
	GroupID uint   `json:"groupId"`
	Month   string `json:"month"`
}

func NewSchedulerService(db *gorm.DB) *SchedulerService {
	return NewSchedulerServiceWithStore(NewGormSchedulerStore(db))
}

func NewSchedulerServiceWithStore(store SchedulerStore) *SchedulerService {
	return &SchedulerService{
		store:  store,
		scorer: WorkloadScorer{},
	}
}

func (s *SchedulerService) Generate(req GenerateRequest) ([]models.ScheduleEntry, error) {
	planner := newSchedulePlanner(req, s.store, s.scorer)
	return planner.Generate()
}

func daysInMonth(month string) (int, error) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return 0, errInvalidMonthFormat
	}
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day(), nil
}
