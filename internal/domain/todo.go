package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// TodoStatus type
type TodoStatus string

const (
	// TodoStatusInProgress const
	TodoStatusInProgress TodoStatus = "IN_PROGRESS"
	// TodoStatusComplete const
	TodoStatusComplete TodoStatus = "COMPLETE"
)

// Todo struct - Core domain entity
type Todo struct {
	ID          *uuid.UUID      `gorm:"type:uuid;primary_key;"`
	Title       *string         `gorm:"type:varchar(100);not null;"`
	Description *string         `gorm:"type:TEXT"`
	Date        *time.Time      `gorm:"type:timestamp;not null;"`
	Image       *string         `gorm:"type:text"`
	Status      *TodoStatus     `gorm:"type:varchar(11);not null;"`
	CreatedAt   *time.Time      `gorm:"type:timestamp"`
	UpdatedAt   *time.Time      `gorm:"type:timestamp"`
	DeletedAt   *gorm.DeletedAt `gorm:"type:timestamp"`
}

// TableName func
func (t *Todo) TableName() string {
	return "todos"
}

// BeforeCreate hook - generates UUID before creating
func (t *Todo) BeforeCreate(tx *gorm.DB) (err error) {
	logrus.Info("BeforeCreate")
	uuid, err := uuid.NewRandom() // v4
	if err != nil {
		return err
	}
	t.ID = &uuid
	return nil
}

// MigrateDatabase func - Auto-migrate database schema
func MigrateDatabase(db *gorm.DB) {
	if db == nil {
		panic("An error when connect database")
	}

	err := db.AutoMigrate(&Todo{})
	if err != nil {
		panic(err)
	}
}
