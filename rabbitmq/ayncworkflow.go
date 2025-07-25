package xtremerabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	xtrememodel "github.com/globalxtreme/go-core/v2/model"
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	"github.com/rabbitmq/amqp091-go"
	"log"
	"os/exec"
	"strconv"
	"time"
)

type GXAsyncWorkflow struct {
	Strict         bool
	Action         string
	ReferenceId    string
	ReferenceType  string
	SuccessMessage string
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
			Where(`statusId != ?`, RABBITMQ_ASYNC_WORKFLOW_STATUS_FINISH_ID).
			Count(&countWorkflow)
		if err != nil || countWorkflow > 0 {
			log.Panicf("You have an asynchronous workflow not yet finished. Please check your workflow status and reprocess")
		}
	}

	workflow := xtrememodel.RabbitMQAsyncWorkflow{
		Action:           flow.Action,
		ReferenceId:      flow.ReferenceId,
		ReferenceType:    flow.ReferenceType,
		ReferenceService: xtremepkg.GetServiceName(),
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
			workflowStep.Payload = (*xtrememodel.MapInterfaceColumn)(&payload)
		}

		workflowSteps = append(workflowSteps, workflowStep)
	}

	err = RabbitMQSQL.Create(&workflowSteps).Error
	if err != nil {
		log.Panicf("Unable to create workflow steps: %s", err.Error())
	}

	pushWorkflowMessage(workflow.ID, flow.firstStep.Queue, flow.firstStep.Payload)
}

type AsyncWorkflowConsumerInterface interface {
	setReferenceId(referenceId string)
	setReferenceType(referenceType string)

	GetReferenceId() string
	GetReferenceType() string
	Consume(payload interface{}) (interface{}, error)
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
	referenceId      string
	referenceType    string
	referenceService string
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
		failedWorkflow(fmt.Sprintf("Get async workflow data: %s", err), nil, nil)
		return
	}

	var workflowStep xtrememodel.RabbitMQAsyncWorkflowStep
	err = RabbitMQSQL.Where("workflowId = ? AND queue = ?", mqBody.WorkflowId, opt.Queue).First(&workflowStep).Error
	if err != nil {
		failedWorkflow(fmt.Sprintf("Get async workflow step data %s: %s", opt.Queue, err), &workflow, nil)
		return
	}

	opt.Consumer.setReferenceId(workflow.ReferenceId)
	opt.Consumer.setReferenceType(workflow.ReferenceType)

	processingWorkflow(&workflow, &workflowStep)

	var result interface{}
	if workflowStep.StatusId != RABBITMQ_ASYNC_WORKFLOW_STATUS_FINISH_ID {
		result, err = opt.Consumer.Consume(mqBody.Data)
		if err != nil {
			failedWorkflow(fmt.Sprintf("Consume async workflow is failed: %s", err), &workflow, &workflowStep)
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
}

func finishWorkflow(workflow xtrememodel.RabbitMQAsyncWorkflow, workflowStep xtrememodel.RabbitMQAsyncWorkflowStep, result interface{}, forwardPayloads []AsyncWorkflowForwardPayloadResult) {
	var stepResponse *map[string]interface{}
	if stepResponseMap, ok := result.(map[string]interface{}); ok && len(stepResponseMap) > 0 {
		stepResponse = &stepResponseMap
	}

	err := RabbitMQSQL.Where("id = ?", workflowStep.ID).
		Updates(&xtrememodel.RabbitMQAsyncWorkflowStep{
			StatusId: RABBITMQ_ASYNC_WORKFLOW_STATUS_FINISH_ID,
			Response: (*xtrememodel.MapInterfaceColumn)(stepResponse),
		}).Error
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Error updating workflow step status to finish: %s", err), true)
	}

	if workflow.TotalStep == workflowStep.StepOrder {
		err := RabbitMQSQL.Where("id = ?", workflow.ID).
			Updates(&xtrememodel.RabbitMQAsyncWorkflow{
				StatusId: RABBITMQ_ASYNC_WORKFLOW_STATUS_FINISH_ID,
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

			if nextStep.ForwardPayload != nil && len(*nextStep.ForwardPayload) > 0 {
				for _, forwardPayload := range *nextStep.ForwardPayload {
					remappingForwardPayload(forwardPayload, &payload)
				}
			}
		}

		pushWorkflowMessage(workflow.ID, nextStep.Queue, payload)
	}
}

func failedWorkflow(errorMsg string, workflow *xtrememodel.RabbitMQAsyncWorkflow, workflowStep *xtrememodel.RabbitMQAsyncWorkflowStep) {
	xtremepkg.LogError(errorMsg, true)

	if workflowStep != nil && workflowStep.ID > 0 && workflowStep.StatusId != RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID {
		exceptionRes := map[string]interface{}{"message": errorMsg, "trace": ""}

		stepErrors := make([]map[string]interface{}, 0)
		if workflowStep.Errors != nil {
			stepErrors = *workflowStep.Errors
		}

		stepErrors = append(stepErrors, exceptionRes)

		err := RabbitMQSQL.Where("id = ?", workflowStep.ID).
			Updates(&xtrememodel.RabbitMQAsyncWorkflowStep{
				StatusId: RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID,
				Errors:   (*xtrememodel.ArrayMapInterfaceColumn)(&stepErrors),
			}).Error
		if err != nil {
			xtremepkg.LogError(fmt.Sprintf("Unable to update async workflow step to error: %s", err), true)
		}
	}

	if workflow != nil && workflow.ID > 0 && workflow.StatusId != RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID {
		err := RabbitMQSQL.Where("id = ?", workflow.ID).
			Updates(&xtrememodel.RabbitMQAsyncWorkflow{
				StatusId: RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID,
			}).Error
		if err != nil {
			xtremepkg.LogError(fmt.Sprintf("Unable to update async workflow to error: %s", err), true)
		}
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
