package random

import "math/rand/v2"

const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"abcdefghijklmnopqrstuvwxyz" +
	"0123456789" + "_"

// NewRandomString возвращает строку длины size из символов [A-Za-z0-9_].
//
// Безопасна для вызова из нескольких горутин: функции верхнего уровня
// math/rand/v2 используют глобальный генератор, который потокобезопасен
// и сидируется автоматически. Раньше здесь был общий *rand.Rand, который
// не потокобезопасен — sync.Once защищал только его создание, а не вызовы.
func NewRandomString(size int) string {
	b := make([]byte, size)
	for i := range b {
		b[i] = chars[rand.IntN(len(chars))]
	}

	return string(b)
}
