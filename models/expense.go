package models

import "gorm.io/gorm"

type Expense struct {
	gorm.Model
	ProjectID   uint    `gorm:"not null" json:"project_id"`
	Title       string  `gorm:"type:varchar;not null" json:"title"`
	Amount      float64 `gorm:"type:decimal(10,2);not null" json:"amount"`
	Handler     string  `gorm:"type:varchar;not null" json:"handler"`
	ExpenseDate string  `gorm:"type:date;not null" json:"expense_date"`
	ReceiptImg  string  `gorm:"type:varchar" json:"receipt_img"`
}
