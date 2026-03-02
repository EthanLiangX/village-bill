package models

import "gorm.io/gorm"

type AuditLog struct {
	gorm.Model
	AdminUsername string `gorm:"type:varchar;not null" json:"admin_username"`
	Action        string `gorm:"type:varchar;not null" json:"action"`      // CREATE, UPDATE, DELETE
	EntityType    string `gorm:"type:varchar;not null" json:"entity_type"` // Income, Expense, Project
	EntityID      uint   `gorm:"not null" json:"entity_id"`
	Details       string `gorm:"type:text" json:"details"` // JSON representation of changes or snapshot
}
