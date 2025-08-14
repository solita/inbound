package core

type Message struct {
	Id         string   `json:"inbound_id"`
	MessageId  string   `json:"message_id"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	Subject    string   `json:"subject"`
	References []string `json:"references"`

	Alternatives []Alternative `json:"alternatives"`
	Attachments  []Attachment  `json:"attachments"`
}

type Alternative struct {
	ContentType string `json:"content_type"`
	Text        string `json:"text"`
}

type Attachment struct {
	Id               string `json:"id"`
	OriginalFilename string `json:"original_filename"`
}
