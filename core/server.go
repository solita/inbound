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

	_ "github.com/emersion/go-message/charset"
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

	alternatives := make([]Alternative, 0)
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
			// Inline = one variation of message content/body
			content, err := io.ReadAll(p.Body)
			if err != nil {
				return handleError(fmt.Errorf("failed to read message content: %w", err))
			}
			text := string(content)
			// Remove trailing newlines that some email clients sometimes add
			text = strings.TrimRight(text, "\r\n")

			partType, _, err := h.ContentType()
			if err != nil {
				return handleError(fmt.Errorf("failed to parse content type: %w", err))
			}
			alternative := Alternative{
				Text:        text,
				ContentType: partType,
			}
			alternatives = append(alternatives, alternative)
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

	// Store message metadata
	subject, err := mr.Header.Subject()
	if err != nil {
		return handleError(fmt.Errorf("failed to parse email subject: %w", err))
	}
	messageId, err := mr.Header.MessageID()
	if err != nil {
		return handleError(fmt.Errorf("failed to parse message ID: %w", err))
	}

	msg := Message{
		Id:           uuid.New().String(),
		MessageId:    messageId,
		From:         s.from,
		To:           s.to,
		Subject:      subject,
		Alternatives: alternatives,
		Attachments:  attachments,
		References:   parseReferences(mr.Header),
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

func parseReferences(header mail.Header) []string {
	// Most mail clients should include both References and In-Reply-To
	references := header.Get("References")
	inReplyTo := header.Get("In-Reply-To")
	refArray := strings.Fields(references)

	// But in case the references don't include last (or anything at all)
	// just add it there
	if len(refArray) == 0 || refArray[len(refArray)-1] != inReplyTo {
		refArray = append(refArray, inReplyTo)
	}

	// Remove < and > from the references so they are only Message-IDs
	for i, ref := range refArray {
		ref = strings.TrimPrefix(ref, "<")
		ref = strings.TrimSuffix(ref, ">")
		refArray[i] = ref
	}

	return refArray
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
