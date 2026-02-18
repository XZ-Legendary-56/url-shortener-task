package random

import (
	"math/rand"
	"sync"
	"time"
)

var (
	rng  *rand.Rand
	once sync.Once
)

func NewRandomString(size int) string {
	once.Do(func() {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	})

	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789" + "_")

	b := make([]rune, size)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}

	return string(b)
}
