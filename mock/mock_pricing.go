package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/prices", func(w http.ResponseWriter, r *http.Request) {
		// Simulate network latency (between 0 and 3 seconds)
		delay := time.Duration(rand.Intn(3000)) * time.Millisecond
		time.Sleep(delay)

		// Simulate a 20% chance of complete API failure
		if rand.Float32() < 0.2 {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Return mock prices for our seeded IDs
		prices := map[string]float64{
			"prod-001": 999.99,
			"prod-002": 249.50,
			"prod-003": 349.00,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prices)
	})

	fmt.Println("⚠️  Mock Pricing API running on http://localhost:8081")
	http.ListenAndServe(":8081", nil)
}
