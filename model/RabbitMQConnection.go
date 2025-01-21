package xtrememodel

type RabbitMQConnection struct {
	RabbitMQBaseModel
	Connection string `gorm:"column:connection;type:varchar(50);null"`
	Service    string `gorm:"column:service;type:varchar(150);null"`
}

func (RabbitMQConnection) TableName() string {
	return "connections"
}
