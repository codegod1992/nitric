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

package grpc

import (
	"github.com/nitric-dev/membrane/pkg/plugins"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Provides GRPC error reporting
func NewGrpcError(operation string, err error) error {
	if iae, ok := err.(*plugins.InvalidArgumentError); ok {
		return status.Errorf(codes.InvalidArgument, "%s: %v", operation, iae)
	} else {
		return status.Errorf(codes.Internal, "%s: %v", operation, err)
	}
}
