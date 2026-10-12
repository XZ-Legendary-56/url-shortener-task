// Package tests содержит функциональные тесты: сервис проверяется
// снаружи, только через HTTP, как его видит клиент.
//
// По умолчанию тесты сами поднимают сервис в процессе (настоящий роутер
// и in-memory хранилище), поэтому достаточно `go test ./tests/...`.
// Чтобы прогнать их против уже запущенного сервиса (например, из
// docker-compose с Postgres), задайте адрес:
//
//	URL_SHORTENER_BASE_URL=http://localhost:8082 go test ./tests/...
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/router"
	"url-shortener/internal/storage/memory"
)

const baseURLEnv = "URL_SHORTENER_BASE_URL"

var generatedAlias = regexp.MustCompile(`^[A-Za-z0-9_]{10}$`)

type saveResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	Alias  string `json:"alias,omitempty"`
}

// client ходит в сервис и не следует редиректам, чтобы можно было
// проверить сам ответ 302 и заголовок Location.
type client struct {
	baseURL string
	http    *http.Client
}

func newClient(t *testing.T) *client {
	t.Helper()

	baseURL := os.Getenv(baseURLEnv)
	if baseURL == "" {
		log := slog.New(slog.NewTextHandler(io.Discard, nil))
		srv := httptest.NewServer(router.New(log, memory.New()))
		t.Cleanup(srv.Close)
		baseURL = srv.URL
	}

	return &client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// postRaw и save используют assert, а не require: их вызывают и из
// горутин, где require (t.FailNow) использовать нельзя.
func (c *client) postRaw(t testing.TB, body string) (int, saveResponse) {
	t.Helper()

	var out saveResponse

	resp, err := c.http.Post(c.baseURL+"/url", "application/json", bytes.NewBufferString(body))
	if !assert.NoError(t, err) {
		return 0, out
	}
	defer resp.Body.Close()

	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&out))

	return resp.StatusCode, out
}

func (c *client) save(t testing.TB, url, alias string) (int, saveResponse) {
	t.Helper()

	body, err := json.Marshal(map[string]string{"url": url, "alias": alias})
	if !assert.NoError(t, err) {
		return 0, saveResponse{}
	}

	return c.postRaw(t, string(body))
}

func (c *client) get(t testing.TB, alias string) *http.Response {
	t.Helper()

	resp, err := c.http.Get(c.baseURL + "/" + alias)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	return resp
}

// uniq делает данные уникальными для каждого прогона, чтобы тесты можно
// было многократно запускать против сервиса с постоянным хранилищем.
var seq atomic.Int64

func uniq(prefix string) string {
	return fmt.Sprintf("%s%d_%d", prefix, time.Now().UnixNano(), seq.Add(1))
}

func uniqURL() string {
	return "https://example.com/" + uniq("page")
}

func TestSaveAndRedirect_CustomAlias(t *testing.T) {
	c := newClient(t)
	url, alias := uniqURL(), uniq("a")

	code, resp := c.save(t, url, alias)
	require.Equal(t, http.StatusCreated, code)
	assert.Equal(t, "OK", resp.Status)
	assert.Empty(t, resp.Error)
	assert.Equal(t, alias, resp.Alias)

	r := c.get(t, alias)
	assert.Equal(t, http.StatusFound, r.StatusCode)
	assert.Equal(t, url, r.Header.Get("Location"))
}

func TestSaveAndRedirect_GeneratedAlias(t *testing.T) {
	c := newClient(t)
	url := uniqURL()

	code, resp := c.save(t, url, "")
	require.Equal(t, http.StatusCreated, code)
	assert.Equal(t, "OK", resp.Status)
	assert.Regexp(t, generatedAlias, resp.Alias)

	r := c.get(t, resp.Alias)
	assert.Equal(t, http.StatusFound, r.StatusCode)
	assert.Equal(t, url, r.Header.Get("Location"))
}

func TestSave_Conflict(t *testing.T) {
	c := newClient(t)
	url, alias := uniqURL(), uniq("a")

	code, _ := c.save(t, url, alias)
	require.Equal(t, http.StatusCreated, code)

	t.Run("same alias", func(t *testing.T) {
		code, resp := c.save(t, uniqURL(), alias)
		assert.Equal(t, http.StatusConflict, code)
		assert.Equal(t, "Error", resp.Status)
		assert.Equal(t, "url already exists", resp.Error)
	})

	t.Run("same url", func(t *testing.T) {
		code, resp := c.save(t, url, uniq("a"))
		assert.Equal(t, http.StatusConflict, code)
		assert.Equal(t, "url already exists", resp.Error)
	})

	// Первая ссылка не должна была пострадать от неудачных попыток.
	r := c.get(t, alias)
	assert.Equal(t, http.StatusFound, r.StatusCode)
	assert.Equal(t, url, r.Header.Get("Location"))
}

func TestSave_BadRequest(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "malformed json",
			body:    `{"url": `,
			wantErr: "failed to decode request",
		},
		{
			name:    "empty body",
			body:    ``,
			wantErr: "failed to decode request",
		},
		{
			name:    "missing url",
			body:    `{"alias": "whatever"}`,
			wantErr: "field URL is a required field",
		},
		{
			name:    "invalid url",
			body:    `{"url": "not a url"}`,
			wantErr: "field URL is not a valid URL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(t)

			code, resp := c.postRaw(t, tc.body)
			assert.Equal(t, http.StatusBadRequest, code)
			assert.Equal(t, "Error", resp.Status)
			assert.Equal(t, tc.wantErr, resp.Error)
		})
	}
}

func TestRedirect_NotFound(t *testing.T) {
	c := newClient(t)

	r := c.get(t, uniq("missing"))
	assert.Equal(t, http.StatusNotFound, r.StatusCode)
	assert.Empty(t, r.Header.Get("Location"))

	var resp saveResponse
	require.NoError(t, json.NewDecoder(r.Body).Decode(&resp))
	assert.Equal(t, "Error", resp.Status)
	assert.Equal(t, "not found", resp.Error)
}

// Много одновременных запросов без alias: все должны сохраниться и
// получить разные алиасы. С `go test -race` это заодно ловит гонки
// в генераторе алиасов и обработчиках.
func TestSave_ConcurrentGeneratedAliases(t *testing.T) {
	c := newClient(t)

	const n = 50

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		aliases = make(map[string]string, n)
	)

	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()

			url := uniqURL()
			code, resp := c.save(t, url, "")
			if !assert.Equal(t, http.StatusCreated, code, resp.Error) {
				return
			}

			mu.Lock()
			aliases[resp.Alias] = url
			mu.Unlock()
		}()
	}
	wg.Wait()

	require.Len(t, aliases, n, "aliases must be unique")

	for alias, url := range aliases {
		r := c.get(t, alias)
		assert.Equal(t, http.StatusFound, r.StatusCode)
		assert.Equal(t, url, r.Header.Get("Location"))
	}
}
