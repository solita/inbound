package sinks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/solita/inbound/core"
)

type LocalSink struct {
	context context.Context
	path    string
}

func (s *LocalSink) StoreMessage(msg core.Message) error {
	key := s.path + "/messages/" + msg.Id + ".json"
	value, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message metadata: %w", err)
	}

	err = os.WriteFile(key, value, 0644)
	if err != nil {
		return fmt.Errorf("failed to store message metadata: %w", err)
	}
	return nil
}

func (s *LocalSink) StoreAttachment(id string, data io.Reader) error {
	key := s.path + "/attachments/" + id
	f, err := os.Create(key)
	if err != nil {
		return fmt.Errorf("failed to create attachment file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, data)
	if err != nil {
		return fmt.Errorf("failed to store attachment: %w", err)
	}
	return nil
}

var _ core.Sink = (*LocalSink)(nil)

func NewLocal(path string) (*LocalSink, error) {
	return &LocalSink{
		context: context.TODO(),
		path:    path,
	}, nil
}
