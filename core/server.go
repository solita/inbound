package core

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
	strip "github.com/grokify/html-strip-tags-go"
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
			main, quotedThread := splitHtmlToThread(text)
			alternative := Alternative{
				Text:         text,
				ContentType:  partType,
				Last:         main,
				QuotedThread: quotedThread,
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

func splitHtmlToThread(body string) (string, string) {
	// Quoted thread formatting is very much not standardized for HTML mail
	// ... but it is possible to do this for (some of) most common mail clients

	// Outlook/M365: divRplyFwdMsg
	outlookQuoteStart := strings.Index(body, "\r\n<div id=\"divRplyFwdMsg\"")
	if outlookQuoteStart != -1 {
		beforeVerticalLine := strings.Index(body, "<div id=\"appendonsend\"></div>\r\n")
		if beforeVerticalLine != -1 {
			// Try to strip vertical line before the quoted thread too
			return body[:beforeVerticalLine], body[beforeVerticalLine:]
		}
		// Failing that, just strip the quoted thread
		return body[:outlookQuoteStart], body[outlookQuoteStart:]
	}

	// Gmail: gmail_quote_container
	// gmail_quote_container seems to always hold the entire quoted thread
	// Inside them, further messages are wrapped just with gmail_quote divs
	gmailQuoteStart := strings.Index(body, "<div class=\"gmail_quote gmail_quote_container\">")
	if gmailQuoteStart != -1 {
		return body[:gmailQuoteStart], body[gmailQuoteStart:]
	}

	// Roundcube webmail: reply-intro
	roundcubeQuoteStart := strings.Index(body, "\u003cp id=\"reply-intro\"\u003e")
	if roundcubeQuoteStart != -1 {
		topMsg := body[:roundcubeQuoteStart]
		if strings.TrimSpace(strip.StripTags(topMsg)) == "" {
			// If there is nothing of substance on top, this might be bottom-posting which we can't handle
			// So just fall back to not splitting anything
			return body, ""
		}
		return body[:roundcubeQuoteStart], body[roundcubeQuoteStart:]
	}

	// Failed to find quoted thread; maybe it doesn't exist
	// Or maybe it was just formatted weird, but we can't know
	return body, ""
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
