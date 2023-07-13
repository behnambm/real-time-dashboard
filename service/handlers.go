package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type Source struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Server struct {
	Addr string
	dc   *DynamicConfig
	tmpl *template.Template
}

func (s *Server) indexHandler(w http.ResponseWriter, req *http.Request) {
	s.tmpl.Execute(w, nil)
}

func (s *Server) configHandler(w http.ResponseWriter, req *http.Request) {
	var config = struct {
		RefreshRate int `json:"refresh-rate"`
	}{}
	dec := json.NewDecoder(req.Body)
	err := dec.Decode(&config)
	if err != nil {
		w.Write([]byte("Bad request"))
		w.WriteHeader(http.StatusBadRequest)

		return
	}
}

func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("NEW CONNECTION....")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")

	channels := s.dc.GetConfig("backend/broker/channels")
	var channelsList []Source
	err := json.Unmarshal([]byte(channels), &channelsList)
	if err != nil {
		log.Println("INVALID channels VALUE")
	}
	log.Println("CHANNELS LIST : ", channelsList)

	subscriber := redisClient.Subscribe(context.Background(), "btc", "eth", "dog")
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
}
