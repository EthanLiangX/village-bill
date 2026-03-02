package models

import "gorm.io/gorm"

type Project struct {
	gorm.Model
	Name   string `gorm:"type:varchar;not null" json:"name"`
	Status int    `gorm:"type:integer;not null;default:1" json:"status"` // 1:筹款中, 2:已结项
}
