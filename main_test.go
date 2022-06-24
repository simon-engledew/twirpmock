package main

import (
	"github.com/simon-engledew/twirpmock/pkg/handler"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBasic(t *testing.T) {
	mux := handler.NewServeMux()

	var set descriptorpb.FileDescriptorSet

	require.NoError(t, unmarshalSet("example/service.proto.pb", &set))
	require.NoError(t, mux.Handle(&set, "example/service.star", nil))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "http://localhost:8888/twirp/twirpmock.example.Example/Echo", strings.NewReader(`{"name": "Test"}`))
	r.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Hello Test!")
}
