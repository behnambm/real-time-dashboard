package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var ConsulClient *api.Client

type Coordinate struct {
	X string `json:"x"`
	Y string `json:"y"`
}

var (
	tmpl        *template.Template
	redisClient *redis.Client
)

func main() {
	var sConfig StaticConfig
	log.Println("Loading static configs...")
	sConfig.LoadFromEnv()

	redisClient = redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_HOST"), Password: "", DB: 0})
	stat := redisClient.Ping(context.Background())
	if stat.Err() != nil {
		panic(stat.Err())
	}

	dfConfig := api.DefaultConfig()
	dfConfig.Address = os.Getenv("CONSUL_ADDRESS")
	var err error
	ConsulClient, err = api.NewClient(dfConfig)
	if err != nil {
		log.Fatalln(err)
	}
	dConfig := DynamicConfig{
		configServer: ConsulClient,
		configTable:  make(map[string]string),
	}

	refreshRate, _ := strconv.Atoi(sConfig.ConfigRefreshRate)
	dConfig.WaitAndUpdateConfig("backend/broker/channels", refreshRate)

	go dConfig.RefreshConfigs(refreshRate)

	tmpl = template.Must(template.ParseFiles("templates/index.html"))

	server := Server{
		Addr: ":8080",
		dc:   &dConfig,
		tmpl: tmpl,
	}

	http.HandleFunc("/", server.indexHandler)
	http.HandleFunc("/config", server.configHandler)
	http.HandleFunc("/stream", server.streamHandler)

	webServer := http.Server{
		Addr: server.Addr,
	}

	go func() {
		log.Fatalln(webServer.ListenAndServe())
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	<-ch
	log.Printf("\n[+] Got shutdown signal... \n[+] Press CTRL+C again for force shutdown")
	signal.Reset(os.Interrupt)

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*60))
	if downErr := webServer.Shutdown(ctx); downErr != nil {
		fmt.Println("failed to shutdown. Closing...", downErr)
	}
}
