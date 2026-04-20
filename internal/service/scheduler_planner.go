package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"power/internal/models"
)

type schedulePlanner struct {
	req    GenerateRequest
	store  SchedulerStore
	scorer CandidateScorer

	totalDays int

	night       []models.NightShiftRecord
	prevNight   []models.NightShiftRecord
	employees   []models.Employee
	posts       []models.ShiftPost
	roleOptions []models.RoleOption
	constraints []models.MonthlyConstraint
	restPlans   []models.EmployeeRestPlan
	specials    []models.SpecialRule
	dailyReqs   []models.PostDailyRequirement
	weeklyReqs  []models.PostWeekdayRequirement

	roleAllowLessRest      map[string]bool
	roleTargetByName       map[string]int
	employeeByName         map[string]models.Employee
	employeeByID           map[uint]models.Employee
	planByEmployee         map[uint]models.EmployeeRestPlan
	empRestTarget          map[uint]int
	empAllowLessRest       map[uint]bool
	restByDayEmployee      map[int]map[uint]bool
	fixedRestByDayEmployee map[int]map[uint]bool
	dailyOverrides         map[int]map[string]int
	weekdayOverrides       map[int]map[string]int

	assignment       map[int]map[uint]string
	restCount        map[uint]int
	weekendRestCount map[uint]int
	workCount        map[uint]int
	crossMonthNotes  []string
}

func newSchedulePlanner(req GenerateRequest, store SchedulerStore, scorer CandidateScorer) *schedulePlanner {
	return &schedulePlanner{
		req:                    req,
		store:                  store,
		scorer:                 scorer,
		roleAllowLessRest:      map[string]bool{},
		roleTargetByName:       map[string]int{},
		employeeByName:         map[string]models.Employee{},
		employeeByID:           map[uint]models.Employee{},
		planByEmployee:         map[uint]models.EmployeeRestPlan{},
		empRestTarget:          map[uint]int{},
		empAllowLessRest:       map[uint]bool{},
		restByDayEmployee:      map[int]map[uint]bool{},
		fixedRestByDayEmployee: map[int]map[uint]bool{},
		dailyOverrides:         map[int]map[string]int{},
		weekdayOverrides:       map[int]map[string]int{},
		assignment:             map[int]map[uint]string{},
		restCount:              map[uint]int{},
		weekendRestCount:       map[uint]int{},
		workCount:              map[uint]int{},
		crossMonthNotes:        []string{},
	}
}

func (p *schedulePlanner) Generate() ([]models.ScheduleEntry, error) {
	if err := p.load(); err != nil {
		return nil, err
	}
	if err := p.buildTargets(); err != nil {
		return nil, err
	}
	if err := p.buildForcedRestDays(); err != nil {
		return nil, err
	}

	p.applyForcedRestAssignments()

	if err := p.assignDays(); err != nil {
		return nil, err
	}
	if err := p.validateRestTargets(); err != nil {
		return nil, err
	}

	results := p.buildResults()
	remarks := p.buildRemarks()
	if err := p.store.SaveSchedule(p.req.GroupID, p.req.Month, results, remarks); err != nil {
		return nil, err
	}
	return results, nil
}

func (p *schedulePlanner) load() error {
	totalDays, err := daysInMonth(p.req.Month)
	if err != nil {
		return err
	}
	p.totalDays = totalDays

	if p.night, err = p.store.ListNightShifts(p.req.Month); err != nil {
		return err
	}
	if len(p.night) == 0 {
		return errNightShiftRequired
	}

	if p.employees, err = p.store.ListEmployees(p.req.GroupID); err != nil {
		return err
	}
	if len(p.employees) == 0 {
		return errNoActiveEmployees
	}

	if p.posts, err = p.store.ListPosts(p.req.GroupID); err != nil {
		return err
	}
	if len(p.posts) == 0 {
		return errNoEnabledPosts
	}

	if p.roleOptions, err = p.store.ListRoleOptions(p.req.GroupID); err != nil {
		return err
	}
	if p.constraints, err = p.store.ListConstraints(p.req.GroupID, p.req.Month); err != nil {
		return err
	}
	if p.restPlans, err = p.store.ListRestPlans(p.req.GroupID, p.req.Month); err != nil {
		return err
	}
	if p.specials, err = p.store.ListSpecialRules(p.req.GroupID, p.req.Month); err != nil {
		return err
	}
	if p.dailyReqs, err = p.store.ListPostDailyRequirements(p.req.GroupID, p.req.Month); err != nil {
		return err
	}
	if p.weeklyReqs, err = p.store.ListPostWeekdayRequirements(p.req.GroupID, p.req.Month); err != nil {
		return err
	}

	prevMonth, _, err := previousMonth(p.req.Month)
	if err != nil {
		return err
	}
	if p.prevNight, err = p.store.ListNightShifts(prevMonth); err != nil {
		return err
	}

	p.indexEmployees()
	p.indexRoleOptions()
	p.indexRestPlans()
	p.indexOverrides()

	return nil
}

func (p *schedulePlanner) indexEmployees() {
	for _, employee := range p.employees {
		p.employeeByName[strings.TrimSpace(employee.Name)] = employee
		p.employeeByID[employee.ID] = employee
	}
}

func (p *schedulePlanner) indexRoleOptions() {
	for _, role := range p.roleOptions {
		p.roleAllowLessRest[role.Name] = role.AllowLessRest
		p.roleTargetByName[role.Name] = 5
	}
}

func (p *schedulePlanner) indexRestPlans() {
	for _, plan := range p.restPlans {
		p.planByEmployee[plan.EmployeeID] = plan
	}
}

func (p *schedulePlanner) indexOverrides() {
	for _, item := range p.dailyReqs {
		if item.Day < 1 || item.Day > p.totalDays {
			continue
		}
		if p.dailyOverrides[item.Day] == nil {
			p.dailyOverrides[item.Day] = map[string]int{}
		}
		p.dailyOverrides[item.Day][item.PostName] = item.Required
	}

	for _, item := range p.weeklyReqs {
		if item.Weekday < 0 || item.Weekday > 6 {
			continue
		}
		if p.weekdayOverrides[item.Weekday] == nil {
			p.weekdayOverrides[item.Weekday] = map[string]int{}
		}
		p.weekdayOverrides[item.Weekday][item.PostName] = item.Required
	}
}

func (p *schedulePlanner) buildTargets() error {
	for _, employee := range p.employees {
		if _, ok := p.roleTargetByName[string(employee.Role)]; !ok {
			p.roleTargetByName[string(employee.Role)] = 5
		}
		for _, roleName := range employee.RoleList() {
			if _, ok := p.roleTargetByName[roleName]; !ok {
				p.roleTargetByName[roleName] = 5
			}
		}
	}

	for _, constraint := range p.constraints {
		p.roleTargetByName[string(constraint.Role)] = constraint.RestDaysGoal
	}

	for _, employee := range p.employees {
		target := p.roleTargetByName[string(employee.Role)]
		allowLessRest := p.roleAllowLessRest[string(employee.Role)]

		for _, roleName := range employee.RoleList() {
			if roleTarget := p.roleTargetByName[roleName]; roleTarget > target {
				target = roleTarget
			}
			if p.roleAllowLessRest[roleName] {
				allowLessRest = true
			}
		}

		if plan, ok := p.planByEmployee[employee.ID]; ok {
			fixedRestDays := len(parseFixedDays(plan.FixedDays))
			switch {
			case plan.FloatDays >= 0:
				target = fixedRestDays + plan.FloatDays
			case fixedRestDays > target:
				target = fixedRestDays
			}
		}

		p.empRestTarget[employee.ID] = target
		p.empAllowLessRest[employee.ID] = allowLessRest
	}

	return nil
}

func (p *schedulePlanner) buildForcedRestDays() error {
	if err := p.markCurrentMonthNightRests(); err != nil {
		return err
	}
	p.markPreviousMonthCarryOver()
	p.markFixedRestDays()
	return nil
}

func (p *schedulePlanner) markCurrentMonthNightRests() error {
	for _, record := range p.night {
		for _, name := range []string{record.StaffA, record.StaffB} {
			employee, ok := p.employeeByName[strings.TrimSpace(name)]
			if !ok {
				// The night shift sheet is department-wide. We only apply constraints
				// to employees that belong to the current group.
				continue
			}
			if !employee.CanNight {
				return fmt.Errorf("employee %q is marked as not eligible for night shift", employee.Name)
			}
			for _, restDay := range []int{record.Day + 1, record.Day + 2} {
				if restDay <= p.totalDays {
					p.markRestDay(p.restByDayEmployee, restDay, employee.ID)
					continue
				}
				overflowDay := restDay - p.totalDays
				p.crossMonthNotes = append(
					p.crossMonthNotes,
					fmt.Sprintf("%s: night shift on day %d extends rest to next month day %d", employee.Name, record.Day, overflowDay),
				)
			}
		}
	}
	return nil
}

func (p *schedulePlanner) markPreviousMonthCarryOver() {
	_, prevTotalDays, err := previousMonth(p.req.Month)
	if err != nil {
		return
	}

	for _, record := range p.prevNight {
		for _, name := range []string{record.StaffA, record.StaffB} {
			employee, ok := p.employeeByName[strings.TrimSpace(name)]
			if !ok {
				continue
			}

			for _, restDay := range []int{record.Day + 1, record.Day + 2} {
				if restDay <= prevTotalDays {
					continue
				}
				overflowDay := restDay - prevTotalDays
				if overflowDay < 1 || overflowDay > p.totalDays {
					continue
				}

				p.markRestDay(p.restByDayEmployee, overflowDay, employee.ID)
				p.crossMonthNotes = append(
					p.crossMonthNotes,
					fmt.Sprintf("%s: previous month night shift on day %d extends rest to current month day %d", employee.Name, record.Day, overflowDay),
				)
			}
		}
	}
}

func (p *schedulePlanner) markFixedRestDays() {
	for _, plan := range p.restPlans {
		if _, ok := p.employeeByID[plan.EmployeeID]; !ok {
			continue
		}
		for _, day := range parseFixedDays(plan.FixedDays) {
			if day < 1 || day > p.totalDays {
				continue
			}
			p.markRestDay(p.fixedRestByDayEmployee, day, plan.EmployeeID)
		}
	}
}

func (p *schedulePlanner) markRestDay(target map[int]map[uint]bool, day int, employeeID uint) {
	if target[day] == nil {
		target[day] = map[uint]bool{}
	}
	target[day][employeeID] = true
}

func (p *schedulePlanner) applyForcedRestAssignments() {
	for day, employees := range p.restByDayEmployee {
		p.ensureDayAssignment(day)
		weekday := weekdayOf(p.req.Month, day)
		for employeeID := range employees {
			p.assignment[day][employeeID] = models.ShiftOffNight
			p.restCount[employeeID]++
			if weekday == time.Saturday || weekday == time.Sunday {
				p.weekendRestCount[employeeID]++
			}
		}
	}

	for day, employees := range p.fixedRestByDayEmployee {
		p.ensureDayAssignment(day)
		weekday := weekdayOf(p.req.Month, day)
		for employeeID := range employees {
			if p.assignment[day][employeeID] != "" {
				continue
			}
			p.assignment[day][employeeID] = models.ShiftRest
			p.restCount[employeeID]++
			if weekday == time.Saturday || weekday == time.Sunday {
				p.weekendRestCount[employeeID]++
			}
		}
	}
}

func (p *schedulePlanner) assignDays() error {
	for day := 1; day <= p.totalDays; day++ {
		p.ensureDayAssignment(day)

		used := p.currentUsedEmployees(day)
		required := p.resolveDailyRequirements(day)

		if err := p.applySpecialRules(day, required, used); err != nil {
			return err
		}
		if err := p.assignPosts(day, required, used); err != nil {
			return err
		}
		p.fillRemainingWithRest(day)
	}
	return nil
}

func (p *schedulePlanner) ensureDayAssignment(day int) {
	if p.assignment[day] == nil {
		p.assignment[day] = map[uint]string{}
	}
}

func (p *schedulePlanner) currentUsedEmployees(day int) map[uint]bool {
	used := map[uint]bool{}
	for employeeID, shift := range p.assignment[day] {
		if shift != models.ShiftRest && shift != models.ShiftOffNight {
			used[employeeID] = true
		}
	}
	return used
}

func (p *schedulePlanner) resolveDailyRequirements(day int) map[string]int {
	required := map[string]int{}
	for _, post := range p.posts {
		required[post.Name] = post.Required
	}

	weekdayIndex := int(weekdayOf(p.req.Month, day))
	if overrides := p.weekdayOverrides[weekdayIndex]; overrides != nil {
		for postName, count := range overrides {
			required[postName] = count
		}
	}
	if overrides := p.dailyOverrides[day]; overrides != nil {
		for postName, count := range overrides {
			required[postName] = count
		}
	}

	return required
}

func (p *schedulePlanner) applySpecialRules(day int, required map[string]int, used map[uint]bool) error {
	for _, rule := range p.specials {
		if !p.ruleMatchesDay(rule, day) {
			continue
		}

		if rule.EmployeeID == 0 {
			required[rule.PostName] = rule.Required
			continue
		}

		employee, ok := p.employeeByName[strings.TrimSpace(rule.EmployeeName)]
		if !ok || employee.ID != rule.EmployeeID {
			return fmt.Errorf("special rule employee %q is no longer available in current group", rule.EmployeeName)
		}
		if p.assignment[day][rule.EmployeeID] != "" {
			return fmt.Errorf("employee %q is already assigned on day %d", rule.EmployeeName, day)
		}
		if p.restByDayEmployee[day] != nil && p.restByDayEmployee[day][rule.EmployeeID] {
			return fmt.Errorf("employee %q is resting after night shift on day %d", rule.EmployeeName, day)
		}

		p.assignment[day][rule.EmployeeID] = rule.PostName
		used[rule.EmployeeID] = true
		p.workCount[rule.EmployeeID]++
		if current := required[rule.PostName]; current > 0 {
			required[rule.PostName] = current - 1
		}
	}

	return nil
}

func (p *schedulePlanner) ruleMatchesDay(rule models.SpecialRule, day int) bool {
	if rule.RuleType == "date" && rule.DayOfMonth == day {
		return true
	}
	if rule.RuleType == "weekday" && rule.Weekday == int(weekdayOf(p.req.Month, day)) {
		return true
	}
	return false
}

func (p *schedulePlanner) assignPosts(day int, required map[string]int, used map[uint]bool) error {
	for _, post := range p.posts {
		need := required[post.Name]
		for i := 0; i < need; i++ {
			employeeID, err := p.pickCandidate(day, used)
			if err != nil {
				return fmt.Errorf("day %d post %q: %w", day, post.Name, err)
			}

			p.assignment[day][employeeID] = post.Name
			used[employeeID] = true
			p.workCount[employeeID]++
		}
	}
	return nil
}

func (p *schedulePlanner) pickCandidate(day int, used map[uint]bool) (uint, error) {
	candidateID := uint(0)
	candidateScore := int(^uint(0) >> 1)

	for _, employee := range p.employees {
		if p.assignment[day][employee.ID] != "" {
			continue
		}
		if used[employee.ID] {
			continue
		}
		if p.restByDayEmployee[day] != nil && p.restByDayEmployee[day][employee.ID] {
			continue
		}
		if p.fixedRestByDayEmployee[day] != nil && p.fixedRestByDayEmployee[day][employee.ID] {
			continue
		}

		score := p.scorer.Score(CandidateContext{
			Employee:         employee,
			Day:              day,
			Month:            p.req.Month,
			WorkCount:        p.workCount,
			WeekendRestCount: p.weekendRestCount,
			RestCount:        p.restCount,
			RestTarget:       p.empRestTarget,
		})
		if score < candidateScore {
			candidateScore = score
			candidateID = employee.ID
		}
	}

	if candidateID == 0 {
		return 0, errorsNoCandidate
	}
	return candidateID, nil
}

var errorsNoCandidate = fmt.Errorf("unable to satisfy staffing requirement")

func (p *schedulePlanner) fillRemainingWithRest(day int) {
	type employeeNeed struct {
		id   uint
		need int
	}

	needs := make([]employeeNeed, 0, len(p.employees))
	for _, employee := range p.employees {
		target := p.empRestTarget[employee.ID]
		needs = append(needs, employeeNeed{
			id:   employee.ID,
			need: target - p.restCount[employee.ID],
		})
	}

	sort.SliceStable(needs, func(i, j int) bool {
		return needs[i].need > needs[j].need
	})

	weekday := weekdayOf(p.req.Month, day)
	for _, item := range needs {
		if p.assignment[day][item.id] != "" {
			continue
		}
		p.assignment[day][item.id] = models.ShiftRest
		p.restCount[item.id]++
		if weekday == time.Saturday || weekday == time.Sunday {
			p.weekendRestCount[item.id]++
		}
	}
}

func (p *schedulePlanner) validateRestTargets() error {
	for _, employee := range p.employees {
		actual := p.restCount[employee.ID]
		target := p.empRestTarget[employee.ID]
		if actual < target && !p.empAllowLessRest[employee.ID] {
			return fmt.Errorf("employee %q requires %d rest days, but only %d were assigned", employee.Name, target, actual)
		}
	}
	return nil
}

func (p *schedulePlanner) buildResults() []models.ScheduleEntry {
	results := make([]models.ScheduleEntry, 0, p.totalDays*len(p.employees))

	for day := 1; day <= p.totalDays; day++ {
		for _, employee := range p.employees {
			shift := p.assignment[day][employee.ID]
			if shift == "" {
				shift = models.ShiftRest
			}
			results = append(results, models.ScheduleEntry{
				GroupID:    p.req.GroupID,
				Month:      p.req.Month,
				Day:        day,
				EmployeeID: employee.ID,
				Employee:   employee.Name,
				ShiftName:  shift,
			})
		}
	}

	return results
}

func (p *schedulePlanner) buildRemarks() []models.ScheduleRemark {
	remarks := make([]models.ScheduleRemark, 0, len(p.crossMonthNotes))
	for _, note := range p.crossMonthNotes {
		remarks = append(remarks, models.ScheduleRemark{
			GroupID: p.req.GroupID,
			Month:   p.req.Month,
			Tag:     "crossmonth",
			Content: note,
		})
	}
	return remarks
}
