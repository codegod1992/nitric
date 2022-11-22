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

package common

import "fmt"

type ResourceType = string

const (
	ResourceType_Api = "api"
)

type DetailsResponse[T any] struct {
	Id       string
	Provider string
	Service  string
	Detail   T
}

type ApiDetails struct {
	URL string
}

// ResourceService - Base resource service interface for providers
type ResourceService interface {
	// Details - The details endpoint
	Details(ResourceType, name string) (*DetailsResponse[any], error)
}

type UnimplementResourceService struct{}

var _ ResourceService = &UnimplementResourceService{}

func (*UnimplementResourceService) Details(typ ResourceType, name string) (*DetailsResponse[any], error) {
	return nil, fmt.Errorf("Unimplemented")
}
