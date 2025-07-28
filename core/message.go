package core

type Message struct {
	Id      string
	From    string
	To      string
	Subject string

	Content string

	Attachments []Attachment
}

type Attachment struct {
	Id               string
	OriginalFilename string
}
