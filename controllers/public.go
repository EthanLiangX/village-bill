package controllers

import (
	"net/http"
	"strconv"

	"village-bill/database"
	"village-bill/models"

	"github.com/gin-gonic/gin"
)

func GetProjects(c *gin.Context) {
	var projects []models.Project
	database.DB.Order("created_at desc").Find(&projects)
	c.JSON(http.StatusOK, gin.H{"data": projects})
}

func GetLatestProject(c *gin.Context) {
	var project models.Project
	if err := database.DB.Order("created_at desc").First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No projects found"})
		return
	}

	var totalIncome float64
	var totalExpense float64

	database.DB.Model(&models.Income{}).Where("project_id = ?", project.ID).Select("COALESCE(SUM(amount), 0)").Scan(&totalIncome)
	database.DB.Model(&models.Expense{}).Where("project_id = ?", project.ID).Select("COALESCE(SUM(amount), 0)").Scan(&totalExpense)

	c.JSON(http.StatusOK, gin.H{
		"project_id":    project.ID,
		"project_name":  project.Name,
		"status":        project.Status,
		"total_income":  totalIncome,
		"total_expense": totalExpense,
		"balance":       totalIncome - totalExpense,
	})
}

func GetIncomes(c *gin.Context) {
	projectId := c.Query("project_id")
	if projectId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 {
		limit = 50
	} else if limit > 1000 {
		limit = 1000
	}
	offset := (page - 1) * limit

	var incomes []models.Income
	var total int64
	var totalAmount float64

	database.DB.Model(&models.Income{}).Where("project_id = ?", projectId).Count(&total)
	database.DB.Model(&models.Income{}).Where("project_id = ?", projectId).Select("COALESCE(SUM(amount), 0)").Scan(&totalAmount)
	database.DB.Where("project_id = ?", projectId).Order("created_at desc").Offset(offset).Limit(limit).Find(&incomes)

	c.JSON(http.StatusOK, gin.H{
		"data": incomes,
		"meta": gin.H{
			"total":        total,
			"total_amount": totalAmount,
			"page":         page,
			"limit":        limit,
		},
	})
}

func GetExpenses(c *gin.Context) {
	projectId := c.Query("project_id")
	if projectId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 {
		limit = 50
	} else if limit > 1000 {
		limit = 1000
	}
	offset := (page - 1) * limit

	var expenses []models.Expense
	var total int64
	var totalAmount float64

	database.DB.Model(&models.Expense{}).Where("project_id = ?", projectId).Count(&total)
	database.DB.Model(&models.Expense{}).Where("project_id = ?", projectId).Select("COALESCE(SUM(amount), 0)").Scan(&totalAmount)
	database.DB.Where("project_id = ?", projectId).Order("expense_date desc, created_at desc").Offset(offset).Limit(limit).Find(&expenses)

	c.JSON(http.StatusOK, gin.H{
		"data": expenses,
		"meta": gin.H{
			"total":        total,
			"total_amount": totalAmount,
			"page":         page,
			"limit":        limit,
		},
	})
}

func GetProjectStats(c *gin.Context) {
	projectId := c.Param("project_id")
	if projectId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return
	}

	type MonthlyStat struct {
		Month string  `json:"month"`
		Total float64 `json:"total"`
	}

	var incomeStats []MonthlyStat
	database.DB.Model(&models.Income{}).
		Where("project_id = ?", projectId).
		Select("substr(pay_date, 1, 7) as month, SUM(amount) as total").
		Group("substr(pay_date, 1, 7)").
		Order("month asc").
		Scan(&incomeStats)

	var expenseStats []MonthlyStat
	database.DB.Model(&models.Expense{}).
		Where("project_id = ?", projectId).
		Select("substr(expense_date, 1, 7) as month, SUM(amount) as total").
		Group("substr(expense_date, 1, 7)").
		Order("month asc").
		Scan(&expenseStats)

	var totalIncome float64
	var totalExpense float64
	database.DB.Model(&models.Income{}).Where("project_id = ?", projectId).Select("COALESCE(SUM(amount), 0)").Scan(&totalIncome)
	database.DB.Model(&models.Expense{}).Where("project_id = ?", projectId).Select("COALESCE(SUM(amount), 0)").Scan(&totalExpense)

	c.JSON(http.StatusOK, gin.H{
		"project_id":    projectId,
		"incomes":       incomeStats,
		"expenses":      expenseStats,
		"total_income":  totalIncome,
		"total_expense": totalExpense,
		"balance":       totalIncome - totalExpense,
	})
}
