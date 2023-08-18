package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/cdipaolo/sentiment"
	"github.com/huandu/facebook"
)

const (
	verifyToken      = "YOUR_VERIFY_TOKEN"
	pageAccessToken = "YOUR_PAGE_ACCESS_TOKEN"
	appSecret       = "YOUR_APP_SECRET"
	port            = "3000"
)

func main() {
	http.HandleFunc("/webhook", webhookHandler)

	log.Printf("Server is running on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		verifyWebhook(w, r)
	} else if r.Method == http.MethodPost {
		processWebhook(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func verifyWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify the webhook with the challenge token
	if r.URL.Query().Get("hub.verify_token") == verifyToken {
		w.Write([]byte(r.URL.Query().Get("hub.challenge")))
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Verification failed"))
	}
}

func processWebhook(w http.ResponseWriter, r *http.Request) {
	// Parse incoming messages and handle them
	decoder := facebook.NewDecoder(r.Body)
	var callback facebook.Callback
	err := decoder.Decode(&callback)
	if err != nil {
		log.Println("Error decoding callback:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, entry := range callback.Entry {
		for _, messaging := range entry.Messaging {
			if messaging.Message != nil {
				handleUserMessage(messaging.Sender.ID, messaging.Message.Text)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func handleUserMessage(userID, messageText string) {
	// Perform sentiment analysis on the message
	model, err := sentiment.Restore()
	if err != nil {
		log.Println("Error restoring sentiment model:", err)
		return
	}

	analysis := model.SentimentAnalysis(messageText, sentiment.English)

	// Prepare a response based on sentiment
	var responseText string
	if analysis.Score >= 1 {
		responseText = "Thank you for your positive review!"
	} else if analysis.Score < 0{
		responseText = "We're sorry to hear that you had a negative experience. Please contact us for assistance."
	} else {
		responseText = "Thank you for your feedback!"
	}

	// Send the response to the user
	sendFacebookMessage(userID, responseText)
}

func sendFacebookMessage(userID, messageText string) {
	m := messenger.New(pageAccessToken)

	// Create a new text message
	message := messenger.NewTextMessage(messageText)

	// Send the message
	err := m.SendSimple(userID, message)
	if err != nil {
		log.Println("Error sending message:", err)
	}
}


