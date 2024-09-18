package xtrememodel

type Migration struct {
	BaseModelWithoutID
	Reference string `gorm:"column:reference;type:varchar(250)"`
}

func (Migration) TableName() string {
	return "migrations"
}
