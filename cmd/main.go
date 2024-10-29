package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var ctx = context.Background()

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("No .env file found or unable to load it: %v", err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})
	r.HandleFunc("/chart-data", func(w http.ResponseWriter, r *http.Request) {
		getData(w)
	})

	APIPort := getEnv("APIPort", "8000")

	log.Printf("Starting HTTP server on :%v\n", APIPort)

	err = http.ListenAndServe(fmt.Sprintf(":%v", APIPort), r)
	if err != nil {
		log.Println("Failed to start HTTP server")
	}
}

func getData(w http.ResponseWriter) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	keys, err := rdb.Keys(ctx, "*").Result()
	if err != nil {
		log.Fatalf("Failed to fetch keys: %v", err)
	}

	sort.Strings(keys)

	labels := make([]string, 0, len(keys))
	data := make([]int, 0, len(keys))

	for _, key := range keys {
		value, err := rdb.Get(ctx, key).Result()
		if err == redis.Nil {
			continue
		} else if err != nil {
			log.Fatalf("Failed to get value for key %s: %v", key, err)
		}

		curValue, _ := getLastMonthListeners(value)
		labels = append(labels, key)
		data = append(data, curValue)
	}

	chartData := map[string]interface{}{
		"labels": labels,
		"datasets": []map[string]interface{}{
			{
				"label":       "Last Month Listeners",
				"data":        data,
				"borderColor": "rgba(75, 192, 192, 1)",
				"fill":        false,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(chartData); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getLastMonthListeners(data string) (int, error) {
	re := regexp.MustCompile(`"lastMonthListeners\\\":(\d+)`)

	match := re.FindStringSubmatch(data)
	if len(match) == 0 {
		return 0, errors.New("no match found")
	}

	result, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, errors.New("failed to convert parsed data to int")
	}

	return result, nil
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
