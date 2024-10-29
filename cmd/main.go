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
	"strconv"
	"time"

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

	APIPort := getEnv("APIPort", "")

	log.Printf("Starting HTTP server on :%v\n", APIPort)

	err = http.ListenAndServe(fmt.Sprintf(":%v", APIPort), r)
	if err != nil {
		log.Println("Failed to start HTTP server")
	}
}

func getData(w http.ResponseWriter) {
	RedisPort := getEnv("RedisPort", "")

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:" + RedisPort,
		Password: "",
		DB:       0,
	})

	labels := []string{}
	data := []int{}

	date := time.Date(2024, 3, 27, 0, 0, 0, 0, time.UTC)

	for date.Before(time.Now()) || date.Equal(time.Now()) {
		dateAsRedisKey := date.Format("2006-01-02")
		dateAsOutputForm := date.Format("02-01-2006")

		value, _ := rdb.Get(ctx, dateAsRedisKey).Result()

		curValue, _ := getLastMonthListeners(value)
		labels = append(labels, dateAsOutputForm)
		data = append(data, curValue)

		date = date.Add(time.Hour * 24)
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
