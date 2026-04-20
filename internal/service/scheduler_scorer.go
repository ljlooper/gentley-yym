package service

import (
	"time"

	"power/internal/models"
)

type CandidateContext struct {
	Employee         models.Employee
	Day              int
	Month            string
	WorkCount        map[uint]int
	WeekendRestCount map[uint]int
	RestCount        map[uint]int
	RestTarget       map[uint]int
}

type WorkloadScorer struct{}

func (WorkloadScorer) Score(ctx CandidateContext) int {
	restGap := ctx.RestTarget[ctx.Employee.ID] - ctx.RestCount[ctx.Employee.ID]
	if restGap < 0 {
		restGap = 0
	}

	score := ctx.WorkCount[ctx.Employee.ID]*10 + restGap*12
	if weekday := weekdayOf(ctx.Month, ctx.Day); weekday == time.Saturday || weekday == time.Sunday {
		score += ctx.WeekendRestCount[ctx.Employee.ID] * 3
	}
	return score
}
