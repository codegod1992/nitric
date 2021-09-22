// Copyright 2021 Nitric Pty Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azqueue_service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	azqueueserviceiface "github.com/nitric-dev/membrane/pkg/plugins/queue/azqueue/iface"
	azureutils "github.com/nitric-dev/membrane/pkg/providers/azure/utils"

	"github.com/nitric-dev/membrane/pkg/utils"

	"github.com/Azure/azure-storage-queue-go/azqueue"

	"github.com/nitric-dev/membrane/pkg/plugins/errors"
	"github.com/nitric-dev/membrane/pkg/plugins/errors/codes"
	"github.com/nitric-dev/membrane/pkg/plugins/queue"
)

// Set to 30 seconds,
const defaultVisibilityTimeout = 30 * time.Second

type AzqueueQueueService struct {
	client azqueueserviceiface.AzqueueServiceUrlIface
}

// Returns an adapted azqueue MessagesUrl, which is a client for interacting with messages in a specific queue
func (s *AzqueueQueueService) getMessagesUrl(queue string) azqueueserviceiface.AzqueueMessageUrlIface {
	qUrl := s.client.NewQueueURL(queue)
	// Get a new messages URL (used to interact with messages in the queue)
	return qUrl.NewMessageURL()
}

// Returns an adapted azqueue MessageIdUrl, which is a client for interacting with a specific message (task) in a specific queue
func (s *AzqueueQueueService) getMessageIdUrl(queue string, messageId azqueue.MessageID) azqueueserviceiface.AzqueueMessageIdUrlIface {
	mUrl := s.getMessagesUrl(queue)

	return mUrl.NewMessageIDURL(messageId)
}

func (s *AzqueueQueueService) Send(queue string, task queue.NitricTask) error {
	newErr := errors.ErrorsWithScope(
		"AzqueueQueueService.Send",
		map[string]interface{}{
			"queue": queue,
			"task":  task,
		},
	)

	messages := s.getMessagesUrl(queue)

	// Send the tasks to the queue
	if taskBytes, err := json.Marshal(task); err == nil {
		ctx := context.TODO()
		if _, err := messages.Enqueue(ctx, string(taskBytes), 0, 0); err != nil {
			return newErr(
				codes.Internal,
				"error sending task to queue",
				err,
			)
		}
	} else {
		return newErr(
			codes.Internal,
			"error marshalling the task",
			err,
		)
	}

	return nil
}

func (s *AzqueueQueueService) SendBatch(queueName string, tasks []queue.NitricTask) (*queue.SendBatchResponse, error) {
	failedTasks := make([]*queue.FailedTask, 0)

	for _, task := range tasks {
		// Azure Storage Queues don't support batches, so each task must be sent individually.
		err := s.Send(queueName, task)
		if err != nil {
			failedTasks = append(failedTasks, &queue.FailedTask{
				Task:    &task,
				Message: err.Error(),
			})
		}
	}

	return &queue.SendBatchResponse{
		FailedTasks: failedTasks,
	}, nil
}

// AzureQueueItemLease - Represents a lease of an Azure Storages Queues item
// Azure requires a combination of their unique reference for a queue item (id) and a pop receipt (lease id)
// to perform operations on the item, such as delete it from the queue.
type AzureQueueItemLease struct {
	// The ID of the queue item
	// note: this is an id generated by Azure, it's not the user provided unique id.
	ID string
	// lease id, a new popReceipt is generated each time an item is dequeued.
	PopReceipt string
}

// String - convert the item lease struct to a string, to be returned as a NitricTask LeaseID
func (l *AzureQueueItemLease) String() (string, error) {
	leaseID, err := json.Marshal(l)
	return string(leaseID), err
}

// leaseFromString - Unmarshal a NitricTask Lease ID (JSON) to an AzureQueueItemLease
func leaseFromString(leaseID string) (*AzureQueueItemLease, error) {
	var lease AzureQueueItemLease
	err := json.Unmarshal([]byte(leaseID), &lease)
	if err != nil {
		return nil, err
	}
	return &lease, nil
}

// Receive - Receives a collection of tasks off a given queue.
func (s *AzqueueQueueService) Receive(options queue.ReceiveOptions) ([]queue.NitricTask, error) {
	newErr := errors.ErrorsWithScope(
		"AzqueueQueueService.Receive",
		map[string]interface{}{
			"options": options,
		},
	)

	if err := options.Validate(); err != nil {
		return nil, newErr(
			codes.InvalidArgument,
			"invalid receive options provided",
			err,
		)
	}

	messages := s.getMessagesUrl(options.QueueName)

	ctx := context.TODO()
	dequeueResp, err := messages.Dequeue(ctx, int32(*options.Depth), defaultVisibilityTimeout)
	if err != nil {
		return nil, newErr(
			codes.Internal,
			"failed to received messages from the queue",
			err,
		)
	}

	if dequeueResp.NumMessages() == 0 {
		return []queue.NitricTask{}, nil
	}

	// Convert the Azure Storage Queues messages into Nitric tasks
	var tasks []queue.NitricTask
	for i := int32(0); i < dequeueResp.NumMessages(); i++ {
		m := dequeueResp.Message(i)
		var nitricTask queue.NitricTask
		err := json.Unmarshal([]byte(m.Text), &nitricTask)
		if err != nil {
			// TODO: append error to error list and Nack the message.
			continue
		}

		lease := AzureQueueItemLease{
			ID:         m.ID.String(),
			PopReceipt: m.PopReceipt.String(),
		}
		leaseID, err := lease.String()
		// This should never happen, it's a fatal error
		if err != nil {
			return nil, newErr(
				codes.Internal,
				"failed to construct queue item lease id",
				err,
			)
		}

		tasks = append(tasks, queue.NitricTask{
			ID:          nitricTask.ID,
			Payload:     nitricTask.Payload,
			PayloadType: nitricTask.PayloadType,
			LeaseID:     leaseID,
		})
	}

	return tasks, nil
}

// Complete - Completes a previously popped queue item
func (s *AzqueueQueueService) Complete(queue string, leaseId string) error {
	newErr := errors.ErrorsWithScope(
		"AzqueueQueueService.Complete",
		map[string]interface{}{
			"queue":   queue,
			"leaseId": leaseId,
		},
	)

	lease, err := leaseFromString(leaseId)
	if err != nil {
		return newErr(
			codes.InvalidArgument,
			"failed to unmarshal lease id value",
			err,
		)
	}

	// Client for the specific message referenced by the lease
	task := s.getMessageIdUrl(queue, azqueue.MessageID(lease.ID))
	ctx := context.TODO()
	_, err = task.Delete(ctx, azqueue.PopReceipt(lease.PopReceipt))
	if err != nil {
		return newErr(
			codes.Internal,
			"failed to complete task",
			err,
		)
	}

	return nil
}

const expiryBuffer = 2 * time.Minute

func tokenRefresherFromSpt(spt *adal.ServicePrincipalToken) azqueue.TokenRefresher {
	return func(credential azqueue.TokenCredential) time.Duration {
		if err := spt.Refresh(); err != nil {
			fmt.Println("Error refreshing token: ", err)
		} else {
			tkn := spt.Token()
			credential.SetToken(tkn.AccessToken)

			return tkn.Expires().Sub(time.Now().Add(expiryBuffer))
		}

		// Mark the token as already expired
		return time.Duration(0)
	}
}

// New - Constructs a new Azure Storage Queues client with defaults
func New() (queue.QueueService, error) {
	queueUrl := utils.GetEnv(azureutils.AZURE_STORAGE_QUEUE_ENDPOINT, "")
	if queueUrl == "" {
		return nil, fmt.Errorf("failed to determine Azure Storage Queue endpoint, environment variable %s not set", azureutils.AZURE_STORAGE_QUEUE_ENDPOINT)
	}

	spt, err := azureutils.GetServicePrincipalToken(azure.PublicCloud.ResourceIdentifiers.Storage)
	if err != nil {
		return nil, err
	}

	cTkn := azqueue.NewTokenCredential(spt.Token().AccessToken, tokenRefresherFromSpt(spt))

	var accountURL *url.URL
	if accountURL, err = url.Parse(queueUrl); err != nil {
		return nil, err
	}

	pipeline := azqueue.NewPipeline(cTkn, azqueue.PipelineOptions{})
	client := azqueue.NewServiceURL(*accountURL, pipeline)

	return &AzqueueQueueService{
		client: azqueueserviceiface.AdaptServiceUrl(client),
	}, nil
}

func NewWithClient(client azqueueserviceiface.AzqueueServiceUrlIface) queue.QueueService {
	return &AzqueueQueueService{
		client: client,
	}
}
