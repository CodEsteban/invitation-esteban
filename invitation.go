package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func main() {
	collection := setupMongodb()
	rand.Seed(time.Now().UnixNano())
	// Get environment varibale port
	port := os.Getenv("PORT")
	if port == "" {
		println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		println("!!! please provide env variables. !!!")
		println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		log.Panic()
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	contx := context.WithValue(ctx, mongo.Collection{}, collection)

	// Basic http server
	mux := http.NewServeMux()
	mux.HandleFunc("/i/new", newInvitation)
	mux.HandleFunc("/i/use", useInvitation)
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context {
			return contx
		},
	}
	server.ListenAndServe()
}

func badRequest(err error, w *http.ResponseWriter) bool {
	if err != nil {
		fmt.Println(err)
		http.Error(*w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return true
	}
	return false
}

func internalError(err error, w *http.ResponseWriter) bool {
	if err != nil {
		fmt.Println(err)
		http.Error(*w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return true
	}
	return false
}

func forbidden(err error, w *http.ResponseWriter) bool {
	if err != nil {
		fmt.Println(err)
		http.Error(*w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return true
	}
	return false
}

type Invitation struct {
	Id  string `json:"id"`
	Who int    `json:"who"`
}

func useInvitation(w http.ResponseWriter, r *http.Request) {
	// Gets Thoughts List from context

	body, err := io.ReadAll(r.Body)
	if badRequest(err, &w) {
		return
	}

	var inv *Invitation
	err = json.Unmarshal(body, &inv)
	if badRequest(err, &w) {
		return
	}

	if inv.Id == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	collection := ctx.Value(mongo.Collection{}).(*mongo.Collection)

	filter := bson.D{{Key: "id", Value: inv.Id}}
	var result bson.D
	err = collection.FindOne(context.TODO(), filter).Decode(&result)
	if forbidden(err, &w) {
		return
	}
	val, err := bson.Marshal(result)
	if internalError(err, &w) {
		return
	}
	_, err = collection.DeleteOne(context.TODO(), filter)
	if internalError(err, &w) {
		return
	}

	fmt.Println(string(val))
	w.Write([]byte{})
}
func newInvitation(w http.ResponseWriter, r *http.Request) {
	// Gets Thoughts List from context

	body, err := io.ReadAll(r.Body)
	if badRequest(err, &w) {
		return
	}
	fmt.Println(string(body))
	var inv *Invitation
	err = json.Unmarshal(body, &inv)
	if badRequest(err, &w) {
		return
	}

	if inv.Who == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	collection := ctx.Value(mongo.Collection{}).(*mongo.Collection)

	randomNumber := strconv.Itoa(rand.Intn(999999999999))
	_, err = collection.InsertOne(context.TODO(),
		bson.D{
			{Key: "id", Value: randomNumber},
			{Key: "who", Value: inv.Who}})
	if internalError(err, &w) {
		return
	}

	w.Write([]byte(randomNumber))
}

func handleError(err error) {
	if err != nil {
		println("-----------------")
		println("|invitation died.|")
		println("-----------------")
		fmt.Print(err)
		log.Panic()
	}
}
func setupMongodb() *mongo.Collection {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://mongodb:27017"))
	handleError(err)
	err = client.Ping(context.TODO(), readpref.Primary())
	handleError(err)

	return client.Database("esteban").Collection("invitation")
}
