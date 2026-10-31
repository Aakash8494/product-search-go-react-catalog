package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Product struct {
	ID          string `bson:"_id" json:"id"`
	Name        string `bson:"name" json:"name"`
	Description string `bson:"description" json:"description"`
	Category    string `bson:"category" json:"category"`
}

func main() {
	ctx := context.Background()

	// 1. Connect to MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("Mongo connection failed:", err)
	}
	defer mongoClient.Disconnect(ctx)
	collection := mongoClient.Database("catalog").Collection("products")

	// 2. Connect to Elasticsearch
	esClient, err := elasticsearch.NewDefaultClient()
	if err != nil {
		log.Fatal("Elasticsearch connection failed:", err)
	}

	// 3. Dummy Data
	products := []Product{
		{ID: "prod-001", Name: "iPhone 16 Pro", Description: "Apple smartphone with A18 Pro chip", Category: "Electronics"},
		{ID: "prod-002", Name: "LG UltraGear 24-inch", Description: "144Hz Gaming Monitor with IPS panel", Category: "Electronics"},
		{ID: "prod-003", Name: "Nothing Phone (3a) Pro", Description: "Android smartphone with custom glyph interface", Category: "Electronics"},
	}

	// 4. Insert into both databases
	for _, p := range products {
		// Insert into Mongo
		_, err := collection.InsertOne(ctx, p)
		if err != nil {
			fmt.Println("Already exists in Mongo or error:", err)
		}

		// Insert into Elasticsearch
		body := fmt.Sprintf(`{"name": "%s", "description": "%s", "category": "%s"}`, p.Name, p.Description, p.Category)
		res, _ := esClient.Index("products", strings.NewReader(body), esClient.Index.WithDocumentID(p.ID))
		res.Body.Close()
	}

	fmt.Println("✅ Data successfully seeded into MongoDB and Elasticsearch!")
}
