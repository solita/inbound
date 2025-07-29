package sinks

import (
	"io"
	"log/slog"

	"github.com/solita/inbound/core"
)

type LoggingSink struct{}

func (s *LoggingSink) StoreMessage(msg core.Message) error {
	attachmentIds := make([]string, len(msg.Attachments))
	for i, att := range msg.Attachments {
		attachmentIds[i] = att.Id
	}
	slog.Info("Received message", "id", msg.Id, "attachments", attachmentIds)
	return nil
}

func (s *LoggingSink) StoreAttachment(id string, data io.Reader) error {
	slog.Info("Received attachment", "id", id)
	return nil
}

var _ core.Sink = (*LoggingSink)(nil)
