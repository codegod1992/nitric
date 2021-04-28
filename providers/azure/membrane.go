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
	http_service "github.com/nitric-dev/membrane/plugins/gateway/appservice"
	"github.com/nitric-dev/membrane/sdk"
	"log"

	"github.com/nitric-dev/membrane/membrane"
)

func main() {

	authPlugin := &sdk.UnimplementedAuthPlugin{}
	kvPlugin := &sdk.UnimplementedKeyValuePlugin{}
	eventingPlugin := &sdk.UnimplementedEventingPlugin{}
	gatewayPlugin, _ := http_service.New()
	storagePlugin := &sdk.UnimplementedStoragePlugin{}
	queuePlugin := &sdk.UnimplementedQueuePlugin{}

	m, err := membrane.New(&membrane.MembraneOptions{
		AuthPlugin:              authPlugin,
		KvPlugin:                kvPlugin,
		EventingPlugin:          eventingPlugin,
		GatewayPlugin:           gatewayPlugin,
		StoragePlugin:           storagePlugin,
		QueuePlugin:             queuePlugin,
	})

	if err != nil {
		log.Fatalf("There was an error initialising the m server: %v", err)
	}

	// Start the Membrane server
	m.Start()
}
