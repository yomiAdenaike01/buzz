package clients

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
)

type NotificationContent struct {
	Message string `json:"message"`
	Title   string `json:"title"`
	To      string `json:"to"`
}

type NotificationClient struct {
	baseUrl string
}

func (n *NotificationClient) SendNotification(body NotificationContent) error {
	request := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Path: "https://exp.host/--/api/v2/push/send",
		},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	bodyData, err := json.Marshal(body)
	if err != nil {
		return err
	}
	http.NewRequest("Post", n.baseUrl, bytes.NewBuffer(bodyData))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("[NotificationClient.SendNotification]: Failed to send request to url=%s err=%v", n.baseUrl, err)
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(response.Body)
		if err != nil {
			log.Printf("[NotificationClient.SendNotification]: Failed to send request to url=%s err=%v", n.baseUrl, err)
			return err
		}
		log.Printf("[NotificationClient.SendNotification]: Failed to send request to url=%s err=%v", n.baseUrl, err)

		return errors.New(string(respBody))
	}
	return nil
}

func NewNotificationClient() *NotificationClient {
	return &NotificationClient{
		baseUrl: "https://exp.host/--/api/v2/push/send",
	}
}
