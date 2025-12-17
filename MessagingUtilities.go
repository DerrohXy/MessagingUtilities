package messagingutilities

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type atSmsResponseRecipient struct {
	status string `json:""`
}

type atSmsResponse struct {
	SMSMessageData struct {
		Message    string                   `json:"Message"`
		Recipients []atSmsResponseRecipient `json:"Recipients"`
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

	if strings.Contains(*receiver, ",") {
		return fmt.Errorf("Multiple receivers may hav been passed")
	}

	baseURL := "https://api.africastalking.com/version1/messaging"

	payload := url.Values{}
	payload.Set("username", credentials.Username)
	payload.Set("to", *receiver)
	payload.Set("from", credentials.SenderID)
	payload.Set("message", *message)

	request, err := http.NewRequest("POST", baseURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return fmt.Errorf("Failed to create http request: %w", err)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

	if len(atResp.SMSMessageData.Recipients) != 1 {
		return fmt.Errorf("Sent recipient list is empty.")
	}

	for _, atResp_ := range atResp.SMSMessageData.Recipients {
		if strings.Compare("", atResp_.status) != 0 {
			return fmt.Errorf("Message could not be sent")
		}
	}

	return nil
}
