package core

import "io"

type Sink interface {
	StoreMessage(msg Message) error
	StoreAttachment(id string, data io.Reader) error
}
