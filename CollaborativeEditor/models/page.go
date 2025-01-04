// models/page.go
package models

import "time"

// Page model
type Page struct {
	ID        uint      `gorm:"primaryKey"`
	FileName  string    `gorm:"not null"`
	ParentID  *uint     `gorm:"index"`
	Level     int       `gorm:"default:0"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
	Content   string    `gorm:"type:text"`
}
