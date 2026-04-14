package api

import (
	"net/http"
	"strconv"
	"strings"

	"power/internal/models"
	"power/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Router struct {
	db         *gorm.DB
	nightSvc   *service.NightShiftService
	schedSvc   *service.SchedulerService
	exportSvc  *service.ExportService
}

func NewRouter(db *gorm.DB) *gin.Engine {
	r := &Router{
		db:        db,
		nightSvc:  service.NewNightShiftService(db),
		schedSvc:  service.NewSchedulerService(db),
		exportSvc: service.NewExportService(db),
	}
	engine := gin.Default()
	engine.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})
	engine.GET("/style.css", func(c *gin.Context) {
		c.File("./web/style.css")
	})
	engine.GET("/app.js", func(c *gin.Context) {
		c.File("./web/app.js")
	})
	engine.GET("/api/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	api := engine.Group("/api")
	{
		api.GET("/groups", r.listGroups)
		api.POST("/groups", r.createGroup)
		api.PUT("/groups/:id", r.updateGroup)
		api.DELETE("/groups/:id", r.deleteGroup)

		api.GET("/employees", r.listEmployees)
		api.POST("/employees", r.createEmployee)
		api.PUT("/employees/:id", r.updateEmployee)
		api.DELETE("/employees/:id", r.deleteEmployee)

		api.GET("/specialties", r.listSpecialties)
		api.POST("/specialties", r.createSpecialty)
		api.DELETE("/specialties/:id", r.deleteSpecialty)

		api.GET("/posts", r.listPosts)
		api.POST("/posts", r.createPost)
		api.PUT("/posts/:id", r.updatePost)
		api.DELETE("/posts/:id", r.deletePost)

		api.GET("/rules", r.listRules)
		api.POST("/rules", r.createRule)
		api.PUT("/rules/:id", r.updateRule)
		api.DELETE("/rules/:id", r.deleteRule)

		api.GET("/constraints", r.listConstraints)
		api.POST("/constraints", r.upsertConstraint)

		api.GET("/night-shifts", r.listNightShifts)
		api.POST("/night-shifts/import", r.importNightShifts)
		api.POST("/schedule/generate", r.generateSchedule)
		api.GET("/schedule", r.getSchedule)
		api.GET("/schedule/export", r.exportSchedule)
	}
	return engine
}

func parseID(c *gin.Context) (uint, bool) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(id64), true
}

func (r *Router) listGroups(c *gin.Context) {
	var items []models.Group
	_ = r.db.Order("id desc").Find(&items).Error
	c.JSON(http.StatusOK, items)
}
func (r *Router) createGroup(c *gin.Context) {
	var req models.Group
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	req.Name = strings.TrimSpace(req.Name)
	req.Department = strings.TrimSpace(req.Department)
	if req.Name == "" || req.Department == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "科室名称和小组名称不能为空"})
		return
	}
	if err := r.db.Create(&req).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, req)
}
func (r *Router) updateGroup(c *gin.Context) {
	id, ok := parseID(c); if !ok { return }
	var req models.Group
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if err := r.db.Model(&models.Group{}).Where("id = ?", id).Updates(req).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (r *Router) deleteGroup(c *gin.Context) {
	id, ok := parseID(c); if !ok { return }
	_ = r.db.Delete(&models.Group{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listEmployees(c *gin.Context) {
	groupID := c.Query("groupId")
	q := r.db.Order("id desc")
	if groupID != "" { q = q.Where("group_id = ?", groupID) }
	var items []models.Employee
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}
func (r *Router) createEmployee(c *gin.Context) {
	var req models.Employee
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	req.Name = strings.TrimSpace(req.Name)
	req.Category = strings.TrimSpace(req.Category)
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先选择小组"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "员工姓名不能为空"})
		return
	}
	if err := r.db.Create(&req).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, req)
}
func (r *Router) updateEmployee(c *gin.Context) {
	id, ok := parseID(c); if !ok { return }
	var req models.Employee
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if err := r.db.Model(&models.Employee{}).Where("id = ?", id).Updates(req).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (r *Router) deleteEmployee(c *gin.Context) {
	id, ok := parseID(c); if !ok { return }
	_ = r.db.Delete(&models.Employee{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listSpecialties(c *gin.Context) {
	groupID := c.Query("groupId")
	q := r.db.Order("id asc")
	if groupID != "" { q = q.Where("group_id = ?", groupID) }
	var items []models.SpecialtyOption
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}

func (r *Router) createSpecialty(c *gin.Context) {
	var req models.SpecialtyOption
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	req.Name = strings.TrimSpace(req.Name)
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先选择小组"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "专业方向名称不能为空"})
		return
	}
	if err := r.db.Create(&req).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, req)
}

func (r *Router) deleteSpecialty(c *gin.Context) {
	id, ok := parseID(c); if !ok { return }
	_ = r.db.Delete(&models.SpecialtyOption{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listPosts(c *gin.Context) {
	groupID := c.Query("groupId")
	q := r.db.Order("priority asc,id asc")
	if groupID != "" { q = q.Where("group_id = ?", groupID) }
	var items []models.ShiftPost
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}
func (r *Router) createPost(c *gin.Context) {
	var req models.ShiftPost
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	req.Name = strings.TrimSpace(req.Name)
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先选择小组"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "岗位名称不能为空"})
		return
	}
	if req.Required <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "岗位人数必须大于0"})
		return
	}
	if err := r.db.Create(&req).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, req)
}
func (r *Router) updatePost(c *gin.Context) {
	id, ok := parseID(c); if !ok { return }
	var req models.ShiftPost
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if err := r.db.Model(&models.ShiftPost{}).Where("id = ?", id).Updates(req).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (r *Router) deletePost(c *gin.Context) {
	id, ok := parseID(c); if !ok { return }
	_ = r.db.Delete(&models.ShiftPost{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listRules(c *gin.Context) {
	groupID := c.Query("groupId")
	q := r.db.Order("id desc")
	if groupID != "" { q = q.Where("group_id = ?", groupID) }
	var items []models.SpecialRule
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}
func (r *Router) createRule(c *gin.Context) {
	var req models.SpecialRule
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	req.PostName = strings.TrimSpace(req.PostName)
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先选择小组"})
		return
	}
	if req.PostName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则岗位不能为空"})
		return
	}
	if req.EmployeeID == 0 && req.Required <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则人数必须大于0"})
		return
	}
	if req.RuleType == "date" && (req.DayOfMonth < 1 || req.DayOfMonth > 31) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "按日期规则的日期必须在1-31之间"})
		return
	}
	if req.RuleType == "weekday" && (req.Weekday < 0 || req.Weekday > 6) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "按星期规则的星期必须在0-6之间"})
		return
	}
	if req.EmployeeID != 0 {
		var emp models.Employee
		if err := r.db.Where("id = ? AND group_id = ?", req.EmployeeID, req.GroupID).First(&emp).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "指定人员不存在于当前小组"})
			return
		}
		req.EmployeeName = emp.Name
		req.Required = 1
	}
	if err := r.db.Create(&req).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, req)
}
func (r *Router) updateRule(c *gin.Context) {
	id, ok := parseID(c); if !ok { return }
	var req models.SpecialRule
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if err := r.db.Model(&models.SpecialRule{}).Where("id = ?", id).Updates(req).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (r *Router) deleteRule(c *gin.Context) {
	id, ok := parseID(c); if !ok { return }
	_ = r.db.Delete(&models.SpecialRule{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listConstraints(c *gin.Context) {
	groupID := c.Query("groupId")
	month := c.Query("month")
	q := r.db.Order("id desc")
	if groupID != "" { q = q.Where("group_id = ?", groupID) }
	if month != "" { q = q.Where("month = ?", month) }
	var items []models.MonthlyConstraint
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}
func (r *Router) upsertConstraint(c *gin.Context) {
	var req models.MonthlyConstraint
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先选择小组"})
		return
	}
	if req.Month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "月份不能为空"})
		return
	}
	if req.RestDaysGoal < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "休息目标不能小于0"})
		return
	}
	var existing models.MonthlyConstraint
	err := r.db.Where("group_id = ? AND month = ? AND role = ?", req.GroupID, req.Month, req.Role).First(&existing).Error
	if err == nil {
		existing.RestDaysGoal = req.RestDaysGoal
		_ = r.db.Save(&existing).Error
		c.JSON(http.StatusOK, existing)
		return
	}
	if err := r.db.Create(&req).Error; err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, req)
}

func (r *Router) listNightShifts(c *gin.Context) {
	month := c.Query("month")
	q := r.db.Order("day asc, id asc")
	if month != "" {
		q = q.Where("month = ?", month)
	}
	var items []models.NightShiftRecord
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}

func (r *Router) importNightShifts(c *gin.Context) {
	month := c.PostForm("month")
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少导入文件"})
		return
	}
	defer func() { _ = file.Close() }()
	if err := r.nightSvc.Import(month, file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) generateSchedule(c *gin.Context) {
	var req service.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	result, err := r.schedSvc.Generate(req)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, result)
}

func (r *Router) getSchedule(c *gin.Context) {
	groupID := c.Query("groupId")
	month := c.Query("month")
	var items []models.ScheduleEntry
	q := r.db.Order("day asc, employee asc")
	if groupID != "" { q = q.Where("group_id = ?", groupID) }
	if month != "" { q = q.Where("month = ?", month) }
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}

func (r *Router) exportSchedule(c *gin.Context) {
	groupID64, err := strconv.ParseUint(c.Query("groupId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "groupId不能为空"})
		return
	}
	month := c.Query("month")
	c.Header("Content-Disposition", `attachment; filename="schedule_`+month+`.xlsx"`)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := r.exportSvc.ExportMonth(uint(groupID64), month, c.Writer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}
