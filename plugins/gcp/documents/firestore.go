package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/pubsub"
	"github.com/nitric-dev/membrane/plugins/sdk"
	"golang.org/x/oauth2/google"
)

type FirestorePlugin struct {
	client *firestore.Client
	sdk.UnimplementedDocumentsPlugin
}

func (s *FirestorePlugin) CreateDocument(collection string, key string, document map[string]interface{}) error {
	// Create a new document is firestore
	if key == "" {
		return fmt.Errorf("Key autogeneration unimplemented, please provide non-blank key")
	}

	_, error := s.client.Collection(collection).Doc(key).Create(context.TODO(), document)

	if error != nil {
		return fmt.Errorf("Error creating new document: %v", error)
	}

	return nil
}

func (s *FirestorePlugin) GetDocument(collection string, key string) (map[string]interface{}, error) {
	document, error := s.client.Collection(collection).Doc(key).Get(context.TODO())

	if error != nil {
		return nil, fmt.Errorf("Error retrieving document: %v", error)
	}

	return document.Data(), nil
}

func (s *FirestorePlugin) UpdateDocument(collection string, key string, document map[string]interface{}) error {
	_, error := s.client.Collection(collection).Doc(key).Set(context.TODO(), request.GetDocument().AsMap())

	if error != nil {
		return fmt.Errorf("Error creating retrieving document: %v", error)
	}

	return nil
}

func (s *FirestorePlugin) DeleteDocument(collection string, key string) error {
	_, error := s.client.Collection(collection).Doc(key).Delete(context.TODO())

	if error != nil {
		return fmt.Errorf("Error deleting document: %v", error)
	}

	return nil
}

func New() (sdk.DocumentsPlugin, error) {
	ctx := context.Background()

	credentials, credentialsError := google.FindDefaultCredentials(ctx, pubsub.ScopeCloudPlatform)
	if credentialsError != nil {
		return nil, fmt.Errorf("GCP credentials error: %v", credentialsError)
	}

	client, clientError := firestore.NewClient(ctx, credentials.ProjectID)
	if clientError != nil {
		return nil, fmt.Errorf("firestore client error: %v", clientError)
	}

	return &FirestorePlugin{
		client: client,
	}, nil
}
