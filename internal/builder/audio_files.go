package builder

import (
	"os"
	"path/filepath"
	"strings"
)

func (b *SiteBuilder) appendAssetFiles(index *ContentIndex, absPaths []string) {
	if index == nil || len(absPaths) == 0 {
		return
	}

	existing := make(map[string]struct{}, len(index.AssetFiles))
	for _, f := range index.AssetFiles {
		existing[filepath.Clean(f.Path)] = struct{}{}
	}

	contentAbs := b.resolvePath(b.config.Build.ContentDir)
	for _, p := range absPaths {
		abs := filepath.Clean(p)
		if _, ok := existing[abs]; ok {
			continue
		}

		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			continue
		}
		rel, err := filepath.Rel(contentAbs, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}

		index.AssetFiles = append(index.AssetFiles, ContentFile{
			Path:         abs,
			RelativePath: rel,
			ContentType:  TypeAsset,
			ModifiedTime: info.ModTime().Unix(),
			Size:         info.Size(),
		})
		existing[abs] = struct{}{}
	}
}

func fileExistsNonEmpty(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
