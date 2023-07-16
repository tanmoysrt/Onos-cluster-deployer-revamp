package main

import (
	"context"

	"github.com/docker/docker/client"
)

type Manager struct {
	ctx    context.Context
	client *client.Client
}

// Volume Mount
type VolumeMount struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readonly"`
	Type	 string `json:"type"`
}

// Ports
type Port struct {
	Protocol      string `json:"protocol"`
	TargetPort    uint64 `json:"target_port"`
	PublishedPort uint64 `json:"published_port"`
	PublishedMode string `json:"published_mode"`
}


type SwarmNode struct {
	ID string `json:"id"`
	IP string `json:"ip"`
	Name string `json:"name"`
	Status bool `json:"status"`
}