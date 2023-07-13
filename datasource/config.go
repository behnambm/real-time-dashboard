package main

import (
	"github.com/hashicorp/consul/api"
	"log"
	"os"
	"sync"
	"time"
)

type StaticConfig struct {
	ConsulAddress     string
	ConfigRefreshRate string
}

func (sc *StaticConfig) LoadFromEnv() {
	consulAddress, isPresent := os.LookupEnv("CONSUL_ADDRESS")
	if !isPresent {
		panic("CONSUL ADDRESS is not defined")
	}

	// how many seconds to wait before sending request to the config server to fetch configs
	configRefreshRate, isPresent := os.LookupEnv("DYNAMIC_CONFIG_REFRESH_RATE")
	if !isPresent {
		panic("DYNAMIC_CONFIG_REFRESH_RATE is not defined")
	}

	sc.ConsulAddress = consulAddress
	sc.ConfigRefreshRate = configRefreshRate
}

// DynamicConfig is used to store and manage configs from Consul server
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

func (dc *DynamicConfig) WaitAndUpdateConfig(key string, wait int) {
	for {
		value := dc.FetchConfig(key)
		if value != "" {
			dc.SetConfig(key, value)
			break
		}
		time.Sleep(time.Millisecond * time.Duration(wait))
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
