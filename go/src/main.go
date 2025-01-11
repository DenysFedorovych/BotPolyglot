package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// TelegramUpdate represents the structure of the incoming Telegram webhook payload
type TelegramUpdate struct {
	Message struct {
		MessageID  int `json:"message_id"`
		SenderChat *struct {
			ID       int64  `json:"id"`
			Title    string `json:"title"`
			Username string `json:"username"`
			Type     string `json:"type"`
		} `json:"sender_chat,omitempty"`
		Chat struct {
			ID    int64  `json:"id"`
			Title string `json:"title"`
			Type  string `json:"type"`
		} `json:"chat"`
		Text            string `json:"text,omitempty"`
		Caption         string `json:"caption,omitempty"`
		ForwardFromChat *struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Type     string `json:"type"`
		} `json:"forward_from_chat,omitempty"`
		ForwardFromMessageID int  `json:"forward_from_message_id,omitempty"`
		IsAutomaticForward   bool `json:"is_automatic_forward,omitempty"`
	} `json:"message"`
}

// TelegramMessageResponse represents the structure for sending messages to Telegram
type TelegramMessageResponse struct {
	ChatID           int64  `json:"chat_id"`
	Text             string `json:"text"`
	ReplyToMessageID int    `json:"reply_to_message_id"`
}

// translateText uses the DeepL API to translate text
func translateText(text, targetLang, deepLAuthKey string) (string, error) {
	apiURL := "https://api-free.deepl.com/v2/translate"
	payload := map[string]interface{}{
		"text":        []string{text},
		"target_lang": targetLang,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling DeepL request: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("error creating DeepL request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "DeepL-Auth-Key "+deepLAuthKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to DeepL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := ioutil.ReadAll(resp.Body)
		log.Printf("DeepL API Error - Status: %s, Body: %s", resp.Status, string(responseBody))
		return "", fmt.Errorf("DeepL API responded with status: %s", resp.Status)
	}

	var translationResponse struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}
	err = json.NewDecoder(resp.Body).Decode(&translationResponse)
	if err != nil {
		return "", fmt.Errorf("error decoding DeepL response: %w", err)
	}

	if len(translationResponse.Translations) == 0 {
		return "", fmt.Errorf("no translations returned from DeepL")
	}

	return translationResponse.Translations[0].Text, nil
}

// HandleRequest processes the Telegram webhook and sends a translated response for channel posts
func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	// Here is the version printed for debugging purposes
	log.Printf("Running version: %d", 1)

	// Validate the X-Telegram-Bot-Api-Secret-Token header
	botSecret := os.Getenv("BOT_SECRET")
	if request.Headers["X-Telegram-Bot-Api-Secret-Token"] != botSecret {
		log.Println("Invalid or missing X-Telegram-Bot-Api-Secret-Token header")
		return &events.APIGatewayProxyResponse{
			StatusCode: 403,
			Body:       `{"status": "forbidden"}`,
		}, nil
	}

	// Parse the incoming request body
	var update TelegramUpdate
	err := json.Unmarshal([]byte(request.Body), &update)
	if err != nil {
		log.Printf("Error parsing Telegram update: %v", err)
		return &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       `{"status": "invalid request"}`,
		}, nil
	}

	message := update.Message
	// Validate that the message originates from the channel
	if !message.IsAutomaticForward || message.ForwardFromChat == nil || message.ForwardFromChat.Type != "channel" {
		log.Println("Received message is not a forwarded channel post. Ignoring.")
		return &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       `{"status": "message ignored"}`,
		}, nil
	}

	// Use text or caption for translation
	textToTranslate := message.Text
	if textToTranslate == "" {
		textToTranslate = message.Caption
	}
	if textToTranslate == "" {
		log.Println("Message text or caption is empty. Ignoring.")
		return &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       `{"status": "message ignored"}`,
		}, nil
	}

	// Get environment variables
	targetLang := os.Getenv("TARGET_LANG")
	deepLAuthKey := os.Getenv("DEEPL_AUTH_KEY")
	if targetLang == "" || deepLAuthKey == "" {
		log.Printf("Environment variables TARGET_LANG or DEEPL_AUTH_KEY are missing")
		return &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       `{"status": "missing environment variables"}`,
		}, nil
	}

	// Translate the text using DeepL
	translatedText, err := translateText(textToTranslate, targetLang, deepLAuthKey)
	if err != nil {
		log.Printf("Translation error: %v", err)
		translatedText = "Translation failed. Unable to process the request."
	}

	// Create the response payload to send back to Telegram
	response := TelegramMessageResponse{
		ChatID:           message.Chat.ID,
		Text:             translatedText,
		ReplyToMessageID: message.MessageID, // Reply to the forwarded message in the discussion group
	}

	// Send the response back to Telegram
	botToken := os.Getenv("BOT_TOKEN")
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling Telegram response: %v", err)
		return &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       `{"status": "response marshalling error"}`,
		}, nil
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(responseBody))
	if err != nil {
		log.Printf("Error sending message to Telegram: %v", err)
	} else {
		log.Printf("Telegram API response status: %s", resp.Status)
	}

	// Return success response to Telegram webhook
	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"status": "message processed"}`,
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
