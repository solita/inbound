package sinks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/solita/inbound/core"
)

type S3Sink struct {
	context context.Context
	client  *s3.Client
	bucket  string
	prefix  string
}

func (s *S3Sink) StoreMessage(msg core.Message) error {
	key := s.prefix + "/messages/" + msg.Id + ".json"
	value, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message metadata: %w", err)
	}

	_, err = s.client.PutObject(s.context, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   bytes.NewReader(value),
	})
	if err != nil {
		return fmt.Errorf("failed to store message metadata in S3: %w", err)
	}
	return nil
}

func (s *S3Sink) StoreAttachment(id string, data io.Reader) error {
	key := s.prefix + "/attachments/" + id
	_, err := s.client.PutObject(s.context, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   data,
	})
	if err != nil {
		return fmt.Errorf("failed to store attachment in S3: %w", err)
	}

	return nil
}

var _ core.Sink = (*S3Sink)(nil)

func NewS3(bucket, prefix string) (*S3Sink, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	return &S3Sink{
		context: context.TODO(),
		client:  client,
		bucket:  bucket,
		prefix:  prefix,
	}, nil
}
