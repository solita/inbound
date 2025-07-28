package sinks

import (
	"io"
	"log/slog"

	"github.com/solita/inbound/core"
)

type LoggingSink struct{}

func (s *LoggingSink) StoreMessage(msg core.Message) error {
	slog.Info("Received message", "data", msg)
	return nil
}

func (s *LoggingSink) StoreAttachment(id string, data io.Reader) error {
	slog.Info("Received attachment", "id", id)
	return nil
}

var _ core.Sink = (*LoggingSink)(nil)
