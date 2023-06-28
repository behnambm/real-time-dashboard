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
	"os"
	"strconv"
	"sync"
	"time"
)

type Coordinate struct {
	X string `json:"x"`
	Y string `json:"y"`
}

type StaticConfig struct {
	ConsulAddress     string
	ConfigRefreshRate string
}

func (sc *StaticConfig) LoadFromEnv() {
	consulAddress, isPresent := os.LookupEnv("CONSUL_ADDRESS")
	if !isPresent {
		panic("CONSUL ADDRESS is not defined")
	}

	configRefreshRate, isPresent := os.LookupEnv("DYNAMIC_CONFIG_REFRESH_RATE")
	if !isPresent {
		panic("DYNAMIC_CONFIG_REFRESH_RATE is not defined")
	}

	sc.ConfigRefreshRate = configRefreshRate
	sc.ConsulAddress = consulAddress
}

type DynamicConfig struct {
	configServer *api.Client
	configTable  map[string]string
	mu           sync.Mutex
}

func (dc *DynamicConfig) FetchConfig(key string) string {
	kv := dc.configServer.KV()
	pair, _, err := kv.Get(key, nil)
	if err != nil {
		log.Println(err)
	}

	if pair == nil {
		return ""
	}

	return string(pair.Value)
}

func (dc *DynamicConfig) WaitAndUpdateConfig(key string) {
	for {
		value := dc.FetchConfig(key)
		if value != "" {
			dc.SetConfig(key, value)
			break
		}
		time.Sleep(time.Millisecond * 3500)
	}
}

func (dc *DynamicConfig) SetConfig(key, value string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.configTable[key] = value
}

func (dc *DynamicConfig) GetKeys() []string {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	var keyList []string
	for k, _ := range dc.configTable {
		keyList = append(keyList, k)
	}
	return keyList
}

func (dc *DynamicConfig) RefreshConfigs(refreshRate int) {
	for {
		for _, key := range dc.GetKeys() {
			dc.SetConfig(key, dc.FetchConfig(key))
		}
		time.Sleep(time.Second * time.Duration(refreshRate))
	}
}

func (dc *DynamicConfig) GetConfig(key string) string {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	value, _ := dc.configTable[key]
	return value
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

	dConfig.WaitAndUpdateConfig("datasource/refresh_rate")
	dConfig.WaitAndUpdateConfig("datasource/broker/host")
	dConfig.WaitAndUpdateConfig("datasource/broker/port")

	refreshRate, _ := strconv.Atoi(sConfig.ConfigRefreshRate)
	go dConfig.RefreshConfigs(refreshRate)

	rdb := redis.NewClient(&redis.Options{
		Addr:     dConfig.FetchConfig("datasource/broker/host") + ":" + dConfig.FetchConfig("datasource/broker/port"),
		Password: "",
		DB:       0,
	})

	sem := semaphore.NewWeighted(3)
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
		sem.Acquire(context.Background(), 3)
		go publish("btc", 64)
		go publish("eth", 26)
		go publish("dog", 8)

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
