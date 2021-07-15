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

package main

import (
	"github.com/nitric-dev/membrane/pkg/plugins/document/firestore"
	pubsub_service "github.com/nitric-dev/membrane/pkg/plugins/eventing/pubsub"
	"github.com/nitric-dev/membrane/pkg/plugins/gateway/cloudrun"
	pubsub_queue_service "github.com/nitric-dev/membrane/pkg/plugins/queue/pubsub"
	storage_service "github.com/nitric-dev/membrane/pkg/plugins/storage/storage"
	"github.com/nitric-dev/membrane/pkg/sdk"
)

type GCPServiceFactory struct {
}

func New() sdk.ServiceFactory {
	return &GCPServiceFactory{}
}

// NewDocumentService - Returns Google Cloud Firestore based document service
func (p *GCPServiceFactory) NewDocumentService() (sdk.DocumentService, error) {
	return firestore_service.New()
}

// NewEventService - Returns Google Cloud Pubsub based eventing service
func (p *GCPServiceFactory) NewEventService() (sdk.EventService, error) {
	return pubsub_service.New()
}

// NewGatewayService - Google Cloud Http Gateway service
func (p *GCPServiceFactory) NewGatewayService() (sdk.GatewayService, error) {
	return cloudrun_plugin.New()
}

// NewQueueService - Returns Google Cloud Pubsub based queue service
func (p *GCPServiceFactory) NewQueueService() (sdk.QueueService, error) {
	return pubsub_queue_service.New()
}

// NewStorageService - Returns Google Cloud Storage based storage service
func (p *GCPServiceFactory) NewStorageService() (sdk.StorageService, error) {
	return storage_service.New()
}
