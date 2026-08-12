package valhalla

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/doveva/Gulyaem/backend/internal/routing/port"
)

const maxMetadataBytes = 64 << 10

type FileMetadataSource struct {
	path  string
	mutex sync.Mutex
	cache metadataCache
}

type metadataCache struct {
	key      string
	metadata port.Metadata
}

func NewFileMetadataSource(path string) *FileMetadataSource {
	return &FileMetadataSource{path: filepath.Clean(path)}
}

func (source *FileMetadataSource) Load(ctx context.Context) (port.Metadata, error) {
	if err := ctx.Err(); err != nil {
		return port.Metadata{}, err
	}
	contents, err := os.ReadFile(source.path)
	if err != nil {
		return port.Metadata{}, fmt.Errorf("%w: read graph metadata: %v", port.ErrInvalidResponse, err)
	}
	if len(contents) == 0 || len(contents) > maxMetadataBytes {
		return port.Metadata{}, fmt.Errorf("%w: graph metadata size is invalid", port.ErrInvalidResponse)
	}
	metadata, err := decodeMetadata(contents)
	if err != nil {
		return port.Metadata{}, err
	}
	if filepath.IsAbs(metadata.GraphArtifact) || filepath.Base(metadata.GraphArtifact) != metadata.GraphArtifact ||
		metadata.GraphArtifact == "." {
		return port.Metadata{}, fmt.Errorf("%w: graph artifact must be a file beside metadata", port.ErrInvalidResponse)
	}
	if !validSHA256(metadata.SourceChecksum) || !validSHA256(metadata.GraphChecksum) {
		return port.Metadata{}, fmt.Errorf("%w: metadata checksums must be SHA-256", port.ErrInvalidResponse)
	}
	graphPath := filepath.Join(filepath.Dir(source.path), metadata.GraphArtifact)
	graphInfo, err := os.Stat(graphPath)
	if err != nil || !graphInfo.Mode().IsRegular() || graphInfo.Size() == 0 {
		return port.Metadata{}, fmt.Errorf("%w: graph artifact is unavailable", port.ErrInvalidResponse)
	}
	metadataDigest := sha256.Sum256(contents)
	cacheKey := fmt.Sprintf("%x:%d:%d", metadataDigest, graphInfo.Size(), graphInfo.ModTime().UnixNano())

	source.mutex.Lock()
	defer source.mutex.Unlock()
	if source.cache.key == cacheKey {
		return source.cache.metadata, nil
	}
	graphChecksum, err := checksumFile(ctx, graphPath)
	if err != nil {
		return port.Metadata{}, err
	}
	if !strings.EqualFold(graphChecksum, metadata.GraphChecksum) {
		return port.Metadata{}, fmt.Errorf("%w: graph artifact checksum mismatch", port.ErrInvalidResponse)
	}
	source.cache = metadataCache{key: cacheKey, metadata: metadata}
	return metadata, nil
}

func decodeMetadata(contents []byte) (port.Metadata, error) {
	var metadata port.Metadata
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return port.Metadata{}, fmt.Errorf("%w: decode graph metadata", port.ErrInvalidResponse)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return port.Metadata{}, fmt.Errorf("%w: graph metadata must contain one object", port.ErrInvalidResponse)
	}
	return metadata, nil
}

func checksumFile(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("%w: open graph artifact", port.ErrInvalidResponse)
	}
	defer file.Close()
	hash := sha256.New()
	buffer := make([]byte, 1<<20)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		read, readErr := file.Read(buffer)
		if read > 0 {
			_, _ = hash.Write(buffer[:read])
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("%w: hash graph artifact", port.ErrInvalidResponse)
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
