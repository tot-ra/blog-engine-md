package embeddings

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

const CacheVersion = 1

// Cache is the git-friendly sidecar representation of article embeddings.
type Cache struct {
	Version int              `json:"version"`
	Model   string           `json:"model"`
	Dims    int              `json:"dims"`
	Entries map[string]Entry `json:"entries"`
}

// Entry stores one normalized vector quantized with a per-vector scale.
type Entry struct {
	Hash  string  `json:"hash"`
	Vec   string  `json:"vec"`
	Scale float32 `json:"scale"`
	Lang  string  `json:"lang"`
	URL   string  `json:"url"`
}

func NewCache(model string, dims int) *Cache {
	return &Cache{Version: CacheVersion, Model: model, Dims: dims, Entries: map[string]Entry{}}
}

func Load(path, model string, dims int) (*Cache, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return NewCache(model, dims), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read embeddings cache: %w", err)
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parse embeddings cache: %w", err)
	}
	if cache.Entries == nil {
		cache.Entries = map[string]Entry{}
	}
	return &cache, nil
}

// Save relies on encoding/json's lexicographic map-key ordering and adds a final
// newline so repeated writes are byte-for-byte stable and produce small diffs.
func (c *Cache) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal embeddings cache: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return fmt.Errorf("create embeddings cache directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write embeddings cache: %w", err)
	}
	return nil
}

func Normalize(vec []float32) []float32 {
	out := append([]float32(nil), vec...)
	var sum float64
	for _, value := range out {
		sum += float64(value) * float64(value)
	}
	if sum == 0 {
		return out
	}
	inv := float32(1 / math.Sqrt(sum))
	for i := range out {
		out[i] *= inv
	}
	return out
}

func Quantize(vec []float32) (encoded string, scale float32, err error) {
	normalized := Normalize(vec)
	var maxAbs float32
	for _, value := range normalized {
		abs := float32(math.Abs(float64(value)))
		if abs > maxAbs {
			maxAbs = abs
		}
	}
	if maxAbs == 0 {
		return base64.StdEncoding.EncodeToString(make([]byte, len(normalized))), 1, nil
	}
	scale = maxAbs / 127
	quantized := make([]byte, len(normalized))
	for i, value := range normalized {
		q := int(math.Round(float64(value / scale)))
		if q > 127 {
			q = 127
		} else if q < -127 {
			q = -127
		}
		quantized[i] = byte(int8(q))
	}
	return base64.StdEncoding.EncodeToString(quantized), scale, nil
}

func Dequantize(encoded string, scale float32) ([]float32, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode quantized vector: %w", err)
	}
	vec := make([]float32, len(data))
	for i, value := range data {
		vec[i] = float32(int8(value)) * scale
	}
	return vec, nil
}
