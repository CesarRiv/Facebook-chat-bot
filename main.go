package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"math/rand"

	"github.com/cdipaolo/sentiment"
	_ "github.com/mattn/go-sqlite3"
)

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
var db *sql.DB

// Function to randomly determine if the user recently completed a transaction
func recentlyCompletedTransaction() bool {
	// Generate a random number between 0 and 1
	// If the number is greater than or equal to 0.5, return true
	// Otherwise, return false
	return rand.Float64() >= 0.5
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
			return
		}
		
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
	textMessage := message.Entry[0].Messaging[0].Message.Text
	sendResponseMessage(message.Entry[0].Messaging[0].Sender.ID, textMessage)
}
// determineResponseMessage determines the response message based on sentiment and completed transaction status.
func determineResponseMessage(message string, completedTransaction bool) string {
	// Restore the sentiment model
	sentimentModel, err := sentiment.Restore()
	if err != nil {
		log.Printf("Error initializing sentiment model: %v", err)
		return "Error when initializing sentiment model"
	}
	responseMessage := ""
	completedTransactionInt := 0 // Default to 0
	//Calling model to do sentimental analysis
	results := sentimentModel.SentimentAnalysis(message, sentiment.English)
	// Convert the boolean to an integer (0 or 1)
	if completedTransaction {
		completedTransactionInt = 1
	}
	// Determine the response message based on sentiment score
	if results.Score == 1 && completedTransactionInt == 1 {
		responseMessage = "Thank you for recently purchasing with us and I am glad to hear you had a positive experience with our product!"
	} else if results.Score == 0 && completedTransactionInt == 1 {
		responseMessage = "Thank you for recently purchasing with us and I am sorry to hear your experience wasn't the greatest with our product."
	} else if results.Score == 1 && completedTransactionInt == 0 {
		responseMessage = "Seems like there is no recent transaction tied with your account, what is the product you purchased which you had a positive experience with?"
		completedTransactionInt = 1
	} else if results.Score == 0 && completedTransactionInt == 0{
		responseMessage = "Seems like there is no recent transaction tied with your account, what is the product you purchased which you had a negative experience with?"
		completedTransactionInt = 1
	}
	return responseMessage
}
func sendResponseMessage(senderID, message string) {
	// Determine if the user recently completed a transaction
	completedTransaction := recentlyCompletedTransaction()
	completedTransactionInt := 0 // Default to 0

	// Convert the boolean to an integer (0 or 1)
	if completedTransaction {
		completedTransactionInt = 1
	}
	//Calling Sentimental Analysis function and determining response
	responseMessage := determineResponseMessage(message, completedTransaction)
	// Send the response message to the user
	if err := sendMessage(senderID, responseMessage); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
	// Store the response in the database
	_, err := db.Exec(`
		INSERT INTO responses (sender_id, response_text, completed_transaction)
		VALUES (?, ?, ?)`,
		senderID, responseMessage, completedTransactionInt)
	if err != nil {
		log.Printf("Failed to store response in database: %v", err)
	}
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
	log.Printf("message sent successfully!\n%#v", res)
	return nil
}
// getStoredResponses retrieves stored responses from the database and logs them.
func getStoredResponses() {
	// Query the database to retrieve sender IDs, response texts, and completed transaction status
	rows, err := db.Query(`
		SELECT sender_id, response_text, completed_transaction
		FROM responses`)
	if err != nil {
		log.Printf("Failed to retrieve responses: %v", err)
		return
	}
	defer rows.Close() // Ensure the rows are closed when done with them
	// Iterate through the retrieved rows
	for rows.Next() {
		var senderID string
		var responseText string
		var completedTransaction int
		// Attempt to scan the values from the current row into variables
		if err := rows.Scan(&senderID, &responseText, &completedTransaction); err != nil {
			log.Printf("Failed to retrieve row: %v", err)
			continue // Continue to the next row in case of an error
		}
		// Log the retrieved sender ID, response text, and completed transaction status
		log.Printf("Sender: %s, Response: %s, Completed Transaction: %d", senderID, responseText, completedTransaction)
	}
}
func main() {
	// Read the assigned port from the environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // Default to port 3000 if not provided
	}
	// Open the SQLite database with the specified file name "responses.db"
	var err error
	db, err = sql.Open("sqlite3", "responses.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close() // Ensure the database is closed when we're done with it

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS responses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender_id TEXT,
			response_text TEXT,
			completed_transaction INTEGER
		)`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
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