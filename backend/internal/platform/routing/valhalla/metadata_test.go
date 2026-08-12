package valhalla

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/routing/port"
)

func TestFileMetadataSourceBindsMetadataToGraphArtifact(t *testing.T) {
	directory := t.TempDir()
	graph := []byte("real valhalla graph artifact")
	graphChecksum := sha256.Sum256(graph)
	metadata := validMetadata()
	metadata.GraphChecksum = hex.EncodeToString(graphChecksum[:])
	writeMetadataFixture(t, directory, graph, metadata)

	loaded, err := NewFileMetadataSource(filepath.Join(directory, "routing-dataset.json")).Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.SourceChecksum != metadata.SourceChecksum || loaded.GraphChecksum != metadata.GraphChecksum {
		t.Fatalf("metadata = %#v", loaded)
	}
}

func TestFileMetadataSourceRejectsGraphChecksumMismatch(t *testing.T) {
	directory := t.TempDir()
	metadata := validMetadata()
	writeMetadataFixture(t, directory, []byte("different graph"), metadata)

	_, err := NewFileMetadataSource(filepath.Join(directory, "routing-dataset.json")).Load(context.Background())
	if !errors.Is(err, port.ErrInvalidResponse) || !strings.Contains(err.Error(), "graph artifact checksum mismatch") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestFileMetadataSourceRejectsMissingAndEscapingArtifacts(t *testing.T) {
	directory := t.TempDir()
	metadata := validMetadata()
	metadata.GraphArtifact = "../valhalla_tiles.tar"
	encoded, _ := json.Marshal(metadata)
	if err := os.WriteFile(filepath.Join(directory, "routing-dataset.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewFileMetadataSource(filepath.Join(directory, "routing-dataset.json")).Load(context.Background())
	if !errors.Is(err, port.ErrInvalidResponse) {
		t.Fatalf("Load() error = %v", err)
	}
}

func writeMetadataFixture(t *testing.T, directory string, graph []byte, metadata port.Metadata) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, metadata.GraphArtifact), graph, 0o600); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "routing-dataset.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
}
