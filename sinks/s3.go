package sinks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/solita/inbound/core"
)

type S3Sink struct {
	context       context.Context
	client        *s3.Client
	bucket        string
	prefix        string
	forceSeekable bool
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
	if s.forceSeekable {
		// If S3 endpoint uses HTTP, we need our data to be a seekable reader
		// for checksum calculations; otherwise, this should be avoided to save memory
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, data); err != nil {
			return fmt.Errorf("failed to read attachment data: %w", err)
		}
		data = bytes.NewReader(buf.Bytes())
	}
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

func NewS3(bucket, prefix, baseEndpoint string) (*S3Sink, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	forceSeekable := false
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if baseEndpoint != "" {
			o.BaseEndpoint = aws.String(baseEndpoint)
			o.UsePathStyle = true
			if strings.HasPrefix(baseEndpoint, "http://") {
				forceSeekable = true
			}
		}
	})
	return &S3Sink{
		context:       context.TODO(),
		client:        client,
		bucket:        bucket,
		prefix:        prefix,
		forceSeekable: forceSeekable,
	}, nil
}
