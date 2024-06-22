package xtrememodel

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"os"
	"time"
)

/* --- BASE MODEL CONFIGURATION --- */

type BaseModel struct {
	ID        string         `gorm:"type:varchar(45);primarykey"`
	Timezone  string         `gorm:"column:timezone;type:varchar(50)"`
	CreatedAt time.Time      `gorm:"column:createdAt;type:timestamp"`
	UpdatedAt time.Time      `gorm:"column:updatedAt;type:timestamp"`
	DeletedAt gorm.DeletedAt `gorm:"column:deletedAt;index"`
}

func (m *BaseModel) BeforeCreate(tx *gorm.DB) error {
	m.setID()

	if m.CreatedAt == (time.Time{}) {
		m.CreatedAt = time.Now()
	}

	if m.UpdatedAt == (time.Time{}) {
		m.UpdatedAt = time.Now()
	}

	if len(m.Timezone) == 0 {
		m.Timezone = m.CreatedAt.Location().String()
	}

	return nil
}

func (m *BaseModel) BeforeSave(tx *gorm.DB) error {
	if m.CreatedAt == (time.Time{}) {
		m.CreatedAt = time.Now()
	}

	if m.UpdatedAt == (time.Time{}) {
		m.UpdatedAt = time.Now()
	}

	if len(m.Timezone) == 0 {
		m.Timezone = m.CreatedAt.Location().String()
	}

	return nil
}

func (m *BaseModel) BeforeUpdate(tx *gorm.DB) error {
	if m.UpdatedAt == (time.Time{}) {
		m.UpdatedAt = time.Now()
	}

	return nil
}

func (m *BaseModel) setID() {
	if len(m.ID) == 0 {
		uniqueId := os.Getenv("UNIQUE_ID")
		if len(uniqueId) == 0 {
			uniqueId = "127001"
		}

		m.ID = fmt.Sprintf("%s-%s", uniqueId, uuid.New().String())
	}
}

/* --- BASE MODEL WITHOUT ID CONFIGURATION --- */

type BaseModelWithoutID struct {
	Timezone  string         `gorm:"column:timezone;type:varchar(50)"`
	CreatedAt time.Time      `gorm:"column:createdAt;type:timestamp"`
	UpdatedAt time.Time      `gorm:"column:updatedAt;type:timestamp"`
	DeletedAt gorm.DeletedAt `gorm:"column:deletedAt;index"`
}

func (m *BaseModelWithoutID) BeforeCreate(tx *gorm.DB) error {
	if m.CreatedAt == (time.Time{}) {
		m.CreatedAt = time.Now()
	}

	if m.UpdatedAt == (time.Time{}) {
		m.UpdatedAt = time.Now()
	}

	if len(m.Timezone) == 0 {
		m.Timezone = m.CreatedAt.Location().String()
	}

	return nil
}

func (m *BaseModelWithoutID) BeforeSave(tx *gorm.DB) error {
	if m.CreatedAt == (time.Time{}) {
		m.CreatedAt = time.Now()
	}

	if m.UpdatedAt == (time.Time{}) {
		m.UpdatedAt = time.Now()
	}

	if len(m.Timezone) == 0 {
		m.Timezone = m.CreatedAt.Location().String()
	}

	return nil
}

func (m *BaseModelWithoutID) BeforeUpdate(tx *gorm.DB) error {
	if m.UpdatedAt == (time.Time{}) {
		m.UpdatedAt = time.Now()
	}

	return nil
}

/* --- BASE MODEL CONFIGURATION --- */

type RabbitMQBaseModel struct {
	ID        uint           `gorm:"primarykey"`
	CreatedAt time.Time      `gorm:"column:createdAt;type:timestamp"`
	UpdatedAt time.Time      `gorm:"column:updatedAt;type:timestamp"`
	DeletedAt gorm.DeletedAt `gorm:"column:deletedAt;index"`
}

func (m *RabbitMQBaseModel) BeforeCreate(tx *gorm.DB) error {
	if m.CreatedAt == (time.Time{}) {
		m.CreatedAt = time.Now()
	}

	if m.UpdatedAt == (time.Time{}) {
		m.UpdatedAt = time.Now()
	}

	return nil
}

func (m *RabbitMQBaseModel) BeforeSave(tx *gorm.DB) error {
	if m.CreatedAt == (time.Time{}) {
		m.CreatedAt = time.Now()
	}

	if m.UpdatedAt == (time.Time{}) {
		m.UpdatedAt = time.Now()
	}

	return nil
}

func (m *RabbitMQBaseModel) BeforeUpdate(tx *gorm.DB) error {
	if m.UpdatedAt == (time.Time{}) {
		m.UpdatedAt = time.Now()
	}

	return nil
}

/* --- COLUMN TYPE CONFIGURATION: TIME --- */

type TimeColumn struct {
	time.Time
}

func (timeColumn *TimeColumn) Scan(value interface{}) error {
	scannedTime, err := time.Parse("15:04:05", value.(string))
	if err == nil {
		*timeColumn = TimeColumn{scannedTime}
	}

	return err
}

func (TimeColumn) GormDataType() string {
	return "time"
}

func (TimeColumn) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return "time"
}

func (timeColumn TimeColumn) Value() (driver.Value, error) {
	if !timeColumn.IsZero() {
		return timeColumn.Time.Format("15:04:05"), nil
	} else {
		return nil, nil
	}
}

/* --- COLUMN TYPE CONFIGURATION: OBJECT / MAP IN ARRAY --- */

type ArrayMapInterfaceColumn []map[string]interface{}

func (j *ArrayMapInterfaceColumn) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	var result []map[string]interface{}
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

func (j ArrayMapInterfaceColumn) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}

	return json.Marshal(j)
}

type MapInterfaceColumn map[string]interface{}

func (j *MapInterfaceColumn) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	var result map[string]interface{}
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

func (j MapInterfaceColumn) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}

	return json.Marshal(j)
}

type MapBoolColumn map[string]bool

func (j *MapBoolColumn) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	var result map[string]bool
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

func (j MapBoolColumn) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}

	return json.Marshal(j)
}

type ArrayStringColumn []string

func (j *ArrayStringColumn) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	var result []string
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

func (j ArrayStringColumn) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}

	return json.Marshal(j)
}

type ArrayIntColumn []int

func (j *ArrayIntColumn) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	var result []int
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

func (j ArrayIntColumn) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}

	return json.Marshal(j)
}

type ArrayUintColumn []uint

func (j *ArrayUintColumn) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	var result []uint
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

func (j ArrayUintColumn) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}

	return json.Marshal(j)
}

type ArrayBoolColumn []bool

func (j *ArrayBoolColumn) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	var result []bool
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

func (j ArrayBoolColumn) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}

	return json.Marshal(j)
}
