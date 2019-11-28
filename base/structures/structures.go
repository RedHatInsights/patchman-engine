package structures

import (
    "time"
)

type SystemDAO struct {
	ID              int        `gorm:"column:id"`
	Request         string     `gorm:"column:"`
	Checksum        string     `gorm:"column:"`
	Updated         time.Time  `gorm:"column:"`
}

// db table name, for gorm
func (SystemDAO) TableName() string {
	return "system_platform"
}
