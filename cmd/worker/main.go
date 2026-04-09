package main

import (
	"log"
	"time"
)

// Full async jobs (AI suggestions) are planned.
func main() {
	log.Println("worker scaffold running; no jobs configured yet")
	for {
		time.Sleep(30 * time.Second)
	}
}
