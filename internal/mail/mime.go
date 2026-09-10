package mail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"
	"time"
)

func BuildMIME(from, to, subject, body, pdfName string, pdfData []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	headers := textproto.MIMEHeader{}
	headers.Set("To", to)
	headers.Set("From", from)
	headers.Set("Subject", mime.QEncoding.Encode("utf-8", subject))
	headers.Set("MIME-Version", "1.0")
	headers.Set("Content-Type", "multipart/mixed; boundary="+writer.Boundary())
	headers.Set("Date", time.Now().Format(time.RFC1123Z))

	for k, v := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, "")))
	}
	buf.WriteString("\r\n")

	textPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=utf-8"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := textPart.Write([]byte(body)); err != nil {
		return nil, err
	}

	attachPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"application/pdf"},
		"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, pdfName)},
		"Content-Transfer-Encoding": {"base64"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := attachPart.Write(encodeBase64Lines(pdfData)); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeBase64Lines(data []byte) []byte {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encoded, data)

	var out bytes.Buffer
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		out.Write(encoded[i:end])
		out.WriteString("\r\n")
	}
	return out.Bytes()
}
