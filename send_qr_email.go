package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/skip2/go-qrcode"
	"gopkg.in/gomail.v2"
)

func sendQREmail(params ...string) error {
	// Decode Base64 QR code image data
	var emailAddress string
	m := gomail.NewMessage()
	if params[0] == "" {
		emailAddress = os.Getenv(manager_email)
	}

	// Compose the email message
	body := fmt.Sprintf(`<p>%s</p><b/r><img src="cid:qrcode" alt="scna the qrcode" />`, `Dear Manager,

The WhatsApp device session has ended or the number has been banned

Please find the QR code attached for further action.

Regards,
Your Name`)
	m.SetHeader("From", os.Getenv(gmail_email))
	m.SetHeader("To", emailAddress)
	m.SetDateHeader("X-Date", time.Now())
	m.SetHeader("Subject", " WhatsApp Device Session failed")
	m.SetBody("text/html", body)
	contentID := "qrcode"
	attachment := gomail.SetHeader(map[string][]string{"Content-ID": {fmt.Sprintf("<%s>", contentID)}})
	m.Attach("qrcode.png", gomail.SetCopyFunc(func(w io.Writer) error {
		qrcode, __err := qrcode.New(params[1], qrcode.Highest)
		if __err != nil {
			return __err
		}
		return qrcode.Write(512, w)
	}), attachment)

	d := gomail.NewDialer("smtp.gmail.com", 587, os.Getenv(gmail_email), os.Getenv(gmail_password))

	// Send the email to Bob, Cora and Dan.
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("error sending email: %v", err)
	}
	return nil
}

func test_qr_code_email() {
	qrCodeBase64 := "base64_encoded_data"

	err := sendQREmail("", qrCodeBase64)
	if err != nil {
		fmt.Printf("Error sending email: %v", err)
		return
	}

	fmt.Println("Email sent successfully!")
}
