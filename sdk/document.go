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

type Collection struct {
	Name   string
	Parent *Key
}

type Key struct {
	Collection Collection
	Id         string
}

func (k *Key) String() string {
	return fmt.Sprintf("Key{Collection: %v Id: %v}\n", k.Collection, k.Id)
}

type Document struct {
	Content map[string]interface{}
}

func (d *Document) String() string {
	return fmt.Sprintf("Document{Content: %v}\n", d.Content)
}

type QueryExpression struct {
	Operand  string
	Operator string
	// Value    interface{}
	// TODO: convert to interface
	Value string
}

type QueryResult struct {
	Documents   []Document
	PagingToken map[string]string
}

// The base Document Plugin interface
// Use this over proto definitions to remove dependency on protobuf in the plugin internally
// and open options to adding additional non-grpc interfaces
type DocumentService interface {
	Get(*Key) (*Document, error)
	Set(*Key, map[string]interface{}) error
	Delete(*Key) error
	Query(*Key, []QueryExpression, int, map[string]string) (*QueryResult, error)
}

type UnimplementedDocumentPlugin struct {
	DocumentService
}

func (p *UnimplementedDocumentPlugin) Get(key *Key) (*Document, error) {
	return nil, fmt.Errorf("UNIMPLEMENTED")
}

func (p *UnimplementedDocumentPlugin) Set(key *Key, content map[string]interface{}) error {
	return fmt.Errorf("UNIMPLEMENTED")
}

func (p *UnimplementedDocumentPlugin) Delete(key *Key) error {
	return fmt.Errorf("UNIMPLEMENTED")
}

func (p *UnimplementedDocumentPlugin) Query(key *Key, expressions []QueryExpression, limit int, pagingToken map[string]string) (*QueryResult, error) {
	return nil, fmt.Errorf("UNIMPLEMENTED")
}
