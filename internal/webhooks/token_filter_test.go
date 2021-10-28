package webhooks

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brigadecore/brigade-foundations/crypto"
	"github.com/stretchr/testify/require"
)

func TestNewTokenFilterConfig(t *testing.T) {
	config := NewTokenFilterConfig()
	require.NotNil(t, config.(*tokenFilterConfig).hashedTokens)
}

func TestAddToken(t *testing.T) {
	const testToken = "foo"
	// nolint: forcetypeassert
	config := NewTokenFilterConfig().(*tokenFilterConfig)
	require.Empty(t, config.hashedTokens)
	config.AddToken(testToken)
	require.Len(t, config.hashedTokens, 1)
	require.Equal(t, crypto.Hash("", testToken), config.hashedTokens[0])
}

func TestGetHashedTokens(t *testing.T) {
	testHashedTokens := []string{"foo", "bar"}
	config := tokenFilterConfig{
		hashedTokens: testHashedTokens,
	}
	require.Equal(t, testHashedTokens, config.getHashedTokens())
}

func TestNewTokenFilter(t *testing.T) {
	testConfig := NewTokenFilterConfig()
	filter := NewTokenFilter(testConfig).(*tokenFilter) // nolint: forcetypeassert
	require.Equal(t, testConfig, filter.config)
}

func TestTokenFilter(t *testing.T) {
	testConfig := NewTokenFilterConfig()
	const testToken = "bar"
	testConfig.AddToken(testToken)
	testCases := []struct {
		name       string
		filter     *tokenFilter
		setup      func() *http.Request
		assertions func(handlerCalled bool, rr *httptest.ResponseRecorder)
	}{
		{
			name: "valid token provided",
			filter: &tokenFilter{
				config: testConfig,
			},
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/", nil)
				require.NoError(t, err)
				q := req.URL.Query()
				q.Set("access_token", testToken)
				req.URL.RawQuery = q.Encode()
				return req
			},
			assertions: func(handlerCalled bool, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.True(t, handlerCalled)
			},
		},
		{
			name: "no token provided",
			filter: &tokenFilter{
				config: testConfig,
			},
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/", nil)
				require.NoError(t, err)
				return req
			},
			assertions: func(handlerCalled bool, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, rr.Code)
				require.False(t, handlerCalled)
			},
		},
		{
			name: "invalid token provided",
			filter: &tokenFilter{
				config: testConfig,
			},
			setup: func() *http.Request {
				req, err := http.NewRequest(http.MethodPost, "/", nil)
				require.NoError(t, err)
				q := req.URL.Query()
				q.Set("access_token", "bogus-token")
				req.URL.RawQuery = q.Encode()
				return req
			},
			assertions: func(handlerCalled bool, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, rr.Code)
				require.False(t, handlerCalled)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := testCase.setup()
			handlerCalled := false
			testCase.filter.Decorate(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})(rr, req)
			testCase.assertions(handlerCalled, rr)
		})
	}
}
