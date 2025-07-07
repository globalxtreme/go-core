package xtrememodel

type RabbitMQAsyncWorkflow struct {
	RabbitMQBaseModel
	Action           string  `gorm:"column:action;type:varchar(255);null"`
	StatusId         int     `gorm:"column:statusId;type:tinyint"`
	ReferenceId      string  `gorm:"column:senderId;type:char(45);default:not null"`
	ReferenceType    string  `gorm:"column:senderType;type:varchar(255);default:not null"`
	ReferenceService string  `gorm:"column:senderService;type:varchar(255);default:null"`
	TotalStep        int     `gorm:"column:totalStep;type:int"`
	Reprocessed      float64 `gorm:"column:reprocessed;type:decimal(8,2);default:0"`
	CreatedBy        *string `gorm:"column:createdBy;type:char(255);default:null"`
	CreatedByName    *string `gorm:"column:createdByName;type:varchar(255);default:null"`
}

func (RabbitMQAsyncWorkflow) TableName() string {
	return "async_workflows"
}
