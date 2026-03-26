package models

import "time"

// Asset represents a discovered device on the network
type Asset struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	MAC       string    `json:"mac"`
	Hostname  string    `json:"hostname"`
	OS        string    `json:"os"`
	Ports     []int     `json:"ports"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}
