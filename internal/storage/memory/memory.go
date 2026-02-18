package memory

import (
	"fmt"
	"sync"

	"url-shortener/internal/storage"
)

type Storage struct {
	mu      sync.RWMutex
	urls    map[string]string // alias -> url
	reverse map[string]string // url -> alias (для проверки уникальности)
}

func New() *Storage {
	return &Storage{
		urls:    make(map[string]string),
		reverse: make(map[string]string),
	}
}

func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const op = "storage.memory.SaveURL"

	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем, существует ли уже такой alias
	if _, exists := s.urls[alias]; exists {
		return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
	}

	// Проверяем, существует ли уже такой URL
	if _, exists := s.reverse[urlToSave]; exists {
		return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
	}

	// Сохраняем
	s.urls[alias] = urlToSave
	s.reverse[urlToSave] = alias

	// Для in-memory хранилища возвращаем "id" на основе длины мапы
	return int64(len(s.urls)), nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.memory.GetURL"

	s.mu.RLock()
	defer s.mu.RUnlock()

	url, exists := s.urls[alias]
	if !exists {
		return "", storage.ErrURLNotFound
	}

	return url, nil
}

// Close - заглушка для совместимости с интерфейсом
func (s *Storage) Close() error {
	return nil
}
