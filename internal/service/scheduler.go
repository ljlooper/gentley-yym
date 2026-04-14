package service

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"power/internal/models"

	"gorm.io/gorm"
)

type SchedulerService struct {
	db *gorm.DB
}

func NewSchedulerService(db *gorm.DB) *SchedulerService {
	return &SchedulerService{db: db}
}

type GenerateRequest struct {
	GroupID uint   `json:"groupId"`
	Month   string `json:"month"` // YYYY-MM
}

func daysInMonth(month string) (int, error) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return 0, fmt.Errorf("月份格式应为YYYY-MM")
	}
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day(), nil
}

func (s *SchedulerService) Generate(req GenerateRequest) ([]models.ScheduleEntry, error) {
	totalDays, err := daysInMonth(req.Month)
	if err != nil {
		return nil, err
	}

	var night []models.NightShiftRecord
	if err := s.db.Where("month = ?", req.Month).Find(&night).Error; err != nil {
		return nil, err
	}
	if len(night) == 0 {
		return nil, errors.New("当月夜班表不存在，请先导入夜班表")
	}

	var employees []models.Employee
	if err := s.db.Where("group_id = ? AND active = 1", req.GroupID).Order("sort_priority, id").Find(&employees).Error; err != nil {
		return nil, err
	}
	if len(employees) == 0 {
		return nil, errors.New("组内无可用员工")
	}

	var posts []models.ShiftPost
	if err := s.db.Where("group_id = ? AND enabled = 1", req.GroupID).Order("priority asc, id asc").Find(&posts).Error; err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return nil, errors.New("请先配置岗位与每日人数")
	}

	var constraints []models.MonthlyConstraint
	if err := s.db.Where("group_id = ? AND month = ?", req.GroupID, req.Month).Find(&constraints).Error; err != nil {
		return nil, err
	}
	roleRestTarget := map[models.EmployeeRole]int{
		models.RoleFormal: 8, models.RoleTech: 5, models.RoleMobile: 5,
	}
	for _, c := range constraints {
		roleRestTarget[c.Role] = c.RestDaysGoal
	}

	employeeByName := make(map[string]models.Employee)
	for _, e := range employees {
		employeeByName[e.Name] = e
	}

	// hard constraints from night shift: off day+1 and day+2 for in-group formal employees
	restByDayEmployee := map[int]map[uint]bool{}
	for _, n := range night {
		names := []string{n.StaffA, n.StaffB}
		for _, name := range names {
			emp, ok := employeeByName[name]
			if !ok {
				continue
			}
			if emp.Role != models.RoleFormal {
				continue
			}
			if !emp.CanNight {
				return nil, fmt.Errorf("%s被标记为不可夜班，但出现在夜班表中", emp.Name)
			}
			for _, d := range []int{n.Day + 1, n.Day + 2} {
				if d < 1 || d > totalDays {
					continue
				}
				if restByDayEmployee[d] == nil {
					restByDayEmployee[d] = map[uint]bool{}
				}
				restByDayEmployee[d][emp.ID] = true
			}
		}
	}

	assignment := map[int]map[uint]string{}
	restCount := map[uint]int{}
	weekendRestCount := map[uint]int{}
	workCount := map[uint]int{}

	// Pre-fill hard-rest days.
	for day, m := range restByDayEmployee {
		if assignment[day] == nil {
			assignment[day] = map[uint]string{}
		}
		for empID := range m {
			assignment[day][empID] = models.ShiftOffNight
			restCount[empID]++
		}
	}

	var specialRules []models.SpecialRule
	if err := s.db.Where("group_id = ? AND enabled = 1", req.GroupID).Find(&specialRules).Error; err != nil {
		return nil, err
	}

	for day := 1; day <= totalDays; day++ {
		if assignment[day] == nil {
			assignment[day] = map[uint]string{}
		}
		used := map[uint]bool{}
		for empID, shift := range assignment[day] {
			if shift != models.ShiftRest && shift != models.ShiftOffNight {
				used[empID] = true
			}
		}

		dailyRequired := map[string]int{}
		for _, p := range posts {
			dailyRequired[p.Name] = p.Required
		}
		for _, rule := range specialRules {
			if rule.RuleType == "date" && rule.DayOfMonth == day {
				dailyRequired[rule.PostName] = rule.Required
			}
			if rule.RuleType == "weekday" && rule.Weekday == int(weekdayOf(req.Month, day)) {
				dailyRequired[rule.PostName] = rule.Required
			}
		}

		for _, p := range posts {
			need := dailyRequired[p.Name]
			for i := 0; i < need; i++ {
				candidateID := uint(0)
				candidateScore := 1 << 30
				for _, e := range employees {
					if assignment[day][e.ID] != "" {
						continue
					}
					if used[e.ID] {
						continue
					}
					// strict rule: hard-rest cannot work
					if restByDayEmployee[day] != nil && restByDayEmployee[day][e.ID] {
						continue
					}
					score := workCount[e.ID]*10 + weekendRestCount[e.ID]*2 + restCount[e.ID]
					if score < candidateScore {
						candidateScore = score
						candidateID = e.ID
					}
				}
				if candidateID == 0 {
					return nil, fmt.Errorf("第%d天岗位[%s]无法满足人数要求", day, p.Name)
				}
				assignment[day][candidateID] = p.Name
				used[candidateID] = true
				workCount[candidateID]++
			}
		}

		// Fill remaining with rest, prioritize employees who need more rest.
		weekday := weekdayOf(req.Month, day)
		type empNeed struct {
			id   uint
			need int
		}
		needs := make([]empNeed, 0, len(employees))
		for _, e := range employees {
			target := roleRestTarget[e.Role]
			need := target - restCount[e.ID]
			needs = append(needs, empNeed{id: e.ID, need: need})
		}
		sort.SliceStable(needs, func(i, j int) bool { return needs[i].need > needs[j].need })
		for _, item := range needs {
			if assignment[day][item.id] != "" {
				continue
			}
			assignment[day][item.id] = models.ShiftRest
			restCount[item.id]++
			if weekday == time.Saturday || weekday == time.Sunday {
				weekendRestCount[item.id]++
			}
		}
	}

	// Validate minimum role rest target (soft, but required if possible).
	for _, e := range employees {
		if restCount[e.ID] < roleRestTarget[e.Role] {
			return nil, fmt.Errorf("%s休息天数不足: 当前%d, 目标%d", e.Name, restCount[e.ID], roleRestTarget[e.Role])
		}
	}

	results := make([]models.ScheduleEntry, 0, totalDays*len(employees))
	for day := 1; day <= totalDays; day++ {
		for _, e := range employees {
			shift := assignment[day][e.ID]
			if shift == "" {
				shift = models.ShiftRest
			}
			results = append(results, models.ScheduleEntry{
				GroupID:    req.GroupID,
				Month:      req.Month,
				Day:        day,
				EmployeeID: e.ID,
				Employee:   e.Name,
				ShiftName:  shift,
			})
		}
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ? AND month = ?", req.GroupID, req.Month).Delete(&models.ScheduleEntry{}).Error; err != nil {
			return err
		}
		return tx.Create(&results).Error
	}); err != nil {
		return nil, err
	}
	return results, nil
}

func weekdayOf(month string, day int) time.Weekday {
	t, err := time.Parse("2006-01-02", fmt.Sprintf("%s-%02d", month, day))
	if err != nil {
		return time.Monday
	}
	return t.Weekday()
}
