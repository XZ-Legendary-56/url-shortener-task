package random

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRandomString(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{
			name: "size = 1",
			size: 1,
		},
		{
			name: "size = 5",
			size: 5,
		},
		{
			name: "size = 10",
			size: 10,
		},
		{
			name: "size = 20",
			size: 20,
		},
		{
			name: "size = 30",
			size: 30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str1 := NewRandomString(tt.size)
			str2 := NewRandomString(tt.size)

			assert.Len(t, str1, tt.size)
			assert.Len(t, str2, tt.size)

			assert.NotEqual(t, str1, str2)
		})
	}
}

// Проверяется вместе с go test -race: генератор вызывается
// из HTTP-обработчиков, то есть из многих горутин сразу.
func TestNewRandomString_Concurrent(t *testing.T) {
	const goroutines, perGoroutine = 50, 100

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				if s := NewRandomString(10); len(s) != 10 {
					t.Errorf("unexpected length %d", len(s))
				}
			}
		}()
	}
	wg.Wait()
}
