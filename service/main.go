package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
	"html/template"
	"log"
	"net/http"
	"os"
)

func GetConfig(key string) []byte {
	dfConfig := api.DefaultConfig()
	dfConfig.Address = os.Getenv("CONSUL_ADDRESS")
	consulClient, err := api.NewClient(dfConfig)
	if err != nil {
		log.Println(err)
	}

	kv := consulClient.KV()
	pair, _, err := kv.Get(key, nil)
	if err != nil {
		log.Println(err)
	}

	if pair == nil {
		return nil
	}

	return pair.Value
}

type Coordinate struct {
	X string `json:"x"`
	Y string `json:"y"`
}

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_HOST"), Password: "", DB: 0})
	stat := rdb.Ping(context.Background())
	if stat.Err() != nil {
		panic(stat.Err())
	}

	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		s := GetConfig("BROKER/HOST")
		fmt.Println("CONFIG: ", string(s))
		tmpl.Execute(w, nil)
	})

	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		log.Println("NEW CONNECTION....")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Cache-Control", "no-cache")
		w.Header().Add("Connection", "keep-alive")
		w.Header().Set("Content-Type", "text/event-stream")

		subscriber := rdb.Subscribe(context.Background(), "btc", "eth", "dog")
		for {
			message, _ := subscriber.ReceiveMessage(context.Background())
			var v []Coordinate
			err := json.Unmarshal([]byte(message.Payload), &v)
			if err != nil {
				log.Println("JSON :", err)
				return
			}

			fmt.Fprintf(w, "event: %s\n", message.Channel)
			fmt.Fprintf(w, "data: %s\n\n", message.Payload)
			w.(http.Flusher).Flush()
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
