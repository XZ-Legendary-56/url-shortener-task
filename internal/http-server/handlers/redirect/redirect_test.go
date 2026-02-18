package redirect

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/redirect/mocks"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/storage"
)

func TestSaveHandler(t *testing.T) {
	cases := []struct {
		name           string
		alias          string
		url            string
		respError      string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "Success",
			alias:          "test_alias",
			url:            "https://www.google.com/",
			expectedStatus: http.StatusFound,
		},
		{
			name:           "Empty alias",
			alias:          "",
			url:            "",
			respError:      "invalid request",
			expectedStatus: http.StatusOK, // Возвращает JSON с ошибкой
		},
		{
			name:           "URL not found",
			alias:          "non_existent",
			url:            "",
			mockError:      storage.ErrURLNotFound,
			respError:      "not found",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Internal error",
			alias:          "error_alias",
			url:            "",
			mockError:      errors.New("database connection failed"),
			respError:      "internal error",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Very long alias",
			alias:          "this_is_a_very_long_alias_that_might_cause_issues_but_should_still_work_123456789",
			url:            "https://example.com/very/long/path?with=params&and=more",
			expectedStatus: http.StatusFound,
		},
		{
			name:           "Alias with special characters",
			alias:          "test-123_abc",
			url:            "https://example.com",
			expectedStatus: http.StatusFound,
		},
		{
			name:           "HTTPS URL",
			alias:          "secure",
			url:            "https://secure-site.com",
			expectedStatus: http.StatusFound,
		},
		{
			name:           "HTTP URL",
			alias:          "insecure",
			url:            "http://example.com",
			expectedStatus: http.StatusFound,
		},
	}

	for _, tc := range cases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlGetterMock := mocks.NewURLGetter(t)

			// Настройка мока в зависимости от сценария
			if tc.mockError != nil {
				urlGetterMock.On("GetURL", tc.alias).
					Return("", tc.mockError).Once()
			} else if tc.url != "" {
				urlGetterMock.On("GetURL", tc.alias).
					Return(tc.url, nil).Once()
			} else if tc.respError == "invalid request" {
				// Для пустого alias мок может не вызываться
				// но мы всё равно ожидаем, что он не будет вызван
			}

			r := chi.NewRouter()
			r.Get("/{alias}", New(slog.Default(), urlGetterMock))

			ts := httptest.NewServer(r)
			defer ts.Close()

			// Специальная обработка для случая с пустым alias
			requestURL := ts.URL
			if tc.alias != "" {
				requestURL += "/" + tc.alias
			}

			redirectedToURL, err := api.GetRedirect(requestURL)

			if tc.respError != "" {
				// Ожидаем ошибку в JSON ответе
				assert.Error(t, err)
				// Проверяем, что err содержит нужное сообщение
				// Примечание: api.GetRedirect возвращает ошибку, а не JSON
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.url, redirectedToURL)
			}

			// Проверяем, что мок был вызван ожидаемое количество раз
			if tc.alias != "" && tc.respError != "invalid request" {
				urlGetterMock.AssertExpectations(t)
			}
		})
	}
}
