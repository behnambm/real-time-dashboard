package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/semaphore"
	"log"
	"math/rand"
	"runtime"
	"strconv"
	"time"
)

type Coordinate struct {
	X string `json:"x"`
	Y string `json:"y"`
}

type Source struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func main() {
	var sConfig StaticConfig
	log.Println("Loading static configs...")
	sConfig.LoadFromEnv()

	log.Println("Connecting to config server...")
	configServer, err := api.NewClient(&api.Config{Address: sConfig.ConsulAddress})
	if err != nil {
		log.Fatalln("config server - ", err)
	}

	log.Println("Fetching dynamic configs...")
	dConfig := DynamicConfig{
		configServer: configServer,
		configTable:  make(map[string]string),
	}

	refreshRate, _ := strconv.Atoi(sConfig.ConfigRefreshRate)
	dConfig.WaitAndUpdateConfig("datasource/refresh_rate", refreshRate)
	dConfig.WaitAndUpdateConfig("datasource/broker/host", refreshRate)
	dConfig.WaitAndUpdateConfig("datasource/broker/port", refreshRate)
	dConfig.WaitAndUpdateConfig("datasource/broker/channels", refreshRate)

	go dConfig.RefreshConfigs(refreshRate)

	rdb := redis.NewClient(&redis.Options{
		Addr:     dConfig.FetchConfig("datasource/broker/host") + ":" + dConfig.FetchConfig("datasource/broker/port"),
		Password: "",
		DB:       0,
	})

	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	publish := func(name string, n int) {
		defer sem.Release(1)
		payload, err := json.Marshal(GenerateGraphData(n))
		if err != nil {
			log.Println(err)
			return
		}
		rdb.Publish(context.Background(), name, payload)
	}

	for {
		channels := dConfig.GetConfig("datasource/broker/channels")
		var channelsList []Source
		err := json.Unmarshal([]byte(channels), &channelsList)
		if err != nil {
			log.Println("INVALID channels VALUE")
		} else {
			for _, channelSource := range channelsList {
				sem.Acquire(context.Background(), 1)
				go publish(channelSource.Name, channelSource.Count)
			}
		}

		refreshRate, _ := strconv.Atoi(dConfig.GetConfig("datasource/refresh_rate"))
		time.Sleep(time.Millisecond * time.Duration(refreshRate))
	}
}

func GenerateGraphData(n int) []Coordinate {
	var d []Coordinate
	for i := 0; i < n; i++ {
		c := Coordinate{X: fmt.Sprintf("%f", rand.Float32()*float32(rand.Intn(50))), Y: fmt.Sprintf("%f", rand.Float32()*float32(rand.Intn(70)))}
		d = append(d, c)
	}
	return d
}
