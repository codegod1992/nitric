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

package sdk

import "fmt"

type Key struct {
	Collection string
	Id         string
}

func (k *Key) String() string {
	return fmt.Sprintf("Key{Collection: %v Id: %v}\n", k.Collection, k.Id)
}

type QueryExpression struct {
	Operand  string
	Operator string
	Value    string
}

type QueryResult struct {
	Data        []map[string]interface{}
	PagingToken map[string]string
}

// The base Document Plugin interface
// Use this over proto definitions to remove dependency on protobuf in the plugin internally
// and open options to adding additional non-grpc interfaces
type DocumentService interface {
	Get(*Key, *Key) (map[string]interface{}, error)
	Set(*Key, *Key, map[string]interface{}) error
	Delete(*Key, *Key) error
	Query(*Key, string, []QueryExpression, int, map[string]string) (*QueryResult, error)
}

type UnimplementedDocumentPlugin struct {
	DocumentService
}

func (p *UnimplementedDocumentPlugin) Get(key *Key, subKey *Key) (map[string]interface{}, error) {
	return nil, fmt.Errorf("UNIMPLEMENTED")
}

func (p *UnimplementedDocumentPlugin) Set(key *Key, subKey *Key, value map[string]interface{}) error {
	return fmt.Errorf("UNIMPLEMENTED")
}

func (p *UnimplementedDocumentPlugin) Delete(key *Key, subKey *Key) error {
	return fmt.Errorf("UNIMPLEMENTED")
}

func (p *UnimplementedDocumentPlugin) Query(key *Key, subcollection string, expressions []QueryExpression, limit int, pagingToken map[string]string) (*QueryResult, error) {
	return nil, fmt.Errorf("UNIMPLEMENTED")
}
