package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"image"
	"image/jpeg"
	_ "image/png"

	"github.com/nfnt/resize"

	"village-bill/database"
	"village-bill/middleware"
	"village-bill/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
)

func logAudit(c *gin.Context, action string, entityType string, entityID uint, details interface{}) {
	adminUsername := c.GetString("admin_username")
	if adminUsername == "" {
		adminUsername = "system"
	}
	detailsJSON, _ := json.Marshal(details)
	database.DB.Create(&models.AuditLog{
		AdminUsername: adminUsername,
		Action:        action,
		EntityType:    entityType,
		EntityID:      entityID,
		Details:       string(detailsJSON),
	})
}

func Login(c *gin.Context) {
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var adminUser models.AdminUser
	if err := database.DB.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin user not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(adminUser.PasswordHash), []byte(body.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect password"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin":    true,
		"username": adminUser.Username,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString(middleware.JwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func ChangePassword(c *gin.Context) {
	var body struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	adminUsername := c.GetString("admin_username")
	if adminUsername == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var adminUser models.AdminUser
	if err := database.DB.Where("username = ?", adminUsername).First(&adminUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin user not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(adminUser.PasswordHash), []byte(body.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect old password"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}

	if err := database.DB.Model(&adminUser).Update("password_hash", string(hashed)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	logAudit(c, "UPDATE", "AdminUser", adminUser.ID, "Changed password")

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

func CreateProject(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project name is required"})
		return
	}

	project := models.Project{Name: body.Name, Status: 1}
	if err := database.DB.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	logAudit(c, "CREATE", "Project", project.ID, project)

	c.JSON(http.StatusOK, gin.H{"data": project})
}

func UpdateProject(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Name   string `json:"name"`
		Status int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var project models.Project
	if err := database.DB.First(&project, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	updates := map[string]interface{}{}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.Status != 0 {
		updates["status"] = body.Status
	}

	if err := database.DB.Model(&project).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	logAudit(c, "UPDATE", "Project", project.ID, updates)

	c.JSON(http.StatusOK, gin.H{"data": project})
}

func AddIncome(c *gin.Context) {
	var income models.Income
	if err := c.ShouldBindJSON(&income); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&income).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add income"})
		return
	}

	logAudit(c, "CREATE", "Income", income.ID, income)

	c.JSON(http.StatusOK, gin.H{"data": income})
}

func AddExpense(c *gin.Context) {
	var expense models.Expense
	if err := c.ShouldBindJSON(&expense); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add expense"})
		return
	}

	logAudit(c, "CREATE", "Expense", expense.ID, expense)

	c.JSON(http.StatusOK, gin.H{"data": expense})
}

func UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Upload file is required"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported file type. Only JPG, PNG, and WebP are allowed."})
		return
	}

	srcFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer srcFile.Close()

	img, _, err := image.Decode(srcFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image file content"})
		return
	}

	// Resize if width is larger than 1280
	if img.Bounds().Dx() > 1280 {
		img = resize.Resize(1280, 0, img, resize.Lanczos3)
	}

	// Save as JPEG
	filename := fmt.Sprintf("%d_%s.jpg", time.Now().UnixNano(), strings.TrimSuffix(filepath.Base(file.Filename), filepath.Ext(file.Filename)))
	outPath := fmt.Sprintf("./uploads/%s", filename)

	out, err := os.Create(outPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create output file"})
		return
	}
	defer out.Close()

	if err := jpeg.Encode(out, img, &jpeg.Options{Quality: 75}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode image"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": fmt.Sprintf("/uploads/%s", filename)})
}

func UpdateIncome(c *gin.Context) {
	id := c.Param("id")
	var income models.Income
	if err := database.DB.First(&income, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Income record not found"})
		return
	}

	var body models.Income
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&income).Updates(body).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update income"})
		return
	}

	logAudit(c, "UPDATE", "Income", income.ID, body)

	c.JSON(http.StatusOK, gin.H{"data": income})
}

func UpdateExpense(c *gin.Context) {
	id := c.Param("id")
	var expense models.Expense
	if err := database.DB.First(&expense, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense record not found"})
		return
	}

	var body models.Expense
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&expense).Updates(body).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update expense"})
		return
	}

	logAudit(c, "UPDATE", "Expense", expense.ID, body)

	c.JSON(http.StatusOK, gin.H{"data": expense})
}

func DeleteIncome(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Income{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete income"})
		return
	}

	parsedID, _ := strconv.Atoi(id)
	logAudit(c, "DELETE", "Income", uint(parsedID), nil)

	c.JSON(http.StatusOK, gin.H{"message": "Income deleted successfully"})
}

func DeleteExpense(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Expense{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete expense"})
		return
	}

	parsedID, _ := strconv.Atoi(id)
	logAudit(c, "DELETE", "Expense", uint(parsedID), nil)

	c.JSON(http.StatusOK, gin.H{"message": "Expense deleted successfully"})
}

func ExportProjectExcel(c *gin.Context) {
	projectId := c.Param("project_id")

	var project models.Project
	if err := database.DB.First(&project, projectId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	var incomes []models.Income
	database.DB.Where("project_id = ?", projectId).Order("created_at desc").Find(&incomes)

	var expenses []models.Expense
	database.DB.Where("project_id = ?", projectId).Order("expense_date desc, created_at desc").Find(&expenses)

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Sheet 1: Incomes
	sheet1 := "收入明细"
	f.SetSheetName("Sheet1", sheet1)
	f.SetCellValue(sheet1, "A1", "交款人姓名")
	f.SetCellValue(sheet1, "B1", "所属组别/村落")
	f.SetCellValue(sheet1, "C1", "金额(元)")
	f.SetCellValue(sheet1, "D1", "交款日期")
	f.SetCellValue(sheet1, "E1", "录入时间")

	for i, inc := range incomes {
		row := i + 2
		f.SetCellValue(sheet1, fmt.Sprintf("A%d", row), inc.VillagerName)
		f.SetCellValue(sheet1, fmt.Sprintf("B%d", row), inc.GroupName)
		f.SetCellValue(sheet1, fmt.Sprintf("C%d", row), inc.Amount)
		f.SetCellValue(sheet1, fmt.Sprintf("D%d", row), inc.PayDate)
		f.SetCellValue(sheet1, fmt.Sprintf("E%d", row), inc.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	// Sheet 2: Expenses
	sheet2 := "支出明细"
	f.NewSheet(sheet2)
	f.SetCellValue(sheet2, "A1", "支出摘要")
	f.SetCellValue(sheet2, "B1", "金额(元)")
	f.SetCellValue(sheet2, "C1", "经手人")
	f.SetCellValue(sheet2, "D1", "发生日期")
	f.SetCellValue(sheet2, "E1", "凭证图片")

	f.SetColWidth(sheet2, "E", "E", 30)

	for i, exp := range expenses {
		row := i + 2
		f.SetCellValue(sheet2, fmt.Sprintf("A%d", row), exp.Title)
		f.SetCellValue(sheet2, fmt.Sprintf("B%d", row), exp.Amount)
		f.SetCellValue(sheet2, fmt.Sprintf("C%d", row), exp.Handler)
		f.SetCellValue(sheet2, fmt.Sprintf("D%d", row), exp.ExpenseDate)

		if exp.ReceiptImg != "" {
			// ReceiptImg is typically "/uploads/filename.ext".
			// We need to point to the local filesystem path.
			localPath := "." + exp.ReceiptImg
			if _, err := os.Stat(localPath); err == nil {
				f.SetRowHeight(sheet2, row, 100)
				f.AddPicture(sheet2, fmt.Sprintf("E%d", row), localPath, &excelize.GraphicOptions{
					OffsetX:         0,
					OffsetY:         0,
					AutoFit:         true,
					LockAspectRatio: false,
					Positioning:     "oneCell",
				})
			} else {
				// Try falling back to stripping the leading slash
				fallbackPath := strings.TrimPrefix(exp.ReceiptImg, "/")
				if _, err := os.Stat(fallbackPath); err == nil {
					f.SetRowHeight(sheet2, row, 100)
					f.AddPicture(sheet2, fmt.Sprintf("E%d", row), fallbackPath, &excelize.GraphicOptions{
						OffsetX:         0,
						OffsetY:         0,
						AutoFit:         true,
						LockAspectRatio: false,
						Positioning:     "oneCell",
					})
				} else {
					f.SetCellValue(sheet2, fmt.Sprintf("E%d", row), "(图片丢失)")
				}
			}
		} else {
			f.SetCellValue(sheet2, fmt.Sprintf("E%d", row), "无")
		}
	}

	// Set active sheet
	f.SetActiveSheet(0)

	// Save to buffer and send
	fileName := fmt.Sprintf("%s_财务明细_%s.xlsx", project.Name, time.Now().Format("20060102"))
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Transfer-Encoding", "binary")

	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate excel file"})
		return
	}
}

func GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset := (page - 1) * limit

	var logs []models.AuditLog
	var total int64

	database.DB.Model(&models.AuditLog{}).Count(&total)
	database.DB.Order("created_at desc").Offset(offset).Limit(limit).Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func DownloadIncomeTemplate(c *gin.Context) {
	f := excelize.NewFile()
	sheet := "Sheet1"
	headers := []string{"交款人姓名", "所属组别", "交款金额", "交款日期(YYYY-MM-DD)"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetColWidth(sheet, "A", "D", 20)
	c.Header("Content-Disposition", "attachment; filename=收入导入模板.xlsx")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	f.Write(c.Writer)
}

func DownloadExpenseTemplate(c *gin.Context) {
	f := excelize.NewFile()
	sheet := "Sheet1"
	headers := []string{"支出明细栏目", "金额", "经手人", "支出日期(YYYY-MM-DD)"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetColWidth(sheet, "A", "D", 20)
	c.Header("Content-Disposition", "attachment; filename=支出导入模板.xlsx")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	f.Write(c.Writer)
}

func ImportIncomes(c *gin.Context) {
	projectIdRaw := c.PostForm("project_id")
	projectId, err := strconv.Atoi(projectIdRaw)
	if err != nil || projectId == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}

	srcFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot open file"})
		return
	}
	defer srcFile.Close()

	f, err := excelize.OpenReader(srcFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid excel file"})
		return
	}

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil || len(rows) <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No data found"})
		return
	}

	var incomes []models.Income
	for i, row := range rows {
		if i == 0 {
			continue // skip header
		}
		if len(row) < 4 {
			continue
		}
		amount, _ := strconv.ParseFloat(row[2], 64)
		if amount == 0 {
			continue
		}
		incomes = append(incomes, models.Income{
			ProjectID:    uint(projectId),
			VillagerName: row[0],
			GroupName:    row[1],
			Amount:       amount,
			PayDate:      row[3],
		})
	}

	if len(incomes) > 0 {
		database.DB.Create(&incomes)
		logAudit(c, "IMPORT", "Income", uint(projectId), fmt.Sprintf("Imported %d incomes", len(incomes)))
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Successfully imported %d records", len(incomes))})
}

func ImportExpenses(c *gin.Context) {
	projectIdRaw := c.PostForm("project_id")
	projectId, err := strconv.Atoi(projectIdRaw)
	if err != nil || projectId == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}

	srcFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot open file"})
		return
	}
	defer srcFile.Close()

	f, err := excelize.OpenReader(srcFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid excel file"})
		return
	}

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil || len(rows) <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No data found"})
		return
	}

	var expenses []models.Expense
	for i, row := range rows {
		if i == 0 {
			continue // skip header
		}
		if len(row) < 4 {
			continue
		}
		amount, _ := strconv.ParseFloat(row[1], 64)
		if amount == 0 {
			continue
		}
		expenses = append(expenses, models.Expense{
			ProjectID:   uint(projectId),
			Title:       row[0],
			Amount:      amount,
			Handler:     row[2],
			ExpenseDate: row[3],
		})
	}

	if len(expenses) > 0 {
		database.DB.Create(&expenses)
		logAudit(c, "IMPORT", "Expense", uint(projectId), fmt.Sprintf("Imported %d expenses", len(expenses)))
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Successfully imported %d records", len(expenses))})
}

func DownloadBackup(c *gin.Context) {
	dbPath := "village-bill.db" // Note: This should match the DB path used in database/sqlite.go

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Database file not found"})
		return
	}

	fileName := fmt.Sprintf("village_bill_backup_%s.db", time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/octet-stream")
	c.File(dbPath)

	logAudit(c, "EXPORT", "System", 0, "Downloaded full database backup")
}
