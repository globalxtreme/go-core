package xtremerabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/globalxtreme/go-core/v2/api/privateapi"
	xtrememodel "github.com/globalxtreme/go-core/v2/model"
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	xtremews "github.com/globalxtreme/go-core/v2/websocket"
	"github.com/rabbitmq/amqp091-go"
	"log"
	"os/exec"
	"runtime/debug"
	"strconv"
	"time"
)

type GXAsyncWorkflow struct {
	Strict         bool
	Action         string
	Description    string
	ReferenceId    string
	ReferenceType  string
	SuccessMessage string
	ErrorMessage   string
	CreatedBy      *string
	CreatedByName  *string

	totalStep int
	firstStep GXAsyncWorkflowStepOpt
	steps     []GXAsyncWorkflowStepOpt
}

type GXAsyncWorkflowStepOpt struct {
	Service     string
	Queue       string
	Description string
	Payload     interface{}

	stepOrder int
}

func (flow *GXAsyncWorkflow) OnAction(action string) {
	flow.Action = action
}

func (flow *GXAsyncWorkflow) OnStep(opt GXAsyncWorkflowStepOpt) {
	flow.totalStep++

	opt.stepOrder = flow.totalStep

	if opt.Queue == "" {
		opt.Queue = fmt.Sprintf("%s.%s.async-workflow", opt.Service, flow.Action)
	}

	flow.steps = append(flow.steps, opt)

	if flow.totalStep == 1 {
		flow.firstStep = opt
	}
}

func (flow *GXAsyncWorkflow) OnReference(referenceId any, referenceType string) {
	var strReferenceId string
	switch referenceId.(type) {
	case string:
		strReferenceId = referenceId.(string)
	case uint:
		strReferenceId = strconv.Itoa(int(referenceId.(uint)))
	case int:
		strReferenceId = strconv.Itoa(referenceId.(int))
	}

	flow.ReferenceId = strReferenceId
	flow.ReferenceType = referenceType
}

func (flow *GXAsyncWorkflow) SetCreatedBy(createdBy string, createdByName string) {
	flow.CreatedBy = &createdBy
	flow.CreatedByName = &createdByName
}

func (flow *GXAsyncWorkflow) SetSuccessMessage(message string) {
	flow.SuccessMessage = message
}

func (flow *GXAsyncWorkflow) SetErrorMessage(message string) {
	flow.ErrorMessage = message
}

func (flow *GXAsyncWorkflow) Push() {
	if len(flow.steps) == 0 {
		log.Panicf("Please setup your workflow step")
	}

	_, ok := flow.firstStep.Payload.(map[string]interface{})
	if !ok {
		log.Panicf("Please setup your payload to step order (%d)", flow.firstStep.stepOrder)
	}

	if flow.Strict {
		var countWorkflow int64
		err := RabbitMQSQL.Model(&xtrememodel.RabbitMQAsyncWorkflow{}).
			Where(`action = ? AND referenceId = ? AND referenceType = ? AND referenceService = ?`, flow.Action, flow.ReferenceId, flow.ReferenceType, xtremepkg.GetServiceName()).
			Where(`statusId != ?`, RABBITMQ_ASYNC_WORKFLOW_STATUS_SUCCESS_ID).
			Count(&countWorkflow).Error
		if err != nil || countWorkflow > 0 {
			log.Panicf("You have an asynchronous workflow not yet finished. Please check your workflow status and reprocess")
		}
	}

	workflow := xtrememodel.RabbitMQAsyncWorkflow{
		Action:           flow.Action,
		Description:      flow.Description,
		StatusId:         RABBITMQ_ASYNC_WORKFLOW_STATUS_PENDING_ID,
		ReferenceId:      flow.ReferenceId,
		ReferenceType:    flow.ReferenceType,
		ReferenceService: xtremepkg.GetServiceName(),
		SuccessMessage:   flow.SuccessMessage,
		ErrorMessage:     flow.ErrorMessage,
		TotalStep:        flow.totalStep,
		CreatedBy:        flow.CreatedBy,
		CreatedByName:    flow.CreatedByName,
	}
	err := RabbitMQSQL.Create(&workflow).Error
	if err != nil {
		log.Panicf("Unable to create async workflow: %s", err.Error())
	}

	workflowSteps := make([]xtrememodel.RabbitMQAsyncWorkflowStep, 0)
	for _, step := range flow.steps {
		workflowStep := xtrememodel.RabbitMQAsyncWorkflowStep{
			WorkflowId:  workflow.ID,
			Service:     step.Service,
			Queue:       step.Queue,
			StepOrder:   step.stepOrder,
			StatusId:    RABBITMQ_ASYNC_WORKFLOW_STATUS_PENDING_ID,
			Description: step.Description,
		}

		payload, ok := step.Payload.(map[string]interface{})
		if ok {
			if step.stepOrder == 1 {
				workflowStep.Payload = (*xtrememodel.MapInterfaceColumn)(&payload)
			} else {
				workflowStep.ForwardPayload = (*xtrememodel.MapInterfaceColumn)(&map[string]interface{}{
					flow.Action: payload,
				})
			}
		}

		workflowSteps = append(workflowSteps, workflowStep)
	}

	err = RabbitMQSQL.Create(&workflowSteps).Error
	if err != nil {
		log.Panicf("Unable to create workflow steps: %s", err.Error())
	}

	sendToMonitoringEvent(workflow)

	pushWorkflowMessage(workflow.ID, flow.firstStep.Queue, flow.firstStep.Payload)
}

type AsyncWorkflowConsumerInterface interface {
	setAction(action string)
	setReferenceId(referenceId string)
	setReferenceType(referenceType string)
	setReferenceService(referenceService string)

	GetAction() string
	GetReferenceId() string
	GetReferenceType() string
	GetReferenceService() string
	Consume(payload interface{}) (interface{}, error, []byte)
	Response(payload interface{}, data ...interface{}) interface{}
}

type AsyncWorkflowForwardPayloadInterface interface {
	ForwardPayload() []AsyncWorkflowForwardPayloadResult
}

type AsyncWorkflowForwardPayloadResult struct {
	Queue   string
	Payload interface{}
}

type AsyncWorkflowConsumeOpt struct {
	Queue    string
	Consumer AsyncWorkflowConsumerInterface
}

type asyncWorkflowBody struct {
	WorkflowId uint `json:"workflowId"`
	Data       any  `json:"data"`
}

type AsyncWorkflowConsumerBase struct {
	action           string
	referenceId      string
	referenceType    string
	referenceService string
}

func (b *AsyncWorkflowConsumerBase) setAction(action string) {
	b.action = action
}

func (b *AsyncWorkflowConsumerBase) setReferenceId(referenceId string) {
	b.referenceId = referenceId
}

func (b *AsyncWorkflowConsumerBase) setReferenceType(referenceType string) {
	b.referenceType = referenceType
}

func (b *AsyncWorkflowConsumerBase) setReferenceService(referenceService string) {
	b.referenceService = referenceService
}

func (b *AsyncWorkflowConsumerBase) GetAction() string {
	return b.action
}

func (b *AsyncWorkflowConsumerBase) GetReferenceId() string {
	return b.referenceId
}

func (b *AsyncWorkflowConsumerBase) GetReferenceType() string {
	return b.referenceType
}

func (b *AsyncWorkflowConsumerBase) GetReferenceService() string {
	return b.referenceService
}

func ConsumeWorkflow(options []AsyncWorkflowConsumeOpt) {
	mqConnection, ok := RabbitMQConnectionCache[RABBITMQ_CONNECTION_GLOBAL]
	if !ok {
		if len(RabbitMQConnectionCache) == 0 {
			RabbitMQConnectionCache = make(map[string]xtrememodel.RabbitMQConnection)
		}

		err := RabbitMQSQL.Where("connection = ?", RABBITMQ_CONNECTION_GLOBAL).First(&mqConnection).Error
		if err != nil || mqConnection.ID == 0 {
			log.Panicf("Data connection does not exists: %s", err)
		}

		RabbitMQConnectionCache[RABBITMQ_CONNECTION_GLOBAL] = mqConnection
	}

	connConf := RabbitMQConf.Connection[RABBITMQ_CONNECTION_GLOBAL]
	conn, err := amqp091.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", connConf.Username, connConf.Password, connConf.Host, connConf.Port))
	if err != nil {
		log.Panicf("Failed to connect to RabbitMQ: %s", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Panicf("Failed to open a channel: %s", err)
	}
	defer ch.Close()

	var forever chan struct{}

	for _, opt := range options {
		q, err := ch.QueueDeclare(
			opt.Queue,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Panicf("Failed to declare a queue: %s", err)
		}

		err = ch.Qos(
			1,
			0,
			false,
		)
		if err != nil {
			log.Panicf("Failed to set QoS: %s", err)
		}

		msgs, err := ch.Consume(
			q.Name,
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Panicf("Failed to register a consumer: %s", err)
		}

		go func() {
			for d := range msgs {
				processWorkflow(opt, d.Body)
			}
		}()
	}

	log.Printf(" [*] Waiting for logs. To exit press CTRL+C")
	<-forever
}

func processWorkflow(opt AsyncWorkflowConsumeOpt, body []byte) {
	log.Printf("CONSUMING: %s %s", printMessage(opt.Queue), time.DateTime)

	var mqBody asyncWorkflowBody
	err := json.Unmarshal(body, &mqBody)
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Error unmarshalling: %s", err), true)
		return
	}

	var workflow xtrememodel.RabbitMQAsyncWorkflow
	err = RabbitMQSQL.First(&workflow, mqBody.WorkflowId).Error
	if err != nil {
		failedWorkflow("Get async workflow data is failed", err, debug.Stack(), nil, nil)
		return
	}

	var workflowStep xtrememodel.RabbitMQAsyncWorkflowStep
	err = RabbitMQSQL.Where("workflowId = ? AND queue = ?", mqBody.WorkflowId, opt.Queue).First(&workflowStep).Error
	if err != nil {
		failedWorkflow(fmt.Sprintf("Get async workflow step data (%s) is failed", opt.Queue), err, debug.Stack(), &workflow, nil)
		return
	}

	opt.Consumer.setAction(workflow.Action)
	opt.Consumer.setReferenceId(workflow.ReferenceId)
	opt.Consumer.setReferenceType(workflow.ReferenceType)
	opt.Consumer.setReferenceService(workflow.ReferenceService)

	processingWorkflow(&workflow, &workflowStep)

	var result interface{}
	if workflowStep.StatusId != RABBITMQ_ASYNC_WORKFLOW_STATUS_SUCCESS_ID {
		var trace []byte

		result, err, trace = opt.Consumer.Consume(mqBody.Data)
		if err != nil {
			failedWorkflow(fmt.Sprintf("Process in action (%s) and step (%d) is failed", workflow.Action, workflowStep.StepOrder), err, trace, &workflow, &workflowStep)
			return
		}
	} else {
		result = opt.Consumer.Response(mqBody.Data)
	}

	var forwardPayloads []AsyncWorkflowForwardPayloadResult
	if forwarder, ok := opt.Consumer.(AsyncWorkflowForwardPayloadInterface); ok {
		forwardPayloads = forwarder.ForwardPayload()
	}

	finishWorkflow(workflow, workflowStep, result, forwardPayloads)

	log.Printf("%-10s %s %s", "SUCCESS:", printMessage(opt.Queue), time.DateTime)
}

func processingWorkflow(workflow *xtrememodel.RabbitMQAsyncWorkflow, workflowStep *xtrememodel.RabbitMQAsyncWorkflowStep) {
	if workflow.StatusId != RABBITMQ_ASYNC_WORKFLOW_STATUS_PROCESSING_ID {
		workflow.StatusId = RABBITMQ_ASYNC_WORKFLOW_STATUS_PROCESSING_ID

		err := RabbitMQSQL.Where("id = ?", workflow.ID).
			Updates(&xtrememodel.RabbitMQAsyncWorkflow{
				StatusId: RABBITMQ_ASYNC_WORKFLOW_STATUS_PROCESSING_ID,
			}).Error
		if err != nil {
			xtremepkg.LogError(fmt.Sprintf("Unable to update async workflow to processing: %s", err), true)
		}
	}

	if workflowStep.StatusId != RABBITMQ_ASYNC_WORKFLOW_STATUS_PROCESSING_ID {
		workflowStep.StatusId = RABBITMQ_ASYNC_WORKFLOW_STATUS_PROCESSING_ID

		err := RabbitMQSQL.Where("id = ?", workflowStep.ID).
			Updates(&xtrememodel.RabbitMQAsyncWorkflowStep{
				StatusId: RABBITMQ_ASYNC_WORKFLOW_STATUS_PROCESSING_ID,
			}).Error
		if err != nil {
			xtremepkg.LogError(fmt.Sprintf("Unable to update async workflow step to processing: %s", err), true)
		}
	}

	sendToMonitoringActionEvent(*workflow, *workflowStep)
}

func finishWorkflow(workflow xtrememodel.RabbitMQAsyncWorkflow, workflowStep xtrememodel.RabbitMQAsyncWorkflowStep, result interface{}, forwardPayloads []AsyncWorkflowForwardPayloadResult) {
	var stepResponse *map[string]interface{}
	if stepResponseMap, ok := result.(map[string]interface{}); ok && len(stepResponseMap) > 0 {
		stepResponse = &stepResponseMap
	}

	workflowStep.StatusId = RABBITMQ_ASYNC_WORKFLOW_STATUS_SUCCESS_ID
	workflowStep.Response = (*xtrememodel.MapInterfaceColumn)(stepResponse)

	err := RabbitMQSQL.Where("id = ?", workflowStep.ID).
		Updates(&xtrememodel.RabbitMQAsyncWorkflowStep{
			StatusId: RABBITMQ_ASYNC_WORKFLOW_STATUS_SUCCESS_ID,
			Response: (*xtrememodel.MapInterfaceColumn)(stepResponse),
		}).Error
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Error updating workflow step status to finish: %s", err), true)
	}

	if workflow.TotalStep == workflowStep.StepOrder {
		workflow.StatusId = RABBITMQ_ASYNC_WORKFLOW_STATUS_SUCCESS_ID

		err := RabbitMQSQL.Where("id = ?", workflow.ID).
			Updates(&xtrememodel.RabbitMQAsyncWorkflow{
				StatusId: RABBITMQ_ASYNC_WORKFLOW_STATUS_SUCCESS_ID,
			}).Error
		if err != nil {
			xtremepkg.LogError(fmt.Sprintf("Unable to update async workflow to finish: %s", err), true)
		}
	} else {
		var nextStep xtrememodel.RabbitMQAsyncWorkflowStep
		err := RabbitMQSQL.Where("workflowId = ? AND stepOrder > ?", workflow.ID, workflowStep.StepOrder).
			Order("stepOrder ASC").
			First(&nextStep).Error
		if err != nil {
			xtremepkg.LogError(fmt.Sprintf("Next async workflow step does not exists. Step Order (%d): %s", (workflowStep.StepOrder+1), err), true)
		}

		forwardPayloadMap := make(map[string]AsyncWorkflowForwardPayloadResult)
		forwardPayloadQueues := make([]string, 0)
		for _, forwardPayload := range forwardPayloads {
			if forwardPayload.Payload == nil {
				continue
			}

			if payloadMap, ok := forwardPayload.Payload.(map[string]any); !ok || len(payloadMap) == 0 {
				continue
			}

			forwardPayloadMap[forwardPayload.Queue] = forwardPayload
			forwardPayloadQueues = append(forwardPayloadQueues, forwardPayload.Queue)
		}

		if len(forwardPayloadQueues) > 0 {
			var forwardSteps []xtrememodel.RabbitMQAsyncWorkflowStep
			RabbitMQSQL.Where("workflowId = ? AND queue IN ?", workflow.ID, forwardPayloadQueues).Find(&forwardSteps)
			for _, forwardStep := range forwardSteps {
				originForwardPayload := make(map[string]interface{})
				if forwardStep.ForwardPayload != nil {
					originForwardPayload = *forwardStep.ForwardPayload
				}

				forwardStepPayload := make(map[string]interface{})
				if firstForwardStepPayload, ok := originForwardPayload[workflowStep.Queue].(map[string]interface{}); ok && len(firstForwardStepPayload) > 0 {
					forwardStepPayload = firstForwardStepPayload
				}

				remappingForwardPayload(forwardPayloadMap[forwardStep.Queue].Payload, &forwardStepPayload)

				originForwardPayload[workflowStep.Queue] = forwardStepPayload

				err = RabbitMQSQL.Where("id = ?", forwardStep.ID).
					Updates(&xtrememodel.RabbitMQAsyncWorkflowStep{
						ForwardPayload: (*xtrememodel.MapInterfaceColumn)(&originForwardPayload),
					}).Error
				if err != nil {
					xtremepkg.LogError(fmt.Sprintf("Unable to update forward payload to next step. Step Order (%d): %s", (workflowStep.StepOrder+1), err), true)
				}

				if nextStep.Queue == forwardStep.Queue {
					nextStep.ForwardPayload = (*xtrememodel.MapInterfaceColumn)(&forwardStepPayload)
				}
			}
		}

		payload := make(map[string]interface{})
		resultMap, ok := result.(map[string]interface{})
		if ok && len(resultMap) > 0 {
			payload = resultMap

			err = RabbitMQSQL.Where("id = ?", nextStep.ID).
				Updates(&xtrememodel.RabbitMQAsyncWorkflowStep{
					Payload: (*xtrememodel.MapInterfaceColumn)(&resultMap),
				}).Error
			if err != nil {
				xtremepkg.LogError(fmt.Sprintf("Unable to update payload to next step. Step Order (%d): %s", (workflowStep.StepOrder+1), err), true)
			}
		}

		if nextStep.ForwardPayload != nil && len(*nextStep.ForwardPayload) > 0 {
			for _, forwardPayload := range *nextStep.ForwardPayload {
				remappingForwardPayload(forwardPayload, &payload)
			}
		}

		pushWorkflowMessage(workflow.ID, nextStep.Queue, payload)
	}

	if workflow.StatusId == RABBITMQ_ASYNC_WORKFLOW_STATUS_SUCCESS_ID {
		sendToMonitoringEvent(workflow)

		successMsg := workflow.SuccessMessage
		if successMsg == "" {
			successMsg = fmt.Sprintf("Process in action (%s) has been successfully", workflow.Action)
		}
		pushToNotification(workflow, workflowStep, successMsg, successMsg)
	}

	sendToMonitoringActionEvent(workflow, workflowStep)
}

func sendToMonitoringEvent(workflow xtrememodel.RabbitMQAsyncWorkflow) {
	err := xtremews.Publish(
		xtremews.WS_CHANNEL_MESSAGE_BROKER_ASYNC_WORKFLOW_MONITORING, xtremews.WS_GROUP_ID_ASYNC_WORKFLOW_MONITORING_LIST,
		xtremews.WS_EVENT_MONITORING,
		map[string]interface{}{
			"id":        workflow.ID,
			"service":   workflow.ReferenceService,
			"createdBy": workflow.CreatedBy,
		})
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Unable to send data to monitoring event. %s", err), true)
	}
}

func sendToMonitoringActionEvent(workflow xtrememodel.RabbitMQAsyncWorkflow, workflowStep xtrememodel.RabbitMQAsyncWorkflowStep) {
	result := map[string]interface{}{
		"id":          workflow.ID,
		"action":      workflow.Action,
		"description": workflow.Description,
		"status":      RabbitMQAsyncWorkflowStatus{}.IDAndName(workflow.StatusId),
		"totalStep":   workflow.TotalStep,
		"reprocessed": workflow.Reprocessed,
		"createdBy":   workflow.CreatedByName,
		"createdAt":   workflow.CreatedAt.Format("02/01/2006 15:04:05"),
		"reference": map[string]interface{}{
			"id":      workflow.ReferenceId,
			"type":    workflow.ReferenceType,
			"service": workflow.ReferenceService,
		},
		"step": map[string]interface{}{
			"id":             workflowStep.ID,
			"service":        workflowStep.Service,
			"queue":          workflowStep.Queue,
			"stepOrder":      workflowStep.StepOrder,
			"status":         RabbitMQAsyncWorkflowStatus{}.IDAndName(workflowStep.StatusId),
			"description":    workflowStep.Description,
			"payload":        workflowStep.Payload,
			"forwardPayload": workflowStep.ForwardPayload,
			"errors":         workflowStep.Errors,
			"response":       workflowStep.Response,
			"reprocessed":    workflowStep.Reprocessed,
			"createdAt":      workflowStep.CreatedAt.Format("02/01/2006 15:04:05"),
			"updatedAt":      workflowStep.UpdatedAt.Format("02/01/2006 15:04:05"),
		},
	}

	err := xtremews.Publish(
		xtremews.WS_CHANNEL_MESSAGE_BROKER_ASYNC_WORKFLOW_MONITORING, fmt.Sprintf("%s-%s", workflow.Action, workflow.ReferenceId),
		xtremews.WS_EVENT_MONITORING,
		result)
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Unable to send data to monitoring by action event. Step Order (%d): %s", (workflowStep.StepOrder+1), err), true)
	}
}

func failedWorkflow(message string, err error, trace []byte, workflow *xtrememodel.RabbitMQAsyncWorkflow, workflowStep *xtrememodel.RabbitMQAsyncWorkflowStep) {
	xtremepkg.LogError(message, true)

	workflowStepIsValid := workflowStep != nil && workflowStep.ID > 0
	if workflowStepIsValid && workflowStep.StatusId != RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID {
		exceptionRes := map[string]interface{}{"message": message, "internalMsg": err.Error(), "trace": string(trace)}

		stepErrors := make([]map[string]interface{}, 0)
		if workflowStep.Errors != nil {
			stepErrors = *workflowStep.Errors
		}

		stepErrors = append(stepErrors, exceptionRes)

		workflowStep.StatusId = RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID
		workflowStep.Errors = (*xtrememodel.ArrayMapInterfaceColumn)(&stepErrors)

		err := RabbitMQSQL.Where("id = ?", workflowStep.ID).
			Updates(&xtrememodel.RabbitMQAsyncWorkflowStep{
				StatusId: RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID,
				Errors:   (*xtrememodel.ArrayMapInterfaceColumn)(&stepErrors),
			}).Error
		if err != nil {
			xtremepkg.LogError(fmt.Sprintf("Unable to update async workflow step to error: %s", err), true)
		}
	}

	workflowIsValid := workflow != nil && workflow.ID > 0
	if workflowIsValid && workflow.StatusId != RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID {
		workflow.StatusId = RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID

		err := RabbitMQSQL.Where("id = ?", workflow.ID).
			Updates(&xtrememodel.RabbitMQAsyncWorkflow{
				StatusId: RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID,
			}).Error
		if err != nil {
			xtremepkg.LogError(fmt.Sprintf("Unable to update async workflow to error: %s", err), true)
		}
	}

	if workflowIsValid && workflowStepIsValid {
		sendToMonitoringEvent(*workflow)
		sendToMonitoringActionEvent(*workflow, *workflowStep)

		errTitle := workflow.ErrorMessage
		if errTitle == "" {
			errTitle = message
		}
		pushToNotification(*workflow, *workflowStep, errTitle, err.Error())
	}
}

func pushWorkflowMessage(workflowId uint, queue string, payload interface{}) {
	conn, ok := RabbitMQConnectionDial[RABBITMQ_CONNECTION_GLOBAL]
	if !ok {
		log.Panicf("Please init rabbitmq connection first")
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Panicf("Failed to open a channel: %s", err)
	}
	defer ch.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body, _ := json.Marshal(map[string]interface{}{
		"data":       payload,
		"workflowId": workflowId,
	})

	q, err := ch.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("Failed to declare a queue: %s", err)
	}

	correlationId, _ := exec.Command("uuidgen").Output()
	err = ch.PublishWithContext(ctx,
		"",
		q.Name,
		false,
		false,
		amqp091.Publishing{
			CorrelationId: string(correlationId),
			DeliveryMode:  amqp091.Persistent,
			ContentType:   "application/json",
			Body:          body,
		})
	if err != nil {
		log.Panicf("Failed to send a message: %s", err)
	}
}

func pushToNotification(workflow xtrememodel.RabbitMQAsyncWorkflow, step xtrememodel.RabbitMQAsyncWorkflowStep, title, body string) {
	api, err := privateapi.NewBusinessWorkflowAPI()
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Unable to create new business API: %s", err), true)
		return
	}

	api.NotificationPush(map[string]interface{}{
		"blueprintCode": "async-workflow.admin",
		"service":       workflow.ReferenceService,
		"data": map[string]interface{}{
			"title":       title,
			"body":        body,
			"recipientId": workflow.CreatedBy,
			"deepLink":    "",
		},
	})

	if step.StatusId == RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID && step.Reprocessed >= 10 {
		message := fmt.Sprintf("*ERROR ASA:* %d\n", workflow.ID)
		message += fmt.Sprintf("*Action:* %s\n", workflow.Action)
		message += fmt.Sprintf("*Reference:* %s:%s\n", workflow.ReferenceType, workflow.ReferenceId)
		message += fmt.Sprintf("*Service:* %s\n\n", workflow.ReferenceService)

		message += fmt.Sprintf("*Step:* %d\n", step.StepOrder)
		message += fmt.Sprintf("*Service:* %s\n", step.Service)
		message += fmt.Sprintf("*Executor:* %s\n\n", step.Queue)

		message += fmt.Sprintf("*Title:* %s\n", title)
		message += fmt.Sprintf("*Body:* %s", body)

		api.NotificationPush(map[string]interface{}{
			"blueprintCode": "async-workflow.developer",
			"service":       workflow.ReferenceService,
			"data": map[string]interface{}{
				"message": message,
			},
		})
	}
}

func remappingForwardPayload(forwardPayload any, originStepPayload *map[string]any) {
	if forwardPayload == nil {
		return
	}

	payloadMap, ok := forwardPayload.(map[string]any)
	if !ok {
		return
	}

	for fKey, fPayload := range payloadMap {
		switch val := fPayload.(type) {
		case map[string]any:
			newMap := make(map[string]any)
			(*originStepPayload)[fKey] = newMap
			remappingForwardPayload(val, &newMap)

		case []any:
			newSlice := make([]any, len(val))
			for i, item := range val {
				switch itemVal := item.(type) {
				case map[string]any:
					nestedMap := make(map[string]any)
					remappingForwardPayload(itemVal, &nestedMap)
					newSlice[i] = nestedMap
				default:
					newSlice[i] = itemVal
				}
			}
			(*originStepPayload)[fKey] = newSlice

		default:
			(*originStepPayload)[fKey] = val
		}
	}
}
