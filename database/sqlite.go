package database

import (
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"village-bill/models"

	"golang.org/x/crypto/bcrypt"
)

var DB *gorm.DB

func InitDB(dsn string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("failed to connect database", err)
	}

	err = DB.AutoMigrate(
		&models.Project{},
		&models.Income{},
		&models.Expense{},
		&models.AdminUser{},
		&models.AuditLog{},
	)
	if err != nil {
		log.Fatal("failed to auto migrate", err)
	}

	// Seed default admin user
	var count int64
	DB.Model(&models.AdminUser{}).Count(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		defaultAdmin := models.AdminUser{
			Username:     "admin",
			PasswordHash: string(hash),
		}
		if err := DB.Create(&defaultAdmin).Error; err != nil {
			log.Printf("Failed to create default admin user: %v", err)
		} else {
			log.Println("Created default admin user (admin/admin123)")
		}
	}
}
