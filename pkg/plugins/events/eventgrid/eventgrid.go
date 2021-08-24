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

package eventgrid_service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/eventgrid/2018-01-01/eventgrid"
	"github.com/Azure/azure-sdk-for-go/services/eventgrid/2018-01-01/eventgrid/eventgridapi"
	eventgridmgmt "github.com/Azure/azure-sdk-for-go/services/eventgrid/mgmt/2020-06-01/eventgrid"
	eventgridmgmtapi "github.com/Azure/azure-sdk-for-go/services/eventgrid/mgmt/2020-06-01/eventgrid/eventgridapi"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/nitric-dev/membrane/pkg/plugins/errors"
	"github.com/nitric-dev/membrane/pkg/plugins/errors/codes"
	"github.com/nitric-dev/membrane/pkg/plugins/events"
	"github.com/nitric-dev/membrane/pkg/utils"
)

type EventGridEventService struct {
	events.UnimplementedeventsPlugin
	client        eventgridapi.BaseClientAPI
	topicClient   eventgridmgmtapi.TopicsClientAPI
	topicLocation string
	accessToken   AzureAccessToken
}

type AzureAccessToken struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    string `json:"expires_in"`
	ExtExpiresIn string `json:"ext_expires_in"`
	ExpiresOn    string `json:"expires_on"`
	NotBefore    string `json:"not_before"`
	Resource     string `json:"resource"`
	AccessToken  string `json:"access_token"`
}

func GetToken(tenantId string, clientId string, clientSecret string) (AzureAccessToken, error) {
	requestAccessTokenUri := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/token", tenantId)
	requestBody := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientId},
		"client_secret": {clientSecret},
		"resource":      {"https://management.azure.com/"},
	}
	resp, err := http.PostForm(requestAccessTokenUri, requestBody)
	if err != nil {
		return AzureAccessToken{}, err
	}

	var result AzureAccessToken

	json.NewDecoder(resp.Body).Decode(&result)

	return result, nil
}

func (s *EventGridEventService) NitricEventToEvent(topic string, event *events.NitricEvent) ([]eventgrid.Event, error) {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return nil, err
	}
	subject := fmt.Sprintf("Subject/%s", topic)
	eventType := fmt.Sprintf("Type/%s", topic)
	azureEvent := []eventgrid.Event{
		{
			ID:        &event.ID,
			Data:      &payload,
			Topic:     &topic,
			EventType: &eventType,
			Subject:   &subject,
			EventTime: &date.Time{time.Now()},
		},
	}

	return azureEvent, nil
}

func (s *EventGridEventService) ListTopics() ([]string, error) {
	newErr := errors.ErrorsWithScope("EventGridEventService.ListTopics")
	ctx := context.Background()
	ctx.Value(map[string]string{
		"Authorization": fmt.Sprintf("%s %s", s.accessToken.TokenType, s.accessToken.AccessToken),
	})

	pageLength := int32(20)

	results, err := s.topicClient.ListBySubscription(ctx, "", &pageLength)

	if err != nil {
		return nil, newErr(
			codes.Internal,
			"azure list topics error",
			err,
		)
	}

	var topics []string

	for results.NotDone() {
		topicsList := results.Values()
		for _, topic := range topicsList {
			topics = append(topics, *topic.Name)
		}
		results.NextWithContext(ctx)
	}

	return topics, nil
}

func (s *EventGridEventService) Publish(topic string, event *events.NitricEvent) error {
	newErr := errors.ErrorsWithScope("EventGridEventService.Publish")
	ctx := context.Background()
	ctx.Value(map[string]string{
		"Authorization": fmt.Sprintf("%s %s", s.accessToken.TokenType, s.accessToken.AccessToken),
	})

	if len(topic) == 0 {
		return newErr(
			codes.InvalidArgument,
			"provide non-blank topic",
			fmt.Errorf("provided invalid topic"),
		)
	}
	if event == nil {
		return newErr(
			codes.InvalidArgument,
			"provide non-nil event",
			fmt.Errorf("provided invalid event"),
		)
	}

	//Convert topic -> topic1.westus2-1.eventgrid.azure.net
	topicHostName := fmt.Sprintf("%s.%s.eventgrid.azure.net", topic, strings.ToLower(s.topicLocation))

	events, err := s.NitricEventToEvent(topic, event)
	if err != nil {
		return err
	}

	result, err := s.client.PublishEvents(ctx, topicHostName, events)

	if err != nil {
		return newErr(
			codes.Internal,
			"azure publish event error",
			err,
		)
	}

	if result.StatusCode != 200 {
		return newErr(
			codes.Internal,
			"azure publish event returned non-200 status code",
			fmt.Errorf(string(rune(result.StatusCode))),
		)
	}
	return nil
}

func New() (events.EventService, error) {
	newErr := errors.ErrorsWithScope("EventGridEventService.New")
	topicLocation := utils.GetEnv("AZURE_TOPIC_LOCATION", "")
	subscriptionID := utils.GetEnv("AZURE_SUBSCRIPTION_ID", "")
	tenantId := utils.GetEnv("AZURE_TENANT_ID", "")
	clientId := utils.GetEnv("AZURE_CLIENT_ID", "")
	clientSecret := utils.GetEnv("AZURE_CLIENT_SECRET", "")

	if len(tenantId) == 0 {
		return nil, newErr(
			codes.InvalidArgument,
			"AZURE_TENANT_ID not configured",
			fmt.Errorf(""),
		)
	}
	if len(clientId) == 0 {
		return nil, newErr(
			codes.InvalidArgument,
			"AZURE_CLIENT_ID not configured",
			fmt.Errorf(""),
		)
	}
	if len(clientSecret) == 0 {
		return nil, newErr(
			codes.InvalidArgument,
			"AZURE_CLIENT_SECRET not configured",
			fmt.Errorf(""),
		)
	}
	if len(topicLocation) == 0 {
		return nil, newErr(
			codes.InvalidArgument,
			"AZURE_TOPIC_LOCATION not configured",
			fmt.Errorf(""),
		)
	}
	if len(subscriptionID) == 0 {
		return nil, newErr(
			codes.InvalidArgument,
			"AZURE_SUBSCRIPTION_ID not configured",
			fmt.Errorf(""),
		)
	}
	client := eventgrid.New()
	topicClient := eventgridmgmt.NewTopicsClient(subscriptionID)
	accessToken, err := GetToken(tenantId, clientId, clientSecret)
	if err != nil {
		return nil, newErr(
			codes.Unauthenticated,
			"Error authenticating event grid",
			err,
		)
	}
	return &EventGridEventService{
		client:        client,
		topicClient:   topicClient,
		topicLocation: topicLocation,
		accessToken:   accessToken,
	}, nil
}

func NewWithClient(client eventgridapi.BaseClientAPI, topicClient eventgridmgmtapi.TopicsClientAPI) (events.EventService, error) {
	return &EventGridEventService{
		client:        client,
		topicClient:   topicClient,
		topicLocation: "local1-test",
	}, nil
}
