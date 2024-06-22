package xtrememodel

type RabbitMQMessage struct {
	RabbitMQBaseModel
	Exchange       string             `gorm:"column:exchange;type:varchar(250);not null"`
	Key            string             `gorm:"column:key;type:varchar(250);not null"`
	QueueSender    string             `gorm:"column:queueSender;type:varchar(250);not null"`
	QueueConsumers ArrayStringColumn  `gorm:"column:queueConsumers;type:json;not null"`
	SenderId       *uint              `gorm:"column:senderId;type:int;default:null"`
	SenderType     *string            `gorm:"column:senderType;type:varchar(250);default:null"`
	Payload        MapInterfaceColumn `gorm:"column:payload;type:json;default:null"`
	Finished       bool               `gorm:"column:finished;type:tinyint;default:0"`
	Statuses       MapBoolColumn      `gorm:"column:statuses;type:json;default:null"`
	Resend         float64            `gorm:"column:resend;type:decimal(8,2);default:0"`
	CreatedBy      *string            `gorm:"column:createdBy;type:char(255);default:null"`
	CreatedByName  *string            `gorm:"column:createdByName;type:varchar(255);default:null"`
}

func (RabbitMQMessage) TableName() string {
	return "messages"
}
