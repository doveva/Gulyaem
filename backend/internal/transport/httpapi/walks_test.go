package httpapi

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/walks"
)

func TestWalkCreateRejectsClientControlledActor(t *testing.T) {
	service := walks.NewService(nil, nil)
	handler := NewHandler(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Actor: StaticActorResolver{ID: "01900000-0000-7000-8000-000000000003"}, Walks: service})
	body := `{"actorId":"01900000-0000-7000-8000-000000000999","clientRequestId":"01900000-0000-7000-8000-000000000010","cityId":"01900000-0000-7000-8000-000000000001","profile":"pedestrian","expectedPreviewFingerprint":"opaque","waypoints":[{"lat":59.9,"lon":30.3},{"lat":60,"lon":30.4}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/walks", bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "invalid_body") {
		t.Fatalf("body=%s", response.Body.String())
	}
}
