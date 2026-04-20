package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"power/internal/models"
	"power/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Router struct {
	db        *gorm.DB
	nightSvc  *service.NightShiftService
	schedSvc  *service.SchedulerService
	exportSvc *service.ExportService
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

		api.GET("/roles", r.listRoles)
		api.POST("/roles", r.createRole)
		api.PUT("/roles/:id", r.updateRole)
		api.DELETE("/roles/:id", r.deleteRole)

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
		api.GET("/post-daily-requirements", r.listPostDailyRequirements)
		api.POST("/post-daily-requirements", r.upsertPostDailyRequirement)
		api.POST("/post-daily-requirements/bulk", r.replacePostDailyRequirements)
		api.GET("/post-weekday-requirements", r.listPostWeekdayRequirements)
		api.POST("/post-weekday-requirements", r.upsertPostWeekdayRequirement)
		api.POST("/post-requirements/copy-from-prev-month", r.copyPostRequirementsFromPrevMonth)

		api.GET("/rules", r.listRules)
		api.POST("/rules", r.createRule)
		api.PUT("/rules/:id", r.updateRule)
		api.DELETE("/rules/:id", r.deleteRule)

		api.GET("/constraints", r.listConstraints)
		api.POST("/constraints", r.upsertConstraint)

		api.GET("/night-shifts", r.listNightShifts)
		api.POST("/night-shifts/import", r.importNightShifts)
		api.GET("/schedule/precheck", r.precheckSchedule)
		api.POST("/schedule/generate", r.generateSchedule)
		api.GET("/schedule", r.getSchedule)
		api.GET("/schedule/export", r.exportSchedule)
		api.GET("/schedule/remarks", r.listRemarks)

		api.GET("/employee-rest-plans", r.listRestPlans)
		api.POST("/employee-rest-plans", r.upsertRestPlan)
		api.DELETE("/employee-rest-plans/:id", r.deleteRestPlan)

		api.GET("/rest-debts", r.listRestDebts)
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Department = strings.TrimSpace(req.Department)
	if req.Name == "" || req.Department == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "科室名称和小组名称不能为空"})
		return
	}
	if err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&req).Error; err != nil {
			return err
		}
		defaultRoles := []models.RoleOption{
			{GroupID: req.ID, Name: "姝ｅ紡鍛樺伐", AllowLessRest: true},
			{GroupID: req.ID, Name: "技术支持"},
			{GroupID: req.ID, Name: "鏈哄姩鍛樺伐", AllowLessRest: true},
		}
		return tx.Create(&defaultRoles).Error
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}
func (r *Router) updateGroup(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req models.Group
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := r.db.Model(&models.Group{}).Where("id = ?", id).Updates(req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (r *Router) deleteGroup(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	_ = r.db.Delete(&models.Group{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listRoles(c *gin.Context) {
	groupID := c.Query("groupId")
	q := r.db.Order("id asc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	var items []models.RoleOption
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}

func (r *Router) createRole(c *gin.Context) {
	var req models.RoleOption
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "璇峰厛閫夋嫨灏忕粍"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "瑙掕壊鍚嶇О涓嶈兘涓虹┖"})
		return
	}
	var exists int64
	_ = r.db.Model(&models.RoleOption{}).Where("group_id = ? AND name = ?", req.GroupID, req.Name).Count(&exists).Error
	if exists > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色名称已存在"})
		return
	}
	if err := r.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (r *Router) updateRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Name          string `json:"name"`
		AllowLessRest *bool  `json:"allowLessRest"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var existing models.RoleOption
	if err := r.db.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}
	updates := map[string]interface{}{}
	if strings.TrimSpace(req.Name) != "" && strings.TrimSpace(req.Name) != existing.Name {
		newName := strings.TrimSpace(req.Name)
		var exists int64
		_ = r.db.Model(&models.RoleOption{}).Where("group_id = ? AND name = ? AND id <> ?", existing.GroupID, newName, existing.ID).Count(&exists).Error
		if exists > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "角色名称已存在"})
			return
		}
		updates["name"] = newName
	}
	if req.AllowLessRest != nil {
		updates["allow_less_rest"] = *req.AllowLessRest
	}
	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if err := r.db.Model(&models.RoleOption{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) deleteRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var role models.RoleOption
	if err := r.db.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}
	var employeeCount int64
	_ = r.db.Model(&models.Employee{}).
		Where("group_id = ? AND (role = ? OR roles = ? OR roles LIKE ? OR roles LIKE ? OR roles LIKE ?)", role.GroupID, role.Name, role.Name, role.Name+",%", "%,"+role.Name, "%,"+role.Name+",%").
		Count(&employeeCount).Error
	if employeeCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "璇ヨ鑹插凡琚憳宸ヤ娇鐢紝鏃犳硶鍒犻櫎"})
		return
	}
	_ = r.db.Delete(&models.RoleOption{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listEmployees(c *gin.Context) {
	groupID := c.Query("groupId")
	q := r.db.Order("id desc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	var items []models.Employee
	_ = q.Find(&items).Error
	resp := make([]gin.H, 0, len(items))
	for _, item := range items {
		resp = append(resp, gin.H{
			"id":           item.ID,
			"name":         item.Name,
			"role":         item.Role,
			"roles":        item.RoleList(),
			"category":     item.Category,
			"groupId":      item.GroupID,
			"canNight":     item.CanNight,
			"active":       item.Active,
			"sortPriority": item.SortPriority,
			"createdAt":    item.CreatedAt,
			"updatedAt":    item.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, resp)
}
func (r *Router) createEmployee(c *gin.Context) {
	var req struct {
		GroupID      uint     `json:"groupId"`
		Name         string   `json:"name"`
		Role         string   `json:"role"`
		Roles        []string `json:"roles"`
		Category     string   `json:"category"`
		CanNight     bool     `json:"canNight"`
		Active       bool     `json:"active"`
		SortPriority int      `json:"sortPriority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Category = strings.TrimSpace(req.Category)
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "璇峰厛閫夋嫨灏忕粍"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "鍛樺伐濮撳悕涓嶈兘涓虹┖"})
		return
	}
	roleNames := req.Roles
	if len(roleNames) == 0 && strings.TrimSpace(req.Role) != "" {
		roleNames = []string{strings.TrimSpace(req.Role)}
	}
	roleNames = models.ParseRoleList(models.JoinRoleList(roleNames))
	if len(roleNames) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "鍛樺伐瑙掕壊涓嶈兘涓虹┖"})
		return
	}
	for _, roleName := range roleNames {
		var roleCount int64
		_ = r.db.Model(&models.RoleOption{}).Where("group_id = ? AND name = ?", req.GroupID, roleName).Count(&roleCount).Error
		if roleCount == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "员工角色不在当前小组的角色配置中：" + roleName})
			return
		}
	}
	item := models.Employee{
		GroupID:      req.GroupID,
		Name:         req.Name,
		Role:         models.EmployeeRole(roleNames[0]),
		Roles:        models.JoinRoleList(roleNames),
		Category:     req.Category,
		CanNight:     req.CanNight,
		Active:       req.Active,
		SortPriority: req.SortPriority,
	}
	if err := r.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": item.ID, "name": item.Name, "role": item.Role, "roles": item.RoleList(), "category": item.Category,
		"groupId": item.GroupID, "canNight": item.CanNight, "active": item.Active, "sortPriority": item.SortPriority,
		"createdAt": item.CreatedAt, "updatedAt": item.UpdatedAt,
	})
}
func (r *Router) updateEmployee(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Name     string   `json:"name"`
		Role     string   `json:"role"`
		Roles    []string `json:"roles"`
		Category string   `json:"category"`
		CanNight *bool    `json:"canNight"`
		Active   *bool    `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var existing models.Employee
	if err := r.db.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "员工不存在"})
		return
	}

	updates := map[string]interface{}{}
	if strings.TrimSpace(req.Name) != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.Category != "" || strings.TrimSpace(req.Category) == "" {
		updates["category"] = strings.TrimSpace(req.Category)
	}
	if req.CanNight != nil {
		updates["can_night"] = *req.CanNight
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}

	roleNames := req.Roles
	if len(roleNames) == 0 && strings.TrimSpace(req.Role) != "" {
		roleNames = []string{strings.TrimSpace(req.Role)}
	}
	if len(roleNames) > 0 {
		roleNames = models.ParseRoleList(models.JoinRoleList(roleNames))
		if len(roleNames) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "鍛樺伐瑙掕壊涓嶈兘涓虹┖"})
			return
		}
		for _, roleName := range roleNames {
			var roleCount int64
			_ = r.db.Model(&models.RoleOption{}).Where("group_id = ? AND name = ?", existing.GroupID, roleName).Count(&roleCount).Error
			if roleCount == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "员工角色不在当前小组的角色配置中：" + roleName})
				return
			}
		}
		updates["role"] = roleNames[0]
		updates["roles"] = models.JoinRoleList(roleNames)
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "娌℃湁鍙洿鏂扮殑瀛楁"})
		return
	}
	if err := r.db.Model(&models.Employee{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (r *Router) deleteEmployee(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	_ = r.db.Delete(&models.Employee{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listSpecialties(c *gin.Context) {
	groupID := c.Query("groupId")
	q := r.db.Order("id asc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	var items []models.SpecialtyOption
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}

func (r *Router) createSpecialty(c *gin.Context) {
	var req models.SpecialtyOption
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "璇峰厛閫夋嫨灏忕粍"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "涓撲笟鏂瑰悜鍚嶇О涓嶈兘涓虹┖"})
		return
	}
	if err := r.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (r *Router) deleteSpecialty(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	_ = r.db.Delete(&models.SpecialtyOption{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listPosts(c *gin.Context) {
	groupID := c.Query("groupId")
	q := r.db.Order("priority asc,id asc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	var items []models.ShiftPost
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}
func (r *Router) createPost(c *gin.Context) {
	var req models.ShiftPost
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "璇峰厛閫夋嫨灏忕粍"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "宀椾綅鍚嶇О涓嶈兘涓虹┖"})
		return
	}
	if req.Required <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "宀椾綅浜烘暟蹇呴』澶т簬0"})
		return
	}
	if err := r.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}
func (r *Router) updatePost(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req models.ShiftPost
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := r.db.Model(&models.ShiftPost{}).Where("id = ?", id).Updates(req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (r *Router) deletePost(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	_ = r.db.Delete(&models.ShiftPost{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listPostDailyRequirements(c *gin.Context) {
	groupID := c.Query("groupId")
	month := c.Query("month")
	q := r.db.Order("day asc, post_name asc, id asc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	if month != "" {
		q = q.Where("month = ?", month)
	}
	var items []models.PostDailyRequirement
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}

func (r *Router) upsertPostDailyRequirement(c *gin.Context) {
	var req models.PostDailyRequirement
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.PostName = strings.TrimSpace(req.PostName)
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "?????"})
		return
	}
	if req.Month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??????"})
		return
	}
	if _, err := time.Parse("2006-01", req.Month); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??????YYYY-MM"})
		return
	}
	if req.Day < 1 || req.Day > 31 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "?????1-31??"})
		return
	}
	if req.PostName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "????????"})
		return
	}
	if req.Required < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??????0"})
		return
	}
	var postCount int64
	_ = r.db.Model(&models.ShiftPost{}).Where("group_id = ? AND name = ?", req.GroupID, req.PostName).Count(&postCount).Error
	if postCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "???????????"})
		return
	}
	var existing models.PostDailyRequirement
	err := r.db.Where("group_id = ? AND month = ? AND day = ? AND post_name = ?", req.GroupID, req.Month, req.Day, req.PostName).First(&existing).Error
	if err == nil {
		existing.Required = req.Required
		_ = r.db.Save(&existing).Error
		c.JSON(http.StatusOK, existing)
		return
	}
	if err := r.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (r *Router) replacePostDailyRequirements(c *gin.Context) {
	var req struct {
		GroupID uint                          `json:"groupId"`
		Month   string                        `json:"month"`
		Items   []models.PostDailyRequirement `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "groupId涓嶈兘涓虹┖"})
		return
	}
	if req.Month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "month涓嶈兘涓虹┖"})
		return
	}
	if _, err := time.Parse("2006-01", req.Month); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "month鏍煎紡搴斾负YYYY-MM"})
		return
	}

	var posts []models.ShiftPost
	if err := r.db.Where("group_id = ?", req.GroupID).Find(&posts).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	allowedPosts := map[string]bool{}
	for _, post := range posts {
		allowedPosts[post.Name] = true
	}

	cleaned := make([]models.PostDailyRequirement, 0, len(req.Items))
	for _, item := range req.Items {
		item.GroupID = req.GroupID
		item.Month = req.Month
		item.PostName = strings.TrimSpace(item.PostName)
		if item.Day < 1 || item.Day > 31 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "鏃ユ湡蹇呴』鍦?-31涔嬮棿"})
			return
		}
		if item.PostName == "" || !allowedPosts[item.PostName] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "瀛樺湪鏈畾涔夌殑鐝"})
			return
		}
		if item.Required < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "浜烘暟涓嶈兘灏忎簬0"})
			return
		}
		cleaned = append(cleaned, models.PostDailyRequirement{
			GroupID:  req.GroupID,
			Month:    req.Month,
			Day:      item.Day,
			PostName: item.PostName,
			Required: item.Required,
		})
	}

	if err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ? AND month = ?", req.GroupID, req.Month).Delete(&models.PostDailyRequirement{}).Error; err != nil {
			return err
		}
		if len(cleaned) == 0 {
			return nil
		}
		return tx.Create(&cleaned).Error
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "count": len(cleaned)})
}

func (r *Router) listPostWeekdayRequirements(c *gin.Context) {
	groupID := c.Query("groupId")
	month := c.Query("month")
	q := r.db.Order("weekday asc, post_name asc, id asc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	if month != "" {
		q = q.Where("month = ?", month)
	}
	var items []models.PostWeekdayRequirement
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}

func (r *Router) upsertPostWeekdayRequirement(c *gin.Context) {
	var req models.PostWeekdayRequirement
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.PostName = strings.TrimSpace(req.PostName)
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "?????"})
		return
	}
	if req.Month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??????"})
		return
	}
	if _, err := time.Parse("2006-01", req.Month); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??????YYYY-MM"})
		return
	}
	if req.Weekday < 0 || req.Weekday > 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "?????0-6??"})
		return
	}
	if req.PostName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "????????"})
		return
	}
	if req.Required < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??????0"})
		return
	}
	var postCount int64
	_ = r.db.Model(&models.ShiftPost{}).Where("group_id = ? AND name = ?", req.GroupID, req.PostName).Count(&postCount).Error
	if postCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "???????????"})
		return
	}
	var existing models.PostWeekdayRequirement
	err := r.db.Where("group_id = ? AND month = ? AND weekday = ? AND post_name = ?", req.GroupID, req.Month, req.Weekday, req.PostName).First(&existing).Error
	if err == nil {
		existing.Required = req.Required
		_ = r.db.Save(&existing).Error
		c.JSON(http.StatusOK, existing)
		return
	}
	if err := r.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (r *Router) copyPostRequirementsFromPrevMonth(c *gin.Context) {
	var req struct {
		GroupID uint   `json:"groupId"`
		Month   string `json:"month"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "璇峰厛閫夋嫨灏忕粍"})
		return
	}
	if req.Month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "鏈堜唤涓嶈兘涓虹┖"})
		return
	}
	monthTime, err := time.Parse("2006-01", req.Month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "鏈堜唤鏍煎紡搴斾负YYYY-MM"})
		return
	}
	prevMonth := monthTime.AddDate(0, -1, 0).Format("2006-01")

	var prevDaily []models.PostDailyRequirement
	var prevWeekday []models.PostWeekdayRequirement
	_ = r.db.Where("group_id = ? AND month = ?", req.GroupID, prevMonth).Find(&prevDaily).Error
	_ = r.db.Where("group_id = ? AND month = ?", req.GroupID, prevMonth).Find(&prevWeekday).Error

	if err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ? AND month = ?", req.GroupID, req.Month).Delete(&models.PostDailyRequirement{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ? AND month = ?", req.GroupID, req.Month).Delete(&models.PostWeekdayRequirement{}).Error; err != nil {
			return err
		}
		if len(prevDaily) > 0 {
			newDaily := make([]models.PostDailyRequirement, 0, len(prevDaily))
			for _, item := range prevDaily {
				newDaily = append(newDaily, models.PostDailyRequirement{
					GroupID: req.GroupID, Month: req.Month, Day: item.Day, PostName: item.PostName, Required: item.Required,
				})
			}
			if err := tx.Create(&newDaily).Error; err != nil {
				return err
			}
		}
		if len(prevWeekday) > 0 {
			newWeekday := make([]models.PostWeekdayRequirement, 0, len(prevWeekday))
			for _, item := range prevWeekday {
				newWeekday = append(newWeekday, models.PostWeekdayRequirement{
					GroupID: req.GroupID, Month: req.Month, Weekday: item.Weekday, PostName: item.PostName, Required: item.Required,
				})
			}
			if err := tx.Create(&newWeekday).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"sourceMonth": prevMonth,
		"dailyCount":  len(prevDaily),
		"weeklyCount": len(prevWeekday),
	})
}

func (r *Router) listRules(c *gin.Context) {
	groupID := c.Query("groupId")
	month := c.Query("month")
	q := r.db.Order("id desc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	if month != "" {
		q = q.Where("(month = ? OR month = '')", month)
	}
	var items []models.SpecialRule
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}
func (r *Router) createRule(c *gin.Context) {
	var req struct {
		GroupID     uint   `json:"groupId"`
		Month       string `json:"month"`
		Name        string `json:"name"`
		RuleType    string `json:"ruleType"`
		DayOfMonth  int    `json:"dayOfMonth"`
		Weekday     int    `json:"weekday"`
		PostName    string `json:"postName"`
		Required    int    `json:"required"`
		EmployeeID  uint   `json:"employeeId"`
		EmployeeIDs []uint `json:"employeeIds"`
		Enabled     bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.PostName = strings.TrimSpace(req.PostName)
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "璇峰厛閫夋嫨灏忕粍"})
		return
	}
	if req.PostName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "瑙勫垯宀椾綅涓嶈兘涓虹┖"})
		return
	}
	employeeIDs := req.EmployeeIDs
	if len(employeeIDs) == 0 && req.EmployeeID != 0 {
		employeeIDs = []uint{req.EmployeeID}
	}
	if len(employeeIDs) == 0 && req.Required <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "瑙勫垯浜烘暟蹇呴』澶т簬0"})
		return
	}
	if req.RuleType == "date" && (req.DayOfMonth < 1 || req.DayOfMonth > 31) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "鎸夋棩鏈熻鍒欑殑鏃ユ湡蹇呴』鍦?-31涔嬮棿"})
		return
	}
	if req.RuleType == "weekday" && (req.Weekday < 0 || req.Weekday > 6) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "鎸夋槦鏈熻鍒欑殑鏄熸湡蹇呴』鍦?-6涔嬮棿"})
		return
	}
	capSize := len(employeeIDs)
	if capSize == 0 {
		capSize = 1
	}
	items := make([]models.SpecialRule, 0, capSize)
	if len(employeeIDs) > 0 {
		seen := map[uint]bool{}
		for _, employeeID := range employeeIDs {
			if employeeID == 0 || seen[employeeID] {
				continue
			}
			seen[employeeID] = true
			var emp models.Employee
			if err := r.db.Where("id = ? AND group_id = ?", employeeID, req.GroupID).First(&emp).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "鎸囧畾浜哄憳涓嶅瓨鍦ㄤ簬褰撳墠灏忕粍"})
				return
			}
			items = append(items, models.SpecialRule{
				GroupID:      req.GroupID,
				Month:        req.Month,
				Name:         req.Name,
				RuleType:     req.RuleType,
				DayOfMonth:   req.DayOfMonth,
				Weekday:      req.Weekday,
				PostName:     req.PostName,
				Required:     1,
				EmployeeID:   employeeID,
				EmployeeName: emp.Name,
				Enabled:      req.Enabled,
			})
		}
		if len(items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请至少选择一位指定人员"})
			return
		}
	} else {
		items = append(items, models.SpecialRule{
			GroupID:    req.GroupID,
			Month:      req.Month,
			Name:       req.Name,
			RuleType:   req.RuleType,
			DayOfMonth: req.DayOfMonth,
			Weekday:    req.Weekday,
			PostName:   req.PostName,
			Required:   req.Required,
			Enabled:    req.Enabled,
		})
	}
	if err := r.db.Create(&items).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}
func (r *Router) updateRule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req models.SpecialRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := r.db.Model(&models.SpecialRule{}).Where("id = ?", id).Updates(req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (r *Router) deleteRule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	_ = r.db.Delete(&models.SpecialRule{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listConstraints(c *gin.Context) {
	groupID := c.Query("groupId")
	month := c.Query("month")
	q := r.db.Order("id desc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	if month != "" {
		q = q.Where("month = ?", month)
	}
	var items []models.MonthlyConstraint
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}
func (r *Router) upsertConstraint(c *gin.Context) {
	var req models.MonthlyConstraint
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "璇峰厛閫夋嫨灏忕粍"})
		return
	}
	if req.Month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "鏈堜唤涓嶈兘涓虹┖"})
		return
	}
	if strings.TrimSpace(string(req.Role)) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "瑙掕壊涓嶈兘涓虹┖"})
		return
	}
	var roleCount int64
	_ = r.db.Model(&models.RoleOption{}).Where("group_id = ? AND name = ?", req.GroupID, string(req.Role)).Count(&roleCount).Error
	if roleCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色不在当前小组角色配置中"})
		return
	}
	if req.RestDaysGoal < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "浼戞伅鐩爣涓嶈兘灏忎簬0"})
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
	if err := r.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "缂哄皯瀵煎叆鏂囦欢"})
		return
	}
	defer func() { _ = file.Close() }()
	if err := r.nightSvc.Import(month, file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func daysInMonthForAPI(month string) (int, error) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return 0, err
	}
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day(), nil
}

func weekdayOfDay(month string, day int) time.Weekday {
	t, err := time.Parse("2006-01-02", month+"-"+fmt.Sprintf("%02d", day))
	if err != nil {
		return time.Monday
	}
	return t.Weekday()
}

func parseFixedDaysForAPI(raw string) []int {
	parts := strings.Split(raw, ",")
	days := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		day, err := strconv.Atoi(part)
		if err == nil && day > 0 {
			days = append(days, day)
		}
	}
	return days
}

func buildForcedUnavailableByDay(db *gorm.DB, groupID uint, month string, totalDays int) (map[int]map[uint]bool, error) {
	var employees []models.Employee
	if err := db.Where("group_id = ? AND active = 1", groupID).Find(&employees).Error; err != nil {
		return nil, err
	}
	employeeByName := make(map[string]models.Employee, len(employees))
	for _, emp := range employees {
		employeeByName[strings.TrimSpace(emp.Name)] = emp
	}

	unavailable := map[int]map[uint]bool{}
	mark := func(day int, empID uint) {
		if day < 1 || day > totalDays {
			return
		}
		if unavailable[day] == nil {
			unavailable[day] = map[uint]bool{}
		}
		unavailable[day][empID] = true
	}

	var currentNight []models.NightShiftRecord
	if err := db.Where("month = ?", month).Find(&currentNight).Error; err != nil {
		return nil, err
	}
	lastNightDay := 0
	for _, item := range currentNight {
		if item.Day > lastNightDay {
			lastNightDay = item.Day
		}
	}
	for _, item := range currentNight {
		restDays := []int{item.Day + 1, item.Day + 2}
		if item.Day == lastNightDay {
			restDays = []int{item.Day + 1}
		}
		for _, name := range []string{item.StaffA, item.StaffB} {
			emp, ok := employeeByName[strings.TrimSpace(name)]
			if !ok {
				continue
			}
			for _, day := range restDays {
				mark(day, emp.ID)
			}
		}
	}

	prevMonthTime, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, err
	}
	prevMonthTime = prevMonthTime.AddDate(0, -1, 0)
	prevMonth := prevMonthTime.Format("2006-01")
	prevTotalDays := time.Date(prevMonthTime.Year(), prevMonthTime.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()

	var prevNight []models.NightShiftRecord
	if err := db.Where("month = ?", prevMonth).Find(&prevNight).Error; err != nil {
		return nil, err
	}
	prevLastNightDay := 0
	for _, item := range prevNight {
		if item.Day > prevLastNightDay {
			prevLastNightDay = item.Day
		}
	}
	for _, item := range prevNight {
		restDays := []int{item.Day + 1, item.Day + 2}
		if item.Day == prevLastNightDay {
			restDays = []int{item.Day + 1}
		}
		for _, name := range []string{item.StaffA, item.StaffB} {
			emp, ok := employeeByName[strings.TrimSpace(name)]
			if !ok {
				continue
			}
			for _, day := range restDays {
				mark(day-prevTotalDays, emp.ID)
			}
		}
	}

	var restPlans []models.EmployeeRestPlan
	if err := db.Where("group_id = ? AND month = ?", groupID, month).Find(&restPlans).Error; err != nil {
		return nil, err
	}
	for _, plan := range restPlans {
		for _, day := range parseFixedDaysForAPI(plan.FixedDays) {
			mark(day, plan.EmployeeID)
		}
	}
	return unavailable, nil
}

func (r *Router) precheckSchedule(c *gin.Context) {
	groupID64, err := strconv.ParseUint(c.Query("groupId"), 10, 64)
	if err != nil || groupID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "groupId涓嶈兘涓虹┖"})
		return
	}
	groupID := uint(groupID64)
	month := c.Query("month")
	if month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "month涓嶈兘涓虹┖"})
		return
	}
	totalDays, err := daysInMonthForAPI(month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "month鏍煎紡搴斾负YYYY-MM"})
		return
	}
	var activeCount int64
	_ = r.db.Model(&models.Employee{}).Where("group_id = ? AND active = 1", groupID).Count(&activeCount).Error
	if activeCount == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "当前小组无可用员工"})
		return
	}
	var posts []models.ShiftPost
	_ = r.db.Where("group_id = ? AND enabled = 1", groupID).Find(&posts).Error
	if len(posts) == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "当前小组无班种配置"})
		return
	}
	var daily []models.PostDailyRequirement
	_ = r.db.Where("group_id = ? AND month = ?", groupID, month).Find(&daily).Error
	var weekly []models.PostWeekdayRequirement
	_ = r.db.Where("group_id = ? AND month = ?", groupID, month).Find(&weekly).Error
	var rules []models.SpecialRule
	_ = r.db.Where("group_id = ? AND enabled = 1 AND (month = ? OR month = '')", groupID, month).Find(&rules).Error
	var nightCount int64
	_ = r.db.Model(&models.NightShiftRecord{}).Where("month = ?", month).Count(&nightCount).Error
	unavailableByDay, err := buildForcedUnavailableByDay(r.db, groupID, month, totalDays)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": err.Error()})
		return
	}

	dailyByDay := map[int]map[string]int{}
	for _, item := range daily {
		if dailyByDay[item.Day] == nil {
			dailyByDay[item.Day] = map[string]int{}
		}
		dailyByDay[item.Day][item.PostName] = item.Required
	}
	weekByWeekday := map[int]map[string]int{}
	for _, item := range weekly {
		if weekByWeekday[item.Weekday] == nil {
			weekByWeekday[item.Weekday] = map[string]int{}
		}
		weekByWeekday[item.Weekday][item.PostName] = item.Required
	}

	maxNeed := 0
	overloadDays := []gin.H{}
	for day := 1; day <= totalDays; day++ {
		requiredByPost := map[string]int{}
		for _, post := range posts {
			requiredByPost[post.Name] = post.Required
		}
		weekday := int(weekdayOfDay(month, day))
		if byPost := weekByWeekday[weekday]; byPost != nil {
			for postName, required := range byPost {
				requiredByPost[postName] = required
			}
		}
		if byPost := dailyByDay[day]; byPost != nil {
			for postName, required := range byPost {
				requiredByPost[postName] = required
			}
		}
		for _, rule := range rules {
			if rule.EmployeeID != 0 {
				continue
			}
			match := (rule.RuleType == "date" && rule.DayOfMonth == day) ||
				(rule.RuleType == "weekday" && rule.Weekday == weekday)
			if match {
				requiredByPost[rule.PostName] = rule.Required
			}
		}
		dayNeed := 0
		for _, v := range requiredByPost {
			if v > 0 {
				dayNeed += v
			}
		}
		if dayNeed > maxNeed {
			maxNeed = dayNeed
		}
		available := int(activeCount)
		if unavailableByDay[day] != nil {
			available -= len(unavailableByDay[day])
		}
		if dayNeed > available {
			overloadDays = append(overloadDays, gin.H{
				"day":       day,
				"required":  dayNeed,
				"active":    activeCount,
				"available": available,
			})
		}
	}

	if len(overloadDays) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"ok":           false,
			"message":      "存在日需求超过可用人数，请先调整班种需求",
			"overloadDays": overloadDays,
			"nightCount":   nightCount,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":         true,
		"message":    "棰勬鏌ラ€氳繃",
		"active":     activeCount,
		"maxNeed":    maxNeed,
		"nightCount": nightCount,
		"days":       totalDays,
	})
}

func (r *Router) generateSchedule(c *gin.Context) {
	var req service.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := r.schedSvc.Generate(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (r *Router) getSchedule(c *gin.Context) {
	groupID := c.Query("groupId")
	month := c.Query("month")
	var items []models.ScheduleEntry
	q := r.db.Order("day asc, employee asc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	if month != "" {
		q = q.Where("month = ?", month)
	}
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}

func (r *Router) exportSchedule(c *gin.Context) {
	groupID64, err := strconv.ParseUint(c.Query("groupId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "groupId涓嶈兘涓虹┖"})
		return
	}
	month := c.Query("month")
	c.Header("Content-Disposition", `attachment; filename="schedule_`+month+`.xlsx"`)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := r.exportSvc.ExportMonth(uint(groupID64), month, c.Writer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func (r *Router) listRemarks(c *gin.Context) {
	groupID := c.Query("groupId")
	month := c.Query("month")
	q := r.db.Order("id asc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	if month != "" {
		q = q.Where("month = ?", month)
	}
	var items []models.ScheduleRemark
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}

func (r *Router) listRestPlans(c *gin.Context) {
	groupID := c.Query("groupId")
	month := c.Query("month")
	q := r.db.Order("id asc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	if month != "" {
		q = q.Where("month = ?", month)
	}
	var items []models.EmployeeRestPlan
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}

func (r *Router) upsertRestPlan(c *gin.Context) {
	var req models.EmployeeRestPlan
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.GroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "groupId涓嶈兘涓虹┖"})
		return
	}
	if req.Month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "month涓嶈兘涓虹┖"})
		return
	}
	if req.EmployeeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "employeeId涓嶈兘涓虹┖"})
		return
	}
	// 楠岃瘉鍛樺伐瀛樺湪
	var emp models.Employee
	if err := r.db.Where("id = ? AND group_id = ?", req.EmployeeID, req.GroupID).First(&emp).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "员工不存在"})
		return
	}
	req.EmployeeName = emp.Name
	var existing models.EmployeeRestPlan
	err := r.db.Where("group_id = ? AND month = ? AND employee_id = ?", req.GroupID, req.Month, req.EmployeeID).First(&existing).Error
	if err == nil {
		existing.FixedDays = req.FixedDays
		existing.FloatDays = req.FloatDays
		existing.Note = req.Note
		_ = r.db.Save(&existing).Error
		c.JSON(http.StatusOK, existing)
		return
	}
	if err := r.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (r *Router) deleteRestPlan(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	_ = r.db.Delete(&models.EmployeeRestPlan{}, id).Error
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listRestDebts(c *gin.Context) {
	groupID := c.Query("groupId")
	month := c.Query("month")
	q := r.db.Order("month desc, id asc")
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	if month != "" {
		q = q.Where("month = ?", month)
	}
	var items []models.RestDebtRecord
	_ = q.Find(&items).Error
	c.JSON(http.StatusOK, items)
}
