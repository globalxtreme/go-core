package xtrememodel

type RabbitMQAsyncWorkflow struct {
	RabbitMQBaseModel
	Action           string  `gorm:"column:action;type:varchar(150);null"`
	StatusId         int     `gorm:"column:statusId;type:tinyint"`
	ReferenceId      string  `gorm:"column:referenceId;type:varchar(45);not null"`
	ReferenceType    string  `gorm:"column:referenceType;type:varchar(200);not null"`
	ReferenceService string  `gorm:"column:referenceService;type:varchar(100);null"`
	TotalStep        int     `gorm:"column:totalStep;type:int"`
	Reprocessed      int     `gorm:"column:reprocessed;type:int;default:0"`
	SuccessMessage   string  `gorm:"column:successMessage;type:text;null"`
	CreatedBy        *string `gorm:"column:createdBy;type:char(36);null"`
	CreatedByName    *string `gorm:"column:createdByName;type:varchar(255);null"`
}

func (RabbitMQAsyncWorkflow) TableName() string {
	return "async_workflows"
}
