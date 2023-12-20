// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Message struct to represent a chat message
type Message struct {
	Username string    `json:"username"`
	Text     string    `json:"text"`
	Time     time.Time `json:"time"`
}

var (
	clients     = make(map[*websocket.Conn]bool) // connected clients
	broadcast   = make(chan Message)             // broadcast channel
	chatHistory []Message                        // chat history in memory
	mongoClient *mongo.Client
)

func init() {
	// MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoURI := "mongodb://127.0.0.1:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Gagal konekasi ke mongodb")
	}
	log.Println("Berhasil koneksi ke mongodb")
	mongoClient = client
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer func() {
		conn.Close()
		delete(clients, conn)
	}()

	clients[conn] = true

	existingMessages := getMessagesFromMongoDB()
	for _, msg := range existingMessages {
		err := conn.WriteJSON(msg)
		if err != nil {
			log.Printf("Error writing JSON: %v", err)
			break
		}
	}

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading JSON: %v", err)
			break
		}

		msg.Time = time.Now()
		go saveMessageToMongoDBAsync(msg)

		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("Error writing JSON: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func saveMessageToMongoDB(msg Message) {
	collection := mongoClient.Database("chat").Collection("messages")
	_, err := collection.InsertOne(context.TODO(), msg)
	if err != nil {
		log.Printf("Error saving message to MongoDB: %v", err)
	}
}

func getMessagesFromMongoDB() []Message {
	collection := mongoClient.Database("chat").Collection("messages")
	cur, err := collection.Find(context.TODO(), bson.D{{}})
	if err != nil {
		log.Printf("Error fetching messages from MongoDB: %v", err)
		return nil
	}
	defer cur.Close(context.TODO())

	var messages []Message
	for cur.Next(context.TODO()) {
		var msg Message
		err := cur.Decode(&msg)
		if err != nil {
			log.Printf("Error decoding message from MongoDB: %v", err)
			continue
		}
		messages = append(messages, msg)
	}

	return messages
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/ws", handleConnections)

	go handleMessages()

	http.Handle("/", r)

	port := 8080
	fmt.Printf("Server started on :%d\n", port)

	// Load existing messages from MongoDB
	existingMessages := getMessagesFromMongoDB()
	for _, msg := range existingMessages {
		broadcast <- msg
	}

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func saveMessageToMongoDBAsync(msg Message) {
	go func() {
		saveMessageToMongoDB(msg)
	}()
}
