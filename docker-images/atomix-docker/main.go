package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var configHash string
var isUpdated bool = false
var wg sync.WaitGroup = sync.WaitGroup{}
var CLUSTER_MANAGER_ADDRESS string = os.Getenv("CLUSTER_MANAGER_ADDRESS")

func updateConfig(config string) bool {
	file_path := "/opt/atomix/conf/atomix.conf"
	f, err := os.Create(file_path)
	if err != nil {
		log.Print(err)
		return false
	}
	defer f.Close()
	_, err = f.WriteString(config)
	if err != nil {
		log.Print(err)
		return false
	}
	isUpdated = (strings.Compare(configHash, config) != 0)
	configHash = config
	return true
}

func fetchAndUpdateConfiguration() bool {
	// Current device fqdn
	hostname, err := os.Hostname()
	if err != nil {
		log.Print("Failed to get hostname")
		return false
	}
	// Send http request
	req, err := http.NewRequest("GET", CLUSTER_MANAGER_ADDRESS+"/atomix/config", nil)
	req.Header.Set("hostname", hostname)
	if err != nil {
		log.Print("Failed to create request")
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle error
		log.Print("Failed to fetch configuration, Retrying...")
		return false
	}
	if resp.StatusCode != 200 {
		return false
	}
	defer resp.Body.Close()
	// read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	// update config
	return updateConfig(string(body))
}

func fetchRunningStatus() bool {
	// Test 5678 port if open
	conn, err := net.Dial("tcp", "localhost:5678")
	if err != nil {
		log.Print("Failed to connect to atomix-agent")
		return false
	}
	conn.Close()

	// Test 5679 port if open
	conn, err = net.Dial("tcp", "localhost:5679")
	if err != nil {
		log.Print("Failed to connect to atomix-agent")
		return false
	}
	conn.Close()
	return true
}

func autoUpdateRunningStatus() {
	for {
		status := fetchRunningStatus()
		statusText := "DOWN"
		if status {
			statusText = "UP"
		}
		log.Print("Atomix Status: ", statusText)
		// Current device fqdn
		hostname, err := os.Hostname()
		if err != nil {
			log.Print("Failed to get hostname")
			continue
		}
		// Send http request
		url := ""
		if status {
			url = CLUSTER_MANAGER_ADDRESS + "/atomix/up"
		} else {
			url = CLUSTER_MANAGER_ADDRESS + "/atomix/down"
		}
		req, err := http.NewRequest("GET", url, nil)
		req.Header.Set("hostname", hostname)
		if err != nil {
			log.Print("Failed to create request")
		} else {
			_, err = http.DefaultClient.Do(req)
			if err != nil {
				// handle error
				log.Print("Failed to update status , Retrying...")
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func listenForExitSignalProcess() {
	sigCh := make(chan os.Signal, 1)
	// Notify the channel for specific signals
	signal.Notify(sigCh, syscall.SIGTERM)
	// Wait for the SIGTERM signal
	<-sigCh
	fmt.Println("Received SIGTERM. Exiting gracefully.")
	wg.Done()
}

func main() {
	log.Print("Starting process ...")
	// Start process
	for {
		log.Print("Waiting for cluster manager")
		if fetchAndUpdateConfiguration() {
			log.Print("Configuration fetched successfully")
			go exec.Command("/opt/atomix/bin/atomix-agent", "-c", "/opt/atomix/conf/atomix.conf").CombinedOutput()
			log.Print("Atomix agent started")
			break
		}
		time.Sleep(10 * time.Second)
	}
	wg.Add(1)
	// Start status updater
	go autoUpdateRunningStatus()
	// Listen for exit signal
	go listenForExitSignalProcess()
	wg.Wait()
	fmt.Println("Exiting process.")
}
