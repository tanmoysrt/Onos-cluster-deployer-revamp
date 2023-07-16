package main

import (
	"time"

	"github.com/google/uuid"
)

var sessionTokenMap map[string]int = make(map[string]int, 100)

func generateSessionToken() string {
	sessionToken := uuid.New().String()
	sessionTokenMap[sessionToken] = int(time.Now().Unix())+3600
	return sessionToken
}

func deleteSessionToken(sessionToken string) {
	delete(sessionTokenMap, sessionToken)
}

func isSessionTokenValid(sessionToken string) bool {
	if sessionTokenMap[sessionToken] == 0 {
		return false
	}
	if sessionTokenMap[sessionToken] - int(time.Now().Unix()) < 0 {
		return false
	}
	return true
}