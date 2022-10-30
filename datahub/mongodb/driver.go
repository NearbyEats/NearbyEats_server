package databasehandler

import (
	"context"
	"log"
	"time"

	"github.com/nearby-eats/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DatabaseHandler struct {
	client mongo.Client
	ctx    context.Context
}

func NewDatabaseHandler() *DatabaseHandler {
	config := utils.Config

	client, err := mongo.NewClient(options.Client().ApplyURI(config.DATABASE_URI))
	if err != nil {
		log.Panic(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	dbHandler := DatabaseHandler{client: *client, ctx: ctx}

	return &dbHandler
}

func (dbHandler *DatabaseHandler) Connect() {
    log.Println("CONNECTING TO: " + utils.Config.DATABASE_URI)

	err := dbHandler.client.Connect(dbHandler.ctx)
	if err != nil {
		log.Panic(err)
	}
}

func (dbHandler *DatabaseHandler) Disconnect() {
    log.Println("DISCONNECTING FROM: " + utils.Config.DATABASE_URI)
	dbHandler.client.Disconnect(dbHandler.ctx)
}

func (dbHandler *DatabaseHandler) ListDatabaseNames() {
	databases, err := dbHandler.client.ListDatabaseNames(dbHandler.ctx, bson.M{})
	if err != nil {
		log.Panic(err)
	}
	log.Println(databases)
}