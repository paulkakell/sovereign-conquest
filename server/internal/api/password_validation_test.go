package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestPasswordHandlersRejectInvalidLengthsBeforeDatabaseAccess(t *testing.T) {
	server := &Server{} // No pool: invalid input must never reach database work.
	for _, password := range []string{"", "short", strings.Repeat("a", 73), strings.Repeat("a", 100), strings.Repeat("é", 37)} {
		for _, route := range []string{"register", "change_password"} {
			t.Run(route+"/"+strconv.Itoa(len(password)), func(t *testing.T) {
				payload, err := json.Marshal(map[string]string{"username": "fixture", "password": password, "old_password": "old-fixture-password", "new_password": password})
				if err != nil {
					t.Fatal(err)
				}
				request := httptest.NewRequest(http.MethodPost, "/api/"+route, bytes.NewReader(payload))
				request = request.WithContext(context.WithValue(request.Context(), ctxUserID, "fixture-user"))
				response := httptest.NewRecorder()
				if route == "register" {
					server.handleRegister(response, request)
				} else {
					server.handleChangePassword(response, request)
				}
				if response.Code != http.StatusBadRequest {
					t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
				}
				if !strings.Contains(response.Body.String(), "8-72 bytes") {
					t.Fatalf("missing descriptive byte limit: %s", response.Body.String())
				}
				if password != "" && strings.Contains(response.Body.String(), password) {
					t.Fatal("response echoed a password")
				}
			})
		}
	}
}
