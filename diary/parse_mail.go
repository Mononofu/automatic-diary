package diary

import (
	"fmt"
	"github.com/sloonz/go-qprintable"
	"io/ioutil"
	"strings"
)

type Content struct {
	ContentType string
	Content     interface{}
}

type Mail struct {
	Headers     map[string]string
	Plaintext   string
	HTML        string
	Attachments []AttachmentJSON
	Content     []Content
}

func parse_mail(mail string) (Mail, error) {
	lines := strings.Split(mail, "\n")

	// parse headers
	headers := make(map[string]string)
	curHeaderName := ""

	for _, line := range lines {
		if strings.Contains(line, "Content-Type") { // end of headers
			break
		}

		if line[0] == ' ' { // continue header from before
			headers[curHeaderName] += strings.TrimLeft(line, " ")
		} else { // new header
			parts := strings.SplitN(line, ":", 2)
			headerName := parts[0]
			headerValue := parts[1]

			headers[headerName] = headerValue
			curHeaderName = headerName
		}
	}

	// parse mail body
	contents, err := parseBodyPart(mail)
	if err != nil {
		return Mail{}, fmt.Errorf("Failed to parse body: %v", err)
	}

	m := Mail{
		Headers:     headers,
		Attachments: []AttachmentJSON{},
	}

	for _, content := range contents {
		if content.ContentType == "text/plain" {
			m.Plaintext = content.Content.(string)
		} else if content.ContentType == "text/html" {
			m.HTML = content.Content.(string)
		} else {
			m.Attachments = append(m.Attachments, content.Content.(AttachmentJSON))
		}
	}

	return m, nil
}

func parseBodyPart(part string) ([]Content, error) {
	// our own content-type
	contentType, err := FindStringNthSubmatch(part, "Content-Type: ([^;]+);", 1)
	if err != nil {
		return nil, fmt.Errorf("Failed to match content type: %v \n\n for part:\n%v \n\n\n", err, part)
	}

	part = strings.Trim(part, "- \n")

	if contentType == "multipart/mixed" || contentType == "multipart/alternative" {
		return parseMultipart(part)
	} else if contentType == "text/plain" {
		return parseTextPlain(part)
	} else if contentType == "text/html" {
		return parseTextHtml(part)
	}
	return parseAttachment(part, contentType)
}

func parseMultipart(part string) ([]Content, error) {
	// split ourselves and parse recursively
	// Content-Type: multipart/alternative; boundary=e89a8f23549da6dffb04d395914c 
	boundary, err := FindStringNthSubmatch(part, "Content-Type: [^;]+; boundary=([a-z0-9]+)", 1)
	if err != nil {
		return nil, fmt.Errorf("Failed to match boundary: %v", err)
	}

	// the first part is before the boundary definition (includes the content-type),
	// the second part is from boundary definition to the first actual boundary
	// so we can safely skip those
	// the last part is also empty, so skip that too
	messageParts := strings.Split(part, boundary)
	messageParts = messageParts[2 : len(messageParts)-1]

	parts := []Content{}

	for _, messagePart := range messageParts {
		content, err := parseBodyPart(messagePart)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse multipart: %v \n\n for part:\n%v \n\n\n", err, part)
		}
		parts = concat(parts, content)
	}

	return parts, nil
}

// ignore HTML for now
func parseTextHtml(part string) ([]Content, error) {
	return []Content{}, nil
}

func parseAttachment(part string, contentType string) ([]Content, error) {

	name, err := FindStringNthSubmatch(part, `Content-Disposition: attachment; filename="([^"]+)"`, 1)
	if err != nil {
		return nil, fmt.Errorf("Failed to match filename: %v", err)
	}

	lastHeader := strings.Index(part, "X-Attachment-Id")
	part = part[lastHeader:]
	contentStart := strings.Index(part, "\n")
	part = part[contentStart:]

	rawContent := strings.Trim(part, " \n\r-")
	withoutNewlines := strings.Replace(rawContent, "\n", "", -1)
	withoutSpaces := strings.Replace(withoutNewlines, " ", "", -1)

	return []Content{
		Content{
			ContentType: contentType,
			Content: AttachmentJSON{
				ContentType: contentType,
				Name:        name,
				Content:     withoutSpaces,
			},
		},
	}, nil

}

func parseTextPlain(part string) ([]Content, error) {
	// parse the headers at the beginning
	for strings.Contains(part, ":") {
		parts := strings.SplitN(part, "\n", 2)

		header := parts[0]
		part = parts[1]

		if !strings.Contains(header, ":") { // : is in body text
			break
		}

		parts = strings.SplitN(header, ":", 2)
		headerName := parts[0]
		headerValue := parts[1]

		if headerName == "Content-Type" && !strings.Contains(headerValue, "text/plain") {
			return nil, fmt.Errorf("unexpected content type: %v", headerValue)
		} else if headerName == "Content-Transfer-Encoding" && !strings.Contains(headerValue, "quoted-printable") {
			return nil, fmt.Errorf("unexpected content encoding: %v", headerValue)
		}
	}

	decoder := qprintable.NewDecoder(qprintable.DetectEncoding(part),
		strings.NewReader(part))

	decodedBytes, err := ioutil.ReadAll(decoder)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode message: %v", err)
	}

	plaintext := string(decodedBytes)

	plaintext = strings.Trim(plaintext, " \n")

	return []Content{
		Content{
			ContentType: "text/plain",
			Content:     plaintext,
		},
	}, nil
}

func concat(old1, old2 []Content) []Content {
	newslice := make([]Content, len(old1)+len(old2))
	copy(newslice, old1)
	copy(newslice[len(old1):], old2)
	return newslice
}
