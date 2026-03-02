package models

import "gorm.io/gorm"

type AdminUser struct {
	gorm.Model
	Username     string `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"not null" json:"-"`
}
