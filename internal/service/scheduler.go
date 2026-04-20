package service

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
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
	Month   string `json:"month"`
}

func daysInMonth(month string) (int, error) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return 0, fmt.Errorf("月份格式应为 YYYY-MM")
	}
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day(), nil
}

func parseFixedDays(s string) []int {
	var days []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		d, err := strconv.Atoi(part)
		if err == nil && d > 0 {
			days = append(days, d)
		}
	}
	return days
}

func calculateCandidateScore(emp models.Employee, day int, month string, workCount, weekendRestCount, restCount, restTarget map[uint]int) int {
	restGap := restTarget[emp.ID] - restCount[emp.ID]
	if restGap < 0 {
		restGap = 0
	}
	score := workCount[emp.ID]*10 + restGap*12
	if weekday := weekdayOf(month, day); weekday == time.Saturday || weekday == time.Sunday {
		score += weekendRestCount[emp.ID] * 3
	}
	return score
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

	var roleOptions []models.RoleOption
	if err := s.db.Where("group_id = ?", req.GroupID).Order("id asc").Find(&roleOptions).Error; err != nil {
		return nil, err
	}
	roleAllowLessRest := map[string]bool{}
	for _, role := range roleOptions {
		roleAllowLessRest[role.Name] = role.AllowLessRest
	}

	var constraints []models.MonthlyConstraint
	if err := s.db.Where("group_id = ? AND month = ?", req.GroupID, req.Month).Find(&constraints).Error; err != nil {
		return nil, err
	}
	roleRestTarget := map[models.EmployeeRole]int{}
	for _, e := range employees {
		if _, ok := roleRestTarget[e.Role]; !ok {
			roleRestTarget[e.Role] = 5
		}
	}
	for _, c := range constraints {
		roleRestTarget[c.Role] = c.RestDaysGoal
	}

	roleTargetByName := map[string]int{}
	for role, target := range roleRestTarget {
		roleTargetByName[string(role)] = target
	}

	employeeByName := map[string]models.Employee{}
	employeeByID := map[uint]models.Employee{}
	for _, e := range employees {
		employeeByName[strings.TrimSpace(e.Name)] = e
		employeeByID[e.ID] = e
	}

	var restPlans []models.EmployeeRestPlan
	if err := s.db.Where("group_id = ? AND month = ?", req.GroupID, req.Month).Find(&restPlans).Error; err != nil {
		return nil, err
	}
	planByEmp := map[uint]models.EmployeeRestPlan{}
	for _, p := range restPlans {
		planByEmp[p.EmployeeID] = p
	}

	empRestTarget := map[uint]int{}
	empAllowLessRest := map[uint]bool{}
	for _, e := range employees {
		target := roleRestTarget[e.Role]
		allowLessRest := roleAllowLessRest[string(e.Role)]
		for _, roleName := range e.RoleList() {
			if t, ok := roleTargetByName[roleName]; ok && t > target {
				target = t
			}
			if roleAllowLessRest[roleName] {
				allowLessRest = true
			}
		}
		if plan, ok := planByEmp[e.ID]; ok {
			fixed := len(parseFixedDays(plan.FixedDays))
			if plan.FloatDays >= 0 {
				target = fixed + plan.FloatDays
			} else if fixed > target {
				target = fixed
			}
		}
		empRestTarget[e.ID] = target
		empAllowLessRest[e.ID] = allowLessRest
	}

	prevMonthTime, _ := time.Parse("2006-01", req.Month)
	prevMonthTime = prevMonthTime.AddDate(0, -1, 0)
	prevMonth := prevMonthTime.Format("2006-01")
	prevTotalDays := time.Date(prevMonthTime.Year(), prevMonthTime.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()

	restByDayEmployee := map[int]map[uint]bool{}
	crossMonthRemarks := []string{}

	for _, n := range night {
		for _, name := range []string{n.StaffA, n.StaffB} {
			emp, ok := employeeByName[strings.TrimSpace(name)]
			if !ok {
				return nil, fmt.Errorf("夜班表中的人员[%s]未在当前小组人员名单中找到，请按姓名保持一致", name)
			}
			if !emp.CanNight {
				return nil, fmt.Errorf("%s被标记为不可夜班，但出现在夜班表中", emp.Name)
			}
			for _, d := range []int{n.Day + 1, n.Day + 2} {
				if d <= totalDays {
					if restByDayEmployee[d] == nil {
						restByDayEmployee[d] = map[uint]bool{}
					}
					restByDayEmployee[d][emp.ID] = true
					continue
				}
				overflowDay := d - totalDays
				crossMonthRemarks = append(crossMonthRemarks,
					fmt.Sprintf("%s：本月第%d天夜班后的休息顺延至下月第%d天，下月统计休息时会自动计入", emp.Name, n.Day, overflowDay))
			}
		}
	}

	var prevNight []models.NightShiftRecord
	if err := s.db.Where("month = ?", prevMonth).Find(&prevNight).Error; err != nil {
		return nil, err
	}
	for _, n := range prevNight {
		for _, name := range []string{n.StaffA, n.StaffB} {
			emp, ok := employeeByName[strings.TrimSpace(name)]
			if !ok {
				continue
			}
			for _, d := range []int{n.Day + 1, n.Day + 2} {
				if d <= prevTotalDays {
					continue
				}
				overflowDay := d - prevTotalDays
				if overflowDay < 1 || overflowDay > totalDays {
					continue
				}
				if restByDayEmployee[overflowDay] == nil {
					restByDayEmployee[overflowDay] = map[uint]bool{}
				}
				restByDayEmployee[overflowDay][emp.ID] = true
				crossMonthRemarks = append(crossMonthRemarks,
					fmt.Sprintf("%s：上月（%s）第%d天夜班后的休息顺延至本月第%d天", emp.Name, prevMonth, n.Day, overflowDay))
			}
		}
	}

	fixedRestByDayEmp := map[int]map[uint]bool{}
	for _, plan := range restPlans {
		emp, ok := employeeByID[plan.EmployeeID]
		if !ok {
			continue
		}
		for _, d := range parseFixedDays(plan.FixedDays) {
			if d < 1 || d > totalDays {
				continue
			}
			if fixedRestByDayEmp[d] == nil {
				fixedRestByDayEmp[d] = map[uint]bool{}
			}
			fixedRestByDayEmp[d][emp.ID] = true
		}
	}

	assignment := map[int]map[uint]string{}
	restCount := map[uint]int{}
	weekendRestCount := map[uint]int{}
	workCount := map[uint]int{}

	for day, m := range restByDayEmployee {
		if assignment[day] == nil {
			assignment[day] = map[uint]string{}
		}
		weekday := weekdayOf(req.Month, day)
		for empID := range m {
			assignment[day][empID] = models.ShiftOffNight
			restCount[empID]++
			if weekday == time.Saturday || weekday == time.Sunday {
				weekendRestCount[empID]++
			}
		}
	}

	for day, m := range fixedRestByDayEmp {
		if assignment[day] == nil {
			assignment[day] = map[uint]string{}
		}
		weekday := weekdayOf(req.Month, day)
		for empID := range m {
			if assignment[day][empID] != "" {
				continue
			}
			assignment[day][empID] = models.ShiftRest
			restCount[empID]++
			if weekday == time.Saturday || weekday == time.Sunday {
				weekendRestCount[empID]++
			}
		}
	}

	var specialRules []models.SpecialRule
	if err := s.db.Where("group_id = ? AND enabled = 1 AND (month = ? OR month = '')", req.GroupID, req.Month).Find(&specialRules).Error; err != nil {
		return nil, err
	}
	var dailyRequirements []models.PostDailyRequirement
	if err := s.db.Where("group_id = ? AND month = ?", req.GroupID, req.Month).Find(&dailyRequirements).Error; err != nil {
		return nil, err
	}
	var weekdayRequirements []models.PostWeekdayRequirement
	if err := s.db.Where("group_id = ? AND month = ?", req.GroupID, req.Month).Find(&weekdayRequirements).Error; err != nil {
		return nil, err
	}

	dailyRequiredOverrides := map[int]map[string]int{}
	for _, item := range dailyRequirements {
		if item.Day < 1 || item.Day > totalDays {
			continue
		}
		if dailyRequiredOverrides[item.Day] == nil {
			dailyRequiredOverrides[item.Day] = map[string]int{}
		}
		dailyRequiredOverrides[item.Day][item.PostName] = item.Required
	}

	weekdayRequiredOverrides := map[int]map[string]int{}
	for _, item := range weekdayRequirements {
		if item.Weekday < 0 || item.Weekday > 6 {
			continue
		}
		if weekdayRequiredOverrides[item.Weekday] == nil {
			weekdayRequiredOverrides[item.Weekday] = map[string]int{}
		}
		weekdayRequiredOverrides[item.Weekday][item.PostName] = item.Required
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
		weekdayIdx := int(weekdayOf(req.Month, day))
		if overrideByPost := weekdayRequiredOverrides[weekdayIdx]; overrideByPost != nil {
			for postName, required := range overrideByPost {
				dailyRequired[postName] = required
			}
		}
		if overrideByPost := dailyRequiredOverrides[day]; overrideByPost != nil {
			for postName, required := range overrideByPost {
				dailyRequired[postName] = required
			}
		}

		for _, rule := range specialRules {
			ruleMatched := false
			if rule.RuleType == "date" && rule.DayOfMonth == day {
				ruleMatched = true
			}
			if rule.RuleType == "weekday" && rule.Weekday == int(weekdayOf(req.Month, day)) {
				ruleMatched = true
			}
			if !ruleMatched {
				continue
			}
			if rule.EmployeeID != 0 {
				emp, ok := employeeByName[strings.TrimSpace(rule.EmployeeName)]
				if !ok || emp.ID != rule.EmployeeID {
					return nil, fmt.Errorf("特殊规则中的指定人员[%s]已不在当前小组", rule.EmployeeName)
				}
				if assignment[day][rule.EmployeeID] != "" {
					return nil, fmt.Errorf("第%d天指定人员[%s]已被占用，无法重复安排", day, rule.EmployeeName)
				}
				if restByDayEmployee[day] != nil && restByDayEmployee[day][rule.EmployeeID] {
					return nil, fmt.Errorf("第%d天指定人员[%s]处于夜班后休息，无法安排固定岗位", day, rule.EmployeeName)
				}
				assignment[day][rule.EmployeeID] = rule.PostName
				used[rule.EmployeeID] = true
				workCount[rule.EmployeeID]++
				if current := dailyRequired[rule.PostName]; current > 0 {
					dailyRequired[rule.PostName] = current - 1
				}
				continue
			}
			dailyRequired[rule.PostName] = rule.Required
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
					if restByDayEmployee[day] != nil && restByDayEmployee[day][e.ID] {
						continue
					}
					if fixedRestByDayEmp[day] != nil && fixedRestByDayEmp[day][e.ID] {
						continue
					}
					score := calculateCandidateScore(e, day, req.Month, workCount, weekendRestCount, restCount, empRestTarget)
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

		weekday := weekdayOf(req.Month, day)
		type empNeed struct {
			id   uint
			need int
		}
		needs := make([]empNeed, 0, len(employees))
		for _, e := range employees {
			target := empRestTarget[e.ID]
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

	remarks := []models.ScheduleRemark{}
	for _, e := range employees {
		actual := restCount[e.ID]
		target := empRestTarget[e.ID]
		if actual < target && !empAllowLessRest[e.ID] {
			return nil, fmt.Errorf("%s所属角色本月必须休满%d天，但当前排班只能休%d天，请调整班种人数、固定休息日或夜班安排后重试", e.Name, target, actual)
		}
	}
	for _, r := range crossMonthRemarks {
		remarks = append(remarks, models.ScheduleRemark{
			GroupID: req.GroupID,
			Month:   req.Month,
			Tag:     "crossmonth",
			Content: r,
		})
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
		if err := tx.Where("group_id = ? AND month = ?", req.GroupID, req.Month).Delete(&models.ScheduleRemark{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ? AND month = ?", req.GroupID, req.Month).Delete(&models.RestDebtRecord{}).Error; err != nil {
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
