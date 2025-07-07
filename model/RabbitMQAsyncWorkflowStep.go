package xtrememodel

type RabbitMQAsyncWorkflowStep struct {
	RabbitMQBaseModel
	WorkflowId  uint                     `gorm:"column:workflowId;type:bigint"`
	Service     string                   `gorm:"column:service;type:varchar(100);not null"`
	Queue       string                   `gorm:"column:queue;type:varchar(200);not null"`
	StepOrder   int                      `gorm:"column:stepOrder;type:int"`
	StatusId    int                      `gorm:"column:statusId;type:tinyint"`
	Description string                   `gorm:"column:description;type:text;null"`
	Payload     *MapInterfaceColumn      `gorm:"column:payload;type:json;default:null"`
	Errors      *ArrayMapInterfaceColumn `gorm:"column:errors;type:json;default:null"`
	Response    *MapInterfaceColumn      `gorm:"column:response;type:json;default:null"`
	Reprocessed float64                  `gorm:"column:reprocessed;type:int;default:0"`
}

func (RabbitMQAsyncWorkflowStep) TableName() string {
	return "async_workflow_steps"
}
