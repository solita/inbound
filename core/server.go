package core

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
	"github.com/solita/inbound/metrics"
)

type session struct {
	sinks        []Sink
	errorHandler func(error)
	metrics      metrics.Collector

	from string
	to   string

	startTime int64
}

func (s *session) Reset() {}

func (s *session) Logout() error {
	return nil
}

func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	s.startTime = time.Now().UnixMilli()
	s.from = from
	return nil
}

func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = to
	return nil
}

func (s *session) Data(r io.Reader) error {
	handleError := func(err error) error {
		s.errorHandler(err)
		return err
	}

	mr, err := mail.CreateReader(r)
	if err != nil {
		return handleError(fmt.Errorf("failed to parse incoming mail: %w", err))
	}

	// Mail with attachments will be delivered in multipart/mixed format
	// Go through parts to extract message content and attachments
	textBody := ""
	contentType := ""
	attachments := make([]Attachment, 0)
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return handleError(fmt.Errorf("failed to parse email part: %w", err))
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// Inline = message content/body
			text, err := io.ReadAll(p.Body)
			if err != nil {
				return handleError(fmt.Errorf("failed to read message content: %w", err))
			}
			// Multiple inline parts probably shouldn't happen, but handle it in any case
			textBody += string(text)

			partType, _, err := h.ContentType()
			if err != nil {
				return handleError(fmt.Errorf("failed to parse content type: %w", err))
			}
			if contentType == "" {
				contentType = partType
			} else if contentType != partType {
				contentType = "multipart/mixed"
			}
		case *mail.AttachmentHeader:
			// Attachment file
			filename, err := h.Filename()
			if err != nil {
				return handleError(fmt.Errorf("failed to parse attachment filename: %w", err))
			}
			id := uuid.New().String()

			// Store attachment content (streaming)
			for _, sink := range s.sinks {
				if err = sink.StoreAttachment(id, p.Body); err != nil {
					return handleError(fmt.Errorf("failed to store attachment %q: %w", filename, err))
				}
			}

			// Store attachment metadata in what will be Message
			attachments = append(attachments, Attachment{
				Id:               id,
				OriginalFilename: filename,
			})
		}
	}
	// Strip MIME boundary from end of text body
	textBody = strings.TrimRight(textBody, "\r\n")

	// Store message metadata
	subject, err := mr.Header.Subject()
	if err != nil {
		return handleError(fmt.Errorf("failed to parse email subject: %w", err))
	}
	msg := Message{
		Id:          uuid.New().String(),
		From:        s.from,
		To:          s.to,
		Subject:     subject,
		Content:     textBody,
		ContentType: contentType,
		Attachments: attachments,
	}
	for _, sink := range s.sinks {
		if err = sink.StoreMessage(msg); err != nil {
			return handleError(fmt.Errorf("failed to store message metadata: %w", err))
		}
	}

	duration := time.Now().UnixMilli() - s.startTime
	if s.metrics != nil {
		s.metrics.ReceiveSuccess(duration)
	}
	return nil
}

type ServerConfig struct {
	Listen string
}

func NewServer(sinks []Sink, errorHandler func(error), metrics metrics.Collector) *smtp.Server {
	backend := smtp.BackendFunc(func(c *smtp.Conn) (smtp.Session, error) {
		return &session{
			sinks:        sinks,
			errorHandler: errorHandler,
			metrics:      metrics,
		}, nil
	})

	return smtp.NewServer(backend)
}
