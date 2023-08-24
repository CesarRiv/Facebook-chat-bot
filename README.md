# Facebook Messenger Chatbot with Sentiment Analysis

This project is a Facebook Messenger chatbot built in Go (Golang) that uses sentiment analysis to provide appropriate responses to user messages. The chatbot communicates with the Messenger API to receive messages, analyze sentiment, and send responses back to users.

## Features

- Listens for incoming messages from users through the Facebook Messenger API.
- Performs sentiment analysis on user messages to determine the sentiment score (positive or negative).
- Generates responses based on the sentiment score and whether the user recently completed a transaction.
- Stores user responses and completed transaction status in an SQLite database.
- Sends response messages to users via the Messenger API.
