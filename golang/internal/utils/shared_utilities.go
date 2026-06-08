package utils

import (
	"math/rand"
	"time"
)

// helper function to sleep a random given time in ms
func SleepRandMilliseconds(delayMs int) {
	sleepTime := time.Duration(rand.Intn(delayMs)+100) * time.Millisecond
	time.Sleep(sleepTime)
}
