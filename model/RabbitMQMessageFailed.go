package xtrememodel

type RabbitMQMessageFailed struct {
	RabbitMQBaseModel
	ConnectionId uint               `gorm:"column:connectionId;type:bigint;null"`
	MessageId    uint               `gorm:"column:messageId;type:bigint;not null"`
	Service      string             `gorm:"column:service;type:varchar(255);not null"`
	Exchange     string             `gorm:"column:exchange;type:varchar(255);null"`
	Queue        string             `gorm:"column:queue;type:varchar(255);null"`
	Payload      []byte             `gorm:"column:payload;type:json;default:null"`
	Exception    MapInterfaceColumn `gorm:"column:exception;type:json;default:null"`
	Resend       bool               `gorm:"column:resend;type:boolean;default:0"`
}

func (RabbitMQMessageFailed) TableName() string {
	return "message_faileds"
}
