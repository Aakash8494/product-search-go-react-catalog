package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Product struct {
	ID          string  `json:"id" bson:"_id"`
	Name        string  `json:"name" bson:"name"`
	Description string  `json:"description" bson:"description"`
	Category    string  `json:"category" bson:"category"`
	Price       float64 `json:"price,omitempty"`
	PriceError  string  `json:"price_error,omitempty"`
}

var (
	rdb        *redis.Client
	mongoCol   *mongo.Collection
	ctxContext = context.Background()
)

func init() {
	// 1. Connect Redis
	rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	// 2. Connect MongoDB
	client, err := mongo.Connect(ctxContext, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("Mongo failed:", err)
	}
	mongoCol = client.Database("catalog").Collection("products")
}

func main() {
	r := gin.Default()

	r.GET("/products/search", searchHandler)

	fmt.Println("🚀 Main API running on http://localhost:8080")
	r.Run(":8080")
}

func searchHandler(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	// 1. Check Redis Cache First
	cacheKey := "search:" + query
	if cached, err := rdb.Get(ctxContext, cacheKey).Result(); err == nil {
		var cachedProducts []Product
		json.Unmarshal([]byte(cached), &cachedProducts)
		c.JSON(http.StatusOK, gin.H{"source": "redis", "data": cachedProducts})
		return
	}

	// 2. Elasticsearch Step (Mocked for brevity - assume it returns these IDs for "phone" or "monitor")
	productIDs := []string{"prod-001", "prod-002", "prod-003"}

	// 3. Concurrent Fetching (The Interview Requirement)
	metaChan := make(chan []Product)
	priceChan := make(chan map[string]float64)

	// Worker 1: Fetch Metadata from MongoDB
	go func() {
		var products []Product
		cursor, _ := mongoCol.Find(ctxContext, bson.M{"_id": bson.M{"$in": productIDs}})
		cursor.All(ctxContext, &products)
		metaChan <- products
	}()

	// Worker 2: Fetch Real-Time Pricing from our flaky Mock API
	go func() {
		client := http.Client{Timeout: 2 * time.Second} // Strict 2-second timeout
		resp, err := client.Get("http://localhost:8081/prices")
		if err != nil || resp.StatusCode != 200 {
			priceChan <- nil // Send nil if external API crashes or times out
			return
		}
		defer resp.Body.Close()
		var prices map[string]float64
		json.NewDecoder(resp.Body).Decode(&prices)
		priceChan <- prices
	}()

	// 4. Aggregate Results
	products := <-metaChan
	prices := <-priceChan

	for i, p := range products {
		if prices != nil {
			products[i].Price = prices[p.ID]
		} else {
			products[i].PriceError = "Pricing temporarily unavailable"
		}
	}

	// 5. Save to Redis Cache (30-second TTL)
	cacheData, _ := json.Marshal(products)
	rdb.Set(ctxContext, cacheKey, cacheData, 30*time.Second)

	c.JSON(http.StatusOK, gin.H{"source": "database", "data": products})
}
