package main

import (
	"log"
	"os"
	"regexp"
	"strings"
)

func isValidIP(ip string) bool {
	// regex
	pattern := `^(\d{1,3}\.){3}\d{1,3}$`
	if matched, _ := regexp.MatchString(pattern, ip); matched {
		return true
	}
	return false
}

func backupAtomixIPFromFile() {
	// Read from file
	filename := "node_ips.txt"
	// Read from file
	file, err := os.Open(filename)
	if err != nil {
		os.Create(filename)
		log.Printf("Created file %s", filename)
		return
	}
	defer file.Close()
	// Read from file
	content := make([]byte, 1024)
	_, err = file.Read(content)
	if err != nil {
		log.Print("Failed to read from file")
		return
	}
	contentString := string(content)
	ips := strings.Split(contentString, "\n")
	// Write to atomixNodesIP
	for _, ip := range ips {
		if isValidIP(ip) {
			addAtomixNode(ip)
		}
	}
}

func dumpAtomixIPInFile() {
	// Write to file
	filename := "node_ips.txt"
	file, err := os.Create(filename)
	if err != nil {
		log.Print("Failed to create file")
		return
	}
	defer file.Close()
	// Write to file
	for _, ip := range atomixNodesIP {
		file.WriteString(ip + "\n")
	}
}

func backupOnosIPFromFile() {
	// Read from file
	filename := "onos_ips.txt"
	// Read from file
	file, err := os.Open(filename)
	if err != nil {
		os.Create(filename)
		log.Printf("Created file %s", filename)
		return
	}
	defer file.Close()
	// Read from file
	content := make([]byte, 1024)
	_, err = file.Read(content)
	if err != nil {
		log.Print("Failed to read from file")
		return
	}
	contentString := string(content)
	ips := strings.Split(contentString, "\n")
	// Write to atomixNodesIP
	for _, ip := range ips {
		if isValidIP(ip) {
			addOnosNode(ip)
		}
	}
}

func dumpOnosIPInFile() {
	// Write to file
	filename := "onos_ips.txt"
	file, err := os.Create(filename)
	if err != nil {
		log.Print("Failed to create file")
		return
	}
	defer file.Close()
	// Write to file
	for _, ip := range onosNodesIP {
		file.WriteString(ip + "\n")
	}
}
