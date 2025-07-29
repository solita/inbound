package core

type Message struct {
	Id      string `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`

	Content     string `json:"content"`
	ContentType string `json:"content_type"`

	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Id               string `json:"id"`
	OriginalFilename string `json:"original_filename"`
}
