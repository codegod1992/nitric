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

package gateway

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/nitrictech/nitric/cloud/aws/runtime/core"
	ep "github.com/nitrictech/nitric/core/pkg/plugins/events"
	"github.com/nitrictech/nitric/core/pkg/plugins/gateway"
	"github.com/nitrictech/nitric/core/pkg/triggers"
	"github.com/nitrictech/nitric/core/pkg/worker"
)

type eventType int

const (
	unknown eventType = iota
	sns
	httpEvent
	healthcheck
	xforwardHeader string = "x-forwarded-for"
)

type LambdaRuntimeHandler func(handler interface{})

func getEventType(request map[string]interface{}) eventType {
	// If our event is a HTTP request
	if _, ok := request["rawPath"]; ok {
		return httpEvent
	} else if records, ok := request["Records"]; ok {
		recordsList, _ := records.([]interface{})
		record, _ := recordsList[0].(map[string]interface{})
		// We have some kind of event here...
		// we'll assume its an SNS
		var eventSource string
		if es, ok := record["EventSource"]; ok {
			eventSource = es.(string)
		} else if es, ok := record["eventSource"]; ok {
			eventSource = es.(string)
		}

		switch eventSource {
		case "aws:sns":
			return sns
		}
	}

	return unknown
}

func (s *LambdaGateway) getTopicNameForArn(ctx context.Context, topicArn string) (string, error) {
	topics, err := s.provider.GetResources(ctx, core.AwsResource_Topic)
	if err != nil {
		return "", fmt.Errorf("error retrieving topics: %w", err)
	}

	for name, arn := range topics {
		if arn == topicArn {
			return name, nil
		}
	}

	return "", fmt.Errorf("could not find topic for arn %s", topicArn)
}

func (s *LambdaGateway) isHealthCheck(data map[string]interface{}) bool {
	_, ok := data["x-nitric-healthcheck"]

	return ok
}

func (s *LambdaGateway) triggersFromRequest(ctx context.Context, data map[string]interface{}) ([]triggers.Trigger, error) {
	bytes, _ := json.Marshal(data)
	trigs := make([]triggers.Trigger, 0)

	switch getEventType(data) {
	case sns:
		snsEvent := &events.SNSEvent{}
		if err := json.Unmarshal(bytes, snsEvent); err == nil {
			for _, snsRecord := range snsEvent.Records {
				messageString := snsRecord.SNS.Message
				// FIXME: What about non-nitric SNS events???
				messageJson := &ep.NitricEvent{}
				var payloadBytes []byte
				var id string
				attrs := map[string]string{}

				for k, v := range snsRecord.SNS.MessageAttributes {
					sv, ok := v.(string)
					if ok {
						attrs[k] = sv
					}
				}

				// Populate the JSON
				if err := json.Unmarshal([]byte(messageString), messageJson); err == nil {
					payloadMap := messageJson.Payload
					id = messageJson.ID
					payloadBytes, _ = json.Marshal(&payloadMap)
				} else {
					// just try to capture the raw message
					payloadBytes = []byte(messageString)
					id = snsRecord.SNS.MessageID
				}

				tName, err := s.getTopicNameForArn(ctx, snsRecord.SNS.TopicArn)

				if err == nil {
					trigs = append(trigs, &triggers.Event{
						ID:         id,
						Topic:      tName,
						Payload:    payloadBytes,
						Attributes: attrs,
					})
				} else {
					log.Default().Printf("unable to find nitric topic: %v", err)
				}
			}
		}
	case httpEvent:
		evt := &events.APIGatewayV2HTTPRequest{}

		err := json.Unmarshal(bytes, evt)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal httpEvent: %w", err)
		}

		// Copy the headers and re-write for the proxy
		headerCopy := make(map[string][]string)

		for key, val := range evt.Headers {
			if strings.ToLower(key) == "host" {
				headerCopy[xforwardHeader] = append(headerCopy[xforwardHeader], val)
			} else {
				headerCopy[key] = append(headerCopy[key], val)
			}
		}

		// Copy the cookies over
		headerCopy["Cookie"] = evt.Cookies

		// Parse the raw query string
		qVals, err := url.ParseQuery(evt.RawQueryString)
		if err != nil {
			return nil, fmt.Errorf("error parsing query for httpEvent: %w", err)
		}

		trigs = append(trigs, &triggers.HttpRequest{
			// FIXME: Translate to http.Header
			Header: headerCopy,
			Body:   []byte(evt.Body),
			Method: evt.RequestContext.HTTP.Method,
			Path:   evt.RawPath,
			URL:    evt.RawPath,
			Query:  qVals,
		})

	case healthcheck:

	default:
		return nil, fmt.Errorf("unhandled event type %v", data)
	}

	return trigs, nil
}

type LambdaGateway struct {
	pool     worker.WorkerPool
	provider core.AwsProvider
	runtime  LambdaRuntimeHandler
	gateway.UnimplementedGatewayPlugin
	finished chan int
}

func (s *LambdaGateway) handle(ctx context.Context, data map[string]interface{}) (interface{}, error) {
	if s.isHealthCheck(data) {
		return map[string]interface{}{
			"healthy": true,
		}, nil
	}

	trigs, err := s.triggersFromRequest(ctx, data)
	if err != nil {
		return nil, err
	}

	for _, request := range trigs {
		switch request.GetTriggerType() {
		case triggers.TriggerType_Request:
			if httpEvent, ok := request.(*triggers.HttpRequest); ok {
				wrkr, err := s.pool.GetWorker(&worker.GetWorkerOptions{
					Http: httpEvent,
				})
				if err != nil {
					return nil, fmt.Errorf("unable to get worker to handle http trigger")
				}

				var hc propagation.HeaderCarrier = httpEvent.Header

				response, err := wrkr.HandleHttpRequest(xray.Propagator{}.Extract(ctx, hc), httpEvent)
				if err != nil {
					return events.APIGatewayProxyResponse{
						StatusCode: 500,
						Body:       "Error processing lambda request",
						// TODO: Need to determine best case when to use this...
						IsBase64Encoded: true,
					}, nil
				}

				lambdaHTTPHeaders := make(map[string]string)

				if response.Header != nil {
					response.Header.VisitAll(func(key []byte, val []byte) {
						lambdaHTTPHeaders[string(key)] = string(val)
					})
				}

				responseString := base64.StdEncoding.EncodeToString(response.Body)

				// We want to sniff the content type of the body that we have here as lambda cannot gzip it...
				return events.APIGatewayProxyResponse{
					StatusCode: response.StatusCode,
					Headers:    lambdaHTTPHeaders,
					Body:       responseString,
					// TODO: Need to determine best case when to use this...
					IsBase64Encoded: true,
				}, nil
			} else {
				return nil, fmt.Errorf("found non HttpRequest in event with trigger type: %s", triggers.TriggerType_Request.String())
			}
		case triggers.TriggerType_Subscription:
			if event, ok := request.(*triggers.Event); ok {
				wrkr, err := s.pool.GetWorker(&worker.GetWorkerOptions{
					Event: event,
				})
				if err != nil {
					return nil, fmt.Errorf("unable to get worker to event trigger")
				}

				var mc propagation.MapCarrier = event.Attributes

				if err := wrkr.HandleEvent(xray.Propagator{}.Extract(ctx, mc), event); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("found non Event in event with trigger type: %s", triggers.TriggerType_Subscription.String())
			}
		}
	}

	return nil, nil
}

// Start the lambda gateway handler
func (s *LambdaGateway) Start(pool worker.WorkerPool) error {
	// s.finished = make(chan int)
	s.pool = pool
	// Here we want to begin polling lambda for incoming requests...
	s.runtime(func(ctx context.Context, data map[string]interface{}) (interface{}, error) {
		a, err := s.handle(ctx, data)

		tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
		if ok {
			_ = tp.ForceFlush(ctx)
		}

		return a, err
	})
	// Unblock the 'Stop' function if it's waiting.
	go func() { s.finished <- 1 }()
	return nil
}

func (s *LambdaGateway) Stop() error {
	// XXX: This is a NO_OP Process, as this is a pull based system
	// We don't need to stop listening to anything
	log.Default().Println("gateway 'Stop' called, waiting for lambda runtime to finish")
	// Lambda can't be stopped, need to wait for it to finish
	<-s.finished
	return nil
}

func New(provider core.AwsProvider) (gateway.GatewayService, error) {
	return &LambdaGateway{
		provider: provider,
		runtime:  lambda.Start,
		finished: make(chan int),
	}, nil
}

func NewWithRuntime(provider core.AwsProvider, runtime LambdaRuntimeHandler) (gateway.GatewayService, error) {
	return &LambdaGateway{
		provider: provider,
		runtime:  runtime,
		finished: make(chan int),
	}, nil
}
