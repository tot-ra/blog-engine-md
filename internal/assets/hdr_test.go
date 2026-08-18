package assets

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func addJPEGSegment(t *testing.T, path string, marker byte, payload []byte) {
	t.Helper()
	jpegData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(jpegData) < 2 || !bytes.Equal(jpegData[:2], []byte{0xff, 0xd8}) {
		t.Fatalf("test input is not a JPEG: %s", path)
	}
	if len(payload)+2 > 0xffff {
		t.Fatal("test JPEG segment is too large")
	}

	segment := []byte{0xff, marker, 0, 0}
	binary.BigEndian.PutUint16(segment[2:], uint16(len(payload)+2))
	segment = append(segment, payload...)
	jpegData = append(append(jpegData[:2:2], segment...), jpegData[2:]...)
	if err := os.WriteFile(path, jpegData, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestHasHDRGainMap(t *testing.T) {
	tests := []struct {
		name    string
		marker  byte
		payload []byte
		want    bool
	}{
		{
			name:    "ISO 21496-1 APP2 metadata",
			marker:  0xe2,
			payload: append(append([]byte{}, isoGainMapNamespace...), 0, 0, 0, 0),
			want:    true,
		},
		{
			name:   "Adobe gain map XMP",
			marker: 0xe1,
			payload: []byte("http://ns.adobe.com/xap/1.0/\x00" +
				`<rdf:Description xmlns:hdrgm="http://ns.adobe.com/hdr-gain-map/1.0/" hdrgm:Version="1.0"/>`),
			want: true,
		},
		{
			name:    "Apple gain map XMP with EXIF headroom",
			marker:  0xe1,
			payload: []byte(`<rdf:Description HDRGainMapVersion="1"/>`),
			want:    true,
		},
		{
			name:    "unrelated HDR comment",
			marker:  0xfe,
			payload: []byte("HDR photo with GainMap words in a JPEG comment"),
			want:    false,
		},
		{
			name:    "gain map namespace without version",
			marker:  0xe1,
			payload: adobeGainMapNamespace,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "photo.jpg")
			createTestJPEG(t, path, 16, 12)
			addJPEGSegment(t, path, tt.marker, tt.payload)

			got, err := hasHDRGainMap(path)
			if err != nil {
				t.Fatalf("hasHDRGainMap failed: %v", err)
			}
			if got != tt.want {
				t.Fatalf("hasHDRGainMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImageProcessor_PreservesHDRJPEG(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src", "hdr-photo.jpg")
	outDir := filepath.Join(tmpDir, "dist")
	createTestJPEG(t, srcPath, 80, 60)
	addJPEGSegment(t, srcPath, 0xe2, append(append([]byte{}, isoGainMapNamespace...), 0, 0, 0, 0))

	processor := NewImageProcessor(ImageConfig{
		Quality: 85,
		Sizes:   map[string]int{"thumbnail": 40, "full": 80},
		Enabled: true,
	}, outDir, nil)
	result, err := processor.ProcessFile(srcPath, "photos/hdr-photo.jpg", 0, 0)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}
	if len(result.Variants) != 1 || result.Variants[0].Size != "original" {
		t.Fatalf("expected only the preserved original, got %#v", result.Variants)
	}
	if got := result.Variants[0].FilePath; got != "/assets/img/photos/hdr-photo.jpg" {
		t.Fatalf("unexpected preserved path: %q", got)
	}

	source, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(filepath.Join(outDir, "assets", "img", "photos", "hdr-photo.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output, source) {
		t.Fatal("preserved HDR JPEG differs from its source bytes")
	}
}

func TestImageProcessor_HDRDetectionPrecedesVariantCache(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src", "hdr-photo.jpg")
	outDir := filepath.Join(tmpDir, "dist")
	cache := NewImageCache(filepath.Join(tmpDir, ".cache"))
	createTestJPEG(t, srcPath, 80, 60)
	addJPEGSegment(t, srcPath, 0xe2, append(append([]byte{}, isoGainMapNamespace...), 0, 0, 0, 0))

	const modTime int64 = 123
	info, err := os.Stat(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	stalePath := filepath.Join(outDir, "assets", "img", "hdr-photo-full.webp")
	if err := os.MkdirAll(filepath.Dir(stalePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stalePath, []byte("stale webp"), 0644); err != nil {
		t.Fatal(err)
	}
	staleVariant := ImageVariant{Size: "full", FilePath: "/assets/img/hdr-photo-full.webp"}
	if err := cache.StoreVariantFile(staleVariant.FilePath, outDir); err != nil {
		t.Fatal(err)
	}
	cache.Set("hdr-photo.jpg", &CacheEntry{
		SourceModTime: modTime,
		SourceSize:    info.Size(),
		Variants:      []ImageVariant{staleVariant},
	})

	processor := NewImageProcessor(DefaultImageConfig(), outDir, cache)
	result, err := processor.ProcessFile(srcPath, "hdr-photo.jpg", modTime, info.Size())
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}
	if len(result.Variants) != 1 || result.Variants[0].FilePath != "/assets/img/hdr-photo.jpg" {
		t.Fatalf("expected preserved JPEG instead of stale WebP cache, got %#v", result.Variants)
	}
}
