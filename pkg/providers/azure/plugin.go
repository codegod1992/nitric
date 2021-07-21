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
	"github.com/nitric-dev/membrane/pkg/plugins/document"
	"github.com/nitric-dev/membrane/pkg/plugins/eventing"
	"github.com/nitric-dev/membrane/pkg/plugins/gateway"
	http_service "github.com/nitric-dev/membrane/pkg/plugins/gateway/appservice"
	"github.com/nitric-dev/membrane/pkg/plugins/queue"
	"github.com/nitric-dev/membrane/pkg/plugins/storage"
	"github.com/nitric-dev/membrane/pkg/providers"
	"github.com/nitric-dev/membrane/pkg/sdk"
)

type AzureServiceFactory struct {
}

func New() providers.ServiceFactory {
	return &AzureServiceFactory{}
}

// NewDocumentService - Returns Azure _ based document plugin
func (p *AzureServiceFactory) NewDocumentService() (document.DocumentService, error) {
	return &sdk.UnimplementedDocumentPlugin{}, nil
}

// NewEventService - Returns Azure _ based eventing plugin
func (p *AzureServiceFactory) NewEventService() (eventing.EventService, error) {
	return &sdk.UnimplementedEventingPlugin{}, nil
}

// NewGatewayService - Returns Azure _ Gateway plugin
func (p *AzureServiceFactory) NewGatewayService() (gateway.GatewayService, error) {
	return http_service.New()
}

// NewQueueService - Returns Azure _ based queue plugin
func (p *AzureServiceFactory) NewQueueService() (queue.QueueService, error) {
	return &sdk.UnimplementedQueuePlugin{}, nil
}

// NewStorageService - Returns Azure _ based storage plugin
func (p *AzureServiceFactory) NewStorageService() (storage.StorageService, error) {
	return &sdk.UnimplementedStoragePlugin{}, nil
}
