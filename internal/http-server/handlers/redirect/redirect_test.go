package redirect

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/redirect/mocks"
	"url-shortener/internal/lib/api/response"
	"url-shortener/internal/storage"
)

func TestRedirectHandler(t *testing.T) {
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
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "URL not found",
			alias:          "non_existent",
			url:            "",
			mockError:      storage.ErrURLNotFound,
			respError:      "not found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Internal error",
			alias:          "error_alias",
			url:            "",
			mockError:      errors.New("database connection failed"),
			respError:      "internal error",
			expectedStatus: http.StatusInternalServerError,
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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlGetterMock := mocks.NewURLGetter(t)
			if tc.alias != "" {
				urlGetterMock.On("GetURL", tc.alias).
					Return(tc.url, tc.mockError).Once()
			}

			handler := New(slog.New(slog.NewTextHandler(io.Discard, nil)), urlGetterMock)

			// Обработчик вызываем через chi, чтобы в контексте был параметр
			// alias. Пустой alias через роутер не передать ("/" не совпадёт
			// с "/{alias}"), поэтому кладём пустой параметр в контекст сами.
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("alias", tc.alias)

			req := httptest.NewRequest(http.MethodGet, "/"+tc.alias, nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.expectedStatus, rr.Code)

			if tc.respError == "" {
				assert.Equal(t, tc.url, rr.Header().Get("Location"))
				return
			}

			var body response.Response
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
			assert.Equal(t, response.StatusError, body.Status)
			assert.Equal(t, tc.respError, body.Error)
		})
	}
}
