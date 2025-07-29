package core

import "io"

type Sink interface {
	// Stores message metadata and inline content (body)
	StoreMessage(msg Message) error
	// Stores an attachment content. Note that data may be unseekable stream!
	StoreAttachment(id string, data io.Reader) error
}
