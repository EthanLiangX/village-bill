package models

import "gorm.io/gorm"

type Income struct {
	gorm.Model
	ProjectID    uint    `gorm:"not null" json:"project_id"`
	VillagerName string  `gorm:"type:varchar;not null" json:"villager_name"`
	GroupName    string  `gorm:"type:varchar;not null" json:"group_name"`
	Amount       float64 `gorm:"type:decimal(10,2);not null" json:"amount"`
	PayDate      string  `gorm:"type:date;not null" json:"pay_date"`
}
