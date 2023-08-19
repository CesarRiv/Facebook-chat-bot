package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/cdipaolo/sentiment"
)

var model sentiment.Models

func initSentimentModel() error {
	// Initialize the sentiment analysis model
	_, err := sentiment.Restore()
	if err != nil {
		return err
	}
	return nil
}

const (
	// VERIFY_TOKEN use to verify the incoming request
	VERIFY_TOKEN = "12345"
	// ACCESS_TOKEN use to access Messenger API
	ACCESS_TOKEN = "EAADx2yJWYbEBO3ZAMdGbCaVxIPbyF9yo5kZBNHGMh2eIMVgEGHJKqNc5LpE5bv9Y25e0tFVk0znjePJeN50iyj4shDHkqig9QffdukMiXSgpTgZCeggZAc7RGS0JE3OEu8J6Kq0KN3af1ZCKUdAUL8DDVYVUBmS9ChuUci9n1i8Qaawfynhfgy3NE7UmBW27b"
	// GRAPHQL_URL is a base URL v12.0 for Messenger API
	GRAPHQL_URL = "https://graph.facebook.com/v12.0"
)

// Message data structure for message event
type Message struct {
	Object string `json:"object"`
	Entry  []struct {
		ID        string `json:"id"`
		Time      int64  `json:"time"`
		Messaging []struct {
			Sender struct {
				ID string `json:"id"`
			} `json:"sender"`
			Recipient struct {
				ID string `json:"id"`
			} `json:"recipient"`
			Timestamp int64 `json:"timestamp"`
			Message   struct {
				Mid  string `json:"mid"`
				Text string `json:"text"`
			} `json:"message"`
		} `json:"messaging"`
	} `json:"entry"`
}

// SendMessage data structure for send message
type SendMessage struct {
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
}

// webhook is a handler for Webhook server
func webhook(w http.ResponseWriter, r *http.Request) {
	// return all with status code 200
	w.WriteHeader(http.StatusOK)

	// method that allowed are GET & POST
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		log.Printf("invalid method: not get or post")
		return
	}

	// if the method of request is GET
	if r.Method == http.MethodGet {
		// read token from query parameter
		verifyToken := r.URL.Query().Get("hub.verify_token")

		// verify the token included in the incoming request
		if verifyToken != VERIFY_TOKEN {
			log.Printf("invalid verification token: %s", verifyToken)
			return
		}

		// write string from challenge query parameter
		if _, err := w.Write([]byte(r.URL.Query().Get("hub.challenge"))); err != nil {
			log.Printf("failed to write response body: %v", err)
		}

		return
	}

	// ready body in the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("failed to read body: %v", err)
		return
	}

	// initiate Message data structure to message variable
	// unmarshal []byte data into message
	var message Message
	if err := json.Unmarshal(body, &message); err != nil {
		log.Printf("Failed to unmarshal body: %v", err)
		return
	}

	//textMessage := message.Entry[0].Messaging[0].Message.Text

	// send message to end-user
	/*
	err = sendMessage(message.Entry[0].Messaging[0].Sender.ID, "Automatic Reply")
	if err != nil {
		log.Printf("failed to send message: %v", err)
	}
	*/
	/*
	// Perform sentiment analysis on the text message
	sentimentResult := model.SentimentAnalysis(textMessage, sentiment.English)

	if sentimentResult.Score == 0 {
		err = sendMessage(message.Entry[0].Messaging[0].Sender.ID, "Neuteral")
		if err != nil {
			log.Printf("failed to send message: %v", err)
		}
	} else if sentimentResult.Score > 0 {
		err = sendMessage(message.Entry[0].Messaging[0].Sender.ID, "Positive")
		if err != nil {
			log.Printf("failed to send message: %v", err)
		}
	} else {
		err = sendMessage(message.Entry[0].Messaging[0].Sender.ID, "Negative")
		if err != nil {
			log.Printf("failed to send message: %v", err)
		}
	}
	/*


	// Determine the sentiment label based on the sentiment score
	
	var sentimentLabel string
	if sentimentResult.Score == 0 {
		sentimentLabel = "Neutral"
	} else if sentimentResult.Score > 0 {
		sentimentLabel = "Positive"
	} else {
		sentimentLabel = "Negative"
	}

	responseMessage := fmt.Sprintf("Your message sentiment: %s", sentimentResult)

	err = sendMessage(message.Entry[0].Messaging[0].Sender.ID, responseMessage)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}



	if err := json.Unmarshal(body, &message); err != nil {
		log.Printf("failed to unmarshal body: %v", err)
		return
	}
*/

	return
}
// sendMessage sends a message to end-user
func sendMessage(senderId, message string) error {
	// configure the sender ID and message
	var request SendMessage
	request.Recipient.ID = senderId
	request.Message.Text = message

	// validate empty message
	if len(message) == 0 {
		return errors.New("message can't be empty")
	}

	// marshal request data
	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshall request: %w", err)
	}

	// setup http request
	url := fmt.Sprintf("%s/%s?access_token=%s", GRAPHQL_URL, "me/messages", ACCESS_TOKEN)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed wrap request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	
	// send http request
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed send request: %w", err)
	}
	defer res.Body.Close()

	// print response
	log.Printf("message sent successfully?\n%#v", res)
	
	return nil
}
func main() {
	
	err := initSentimentModel()
	if err != nil {
		fmt.Println("Error initializing sentiment analysis model:", err)
		return
	}
	
	// Read the assigned port from the environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // Default to port 3000 if not provided
	}

	// create the handler
	handler := http.NewServeMux()
	handler.HandleFunc("/", webhook)

	// configure http server
	addr := fmt.Sprintf(":%s", port)
	srv := &http.Server{
		Handler: handler,
		Addr:    addr, // Use the configured port
	}

	// start http server
	log.Printf("http server listening at %v", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}