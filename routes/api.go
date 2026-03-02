package routes

import (
	"village-bill/controllers"
	"village-bill/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// Public
		api.GET("/projects", controllers.GetProjects)
		api.GET("/projects/latest", controllers.GetLatestProject)
		api.GET("/projects/:project_id/stats", controllers.GetProjectStats)
		api.GET("/incomes", controllers.GetIncomes)
		api.GET("/expenses", controllers.GetExpenses)

		// Admin Base
		admin := api.Group("/admin")
		{
			admin.POST("/login", controllers.Login)

			// Admin Protected
			protected := admin.Group("/")
			protected.Use(middleware.AuthRequired())
			{
				protected.PUT("/password", controllers.ChangePassword)

				protected.POST("/projects", controllers.CreateProject)
				protected.PUT("/projects/:id", controllers.UpdateProject)
				protected.GET("/projects/:project_id/export", controllers.ExportProjectExcel)
				protected.POST("/incomes", controllers.AddIncome)
				protected.PUT("/incomes/:id", controllers.UpdateIncome)
				protected.DELETE("/incomes/:id", controllers.DeleteIncome)
				protected.POST("/expenses", controllers.AddExpense)
				protected.PUT("/expenses/:id", controllers.UpdateExpense)
				protected.DELETE("/expenses/:id", controllers.DeleteExpense)
				protected.POST("/upload", controllers.UploadImage)
				protected.GET("/audit_logs", controllers.GetAuditLogs)
				protected.GET("/incomes/template", controllers.DownloadIncomeTemplate)
				protected.GET("/expenses/template", controllers.DownloadExpenseTemplate)
				protected.POST("/incomes/import", controllers.ImportIncomes)
				protected.POST("/expenses/import", controllers.ImportExpenses)
				protected.GET("/backup", controllers.DownloadBackup)
			}
		}
	}
}
