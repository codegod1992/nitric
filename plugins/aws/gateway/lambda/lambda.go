package lambda_service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	events "github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/nitric-dev/membrane/handler"
	"github.com/nitric-dev/membrane/sdk"
	"github.com/nitric-dev/membrane/sources"
)

type eventType int

const (
	unknown eventType = iota
	sns
	http
)

type LambdaRuntimeHandler func(handler interface{})

//Event incoming event
type Event struct {
	Requests []sources.Source
}

func (event *Event) getEventType(data []byte) eventType {
	tmp := make(map[string]interface{})
	// Unmarshal so we can get just enough info about the type of event to fully deserialize it
	json.Unmarshal(data, &tmp)

	// If our event is a HTTP request
	if _, ok := tmp["rawPath"]; ok {
		return http
	} else if records, ok := tmp["Records"]; ok {
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

// implement the unmarshal interface in order to handle multiple event types
func (event *Event) UnmarshalJSON(data []byte) error {
	var err error

	event.Requests = make([]sdk.NitricRequest, 0)

	switch event.getEventType(data) {
	case sns:
		snsEvent := &events.SNSEvent{}
		err = json.Unmarshal(data, snsEvent)

		if err == nil {
			// Map over the records and return
			for _, snsRecord := range snsEvent.Records {
				messageString := snsRecord.SNS.Message
				// FIXME: What about non-nitric SNS events???
				messageJson := &sdk.NitricEvent{}

				// Populate the JSON
				err = json.Unmarshal([]byte(messageString), messageJson)

				topicArn := snsRecord.SNS.TopicArn
				topicParts := strings.Split(topicArn, ":")
				source := topicParts[len(topicParts)-1]
				// get the topic name from the full ARN.
				// Get the topic name from the arn

				if err == nil {
					// Decode the message to see if it's a Nitric message
					payloadMap := messageJson.Payload
					payloadBytes, err := json.Marshal(&payloadMap)

					if err == nil {
						event.Requests = append(event.Requests, &sources.Event{
							ID:      messageJson.RequestId,
							Topic:   source,
							Payload: payloadBytes,
						})
					}
				}
			}
		}
		break
	case http:
		httpEvent := &events.APIGatewayV2HTTPRequest{}

		err = json.Unmarshal(data, httpEvent)

		if err == nil {
			event.Requests = append(event.Requests, &sources.HttpRequest{
				// FIXME: Translate to http.Header
				Header: httpEvent.Headers,
				Body:   ioutil.NoopCloser(bytes.NewReader([]byte(httpEvent.Body))),
				Method: httpEvent.RequestContext.HTTP.Method,
				Path:   httpEvent.RawPath,
			})
		}

		break
	default:
		jsonEvent := make(map[string]interface{})

		err = json.Unmarshal(data, &jsonEvent)

		if err != nil {
			return err
		}

		err = fmt.Errorf("Unhandled Event Type: %v", data)
	}

	return err
}

type LambdaGateway struct {
	handler handler.SourceHandler
	runtime LambdaRuntimeHandler
	sdk.UnimplementedGatewayPlugin
}

func (s *LambdaGateway) handle(ctx context.Context, event Event) (interface{}, error) {
	for _, request := range event.Requests {
		// TODO: Build up an array of responses?
		//in some cases we won't need to send a response as well...
		// resp := s.handler(&request)

		switch request.GetSourceType() {
		case sources.SourceType_Request:
			if httpEvent, ok := request.(*sources.HttpRequest); ok {
				response := s.handler.HandleHttpRequest(httpEvent)

				lambdaHTTPHeaders := make(map[string]string)

				for key := range response.Header {
					lambdaHTTPHeaders[key] = response.Header.Get(key)
				}

				return events.APIGatewayProxyResponse{
					StatusCode: response.StatusCode,
					Headers:    lambdaHTTPHeaders,
					Body:       string(resp.Body),
					// TODO: Need to determine best case when to use this...
					IsBase64Encoded: false,
				}, nil
			} else {
				return nil, fmt.Errorf("Error!: Found non HttpRequest in event with source type: %s", sources.SourceType_Request.String())
			}
			break
		case sources.SourceType_Subscription:
			if event, ok := request.(*sources.Event); ok {
				if err := s.handler.HandleEvent(event); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("Error!: Found non Event in event with source type: %s", sources.SourceType_Subscription.String())
			}
			break
		}
	}
	return nil, nil
}

// Start the lambda gateway handler
func (s *LambdaGateway) Start(handler handler.SourceHandler) error {
	s.handler = handler
	// Here we want to begin polling lambda for incoming requests...
	// Assuming that this is blocking
	s.runtime(s.handle)

	return fmt.Errorf("Something went wrong causing the lambda runtime to stop")
}

func New() (sdk.GatewayService, error) {
	return &LambdaGateway{
		runtime: lambda.Start,
	}, nil
}

func NewWithRuntime(runtime LambdaRuntimeHandler) (sdk.GatewayService, error) {
	return &LambdaGateway{
		runtime: runtime,
	}, nil
}
