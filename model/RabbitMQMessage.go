package xtrememodel

type RabbitMQMessage struct {
	RabbitMQBaseModel
	ConnectionId  uint               `gorm:"column:connectionId;type:bigint;null"`
	Exchange      string             `gorm:"column:exchange;type:varchar(255);null"`
	Queue         string             `gorm:"column:queue;type:varchar(255);null"`
	SenderId      *uint              `gorm:"column:senderId;type:int;default:null"`
	SenderType    *string            `gorm:"column:senderType;type:varchar(255);default:null"`
	SenderService string             `gorm:"column:senderService;type:varchar(255);default:null"`
	Payload       MapInterfaceColumn `gorm:"column:payload;type:json;default:null"`
	Finished      bool               `gorm:"column:finished;type:tinyint;default:0"`
	Resend        float64            `gorm:"column:resend;type:decimal(8,2);default:0"`
	CreatedBy     *string            `gorm:"column:createdBy;type:char(255);default:null"`
	CreatedByName *string            `gorm:"column:createdByName;type:varchar(255);default:null"`
}

func (RabbitMQMessage) TableName() string {
	return "messages"
}
