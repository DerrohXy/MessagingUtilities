package messagingutilities

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/twilio/twilio-go"
	TWILIO_API "github.com/twilio/twilio-go/rest/api/v2010"
	"gopkg.in/gomail.v2"
)

//lint:file-ignore ST1005 TF

type SMTPCredentials struct {
	Host     string
	Port     string
	User     string
	Sender   string
	Password string
	UseTLS   bool
}

type EmailAttachment struct {
	Data io.Reader
	Name *string
}

func SendSMTPEmailMessage(
	credentials *SMTPCredentials,
	subject,
	message *string,
	isHtml bool,
	attachments *[]EmailAttachment,
	receivers *[]string,
) error {
	message_ := gomail.NewMessage()

	message_.SetHeader("From", credentials.Sender)
	message_.SetHeader("To", *receivers...)
	if subject != nil {
		message_.SetHeader("Subject", *subject)
	}

	mimeType := "text/plain"
	if isHtml {
		mimeType = "text/html"
	}
	if message != nil {
		message_.SetBody(mimeType, *message)
	}

	if attachments != nil {
		for _, reader := range *attachments {
			message_.Attach(
				*reader.Name,
				gomail.SetHeader(map[string][]string{
					"Content-Type": {"application/octet-stream"},
				}),
				gomail.SetCopyFunc(func(w io.Writer) error {
					_, err := io.Copy(w, reader.Data)
					return err
				}),
			)
		}
	}

	port, err := strconv.Atoi(credentials.Port)
	if err != nil {
		return fmt.Errorf("Invalid port number: %w", err)
	}

	dialer := gomail.NewDialer(
		credentials.Host,
		port,
		credentials.User,
		credentials.Password,
	)

	if credentials.UseTLS {
		//
	}

	return dialer.DialAndSend(message_)
}

type TwilioCredentials struct {
	AccountSID        string
	AuthToken         string
	SenderPhoneNumber string
	SenderName        string
}

func SendTwilioSmsMessage(
	credentials *TwilioCredentials,
	message *string,
	receiver *string,
) error {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: credentials.AccountSID,
		Password: credentials.AuthToken,
	})

	if message == nil {
		return fmt.Errorf("Message body and receivers cannot be empty")
	}

	params := &TWILIO_API.CreateMessageParams{}
	params.SetBody(*message)
	params.SetFrom(credentials.SenderPhoneNumber)
	params.SetTo(*receiver)

	_, err := client.Api.CreateMessage(params)

	return err
}

type AfricasTalkingCredentials struct {
	ApiKey   string
	Username string
	SenderID string
}

type atSmsRequest struct {
	Username string `json:"username"`
	To       string `json:"to"` // Comma-separated list of recipients
	Message  string `json:"message"`
	From     string `json:"from,omitempty"` // Optional sender ID
}

type atSmsResponse struct {
	SMSMessageData struct {
		Message string `json:"Message"`
	} `json:"SMSMessageData"`
}

func SendAfricasTalkingSmsMessage(
	credentials *AfricasTalkingCredentials,
	message *string,
	receiver *string,
) error {
	if message == nil {
		return fmt.Errorf("Message body cannot be empty")
	}

	baseURL := "https://api.africastalking.com"
	url := baseURL + "/version1/messaging"

	payload := atSmsRequest{
		Username: credentials.Username,
		To:       *receiver,
		Message:  *message,
		From:     credentials.SenderID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Failed to marshal JSON payload: %w", err)
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("Failed to create http request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("apiKey", credentials.ApiKey)

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("Failed to execute http request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)

		return fmt.Errorf(
			"Africa's talking API failed with status %d. Response body: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	var atResp atSmsResponse
	if err := json.NewDecoder(resp.Body).Decode(&atResp); err != nil {
		return fmt.Errorf("Successfully sent, but failed to parse response: %w", err)
	}

	if strings.Contains(atResp.SMSMessageData.Message, "Total Cost: KES 0.00") {
		return fmt.Errorf(
			"africa's talking send likely failed. Check API key/username. Response: %s",
			atResp.SMSMessageData.Message,
		)
	}

	return nil
}
