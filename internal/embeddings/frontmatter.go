package embeddings

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tot-ra/blog-engine/internal/parser"
)

// WriteFrontmatterEmbedding keeps the article itself as the portable source of
// truth. Path, URL, and language are deliberately derived during each build.
func WriteFrontmatterEmbedding(path string, embedding parser.FrontmatterEmbedding) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read article frontmatter: %w", err)
	}
	content := string(data)
	block := formatEmbeddingBlock(embedding)

	var updated string
	if strings.HasPrefix(content, "---") {
		end := strings.Index(content[3:], "---")
		if end < 0 {
			return fmt.Errorf("article has unterminated frontmatter")
		}
		end += 3
		frontmatter := content[3:end]
		frontmatter = removeEmbeddingBlock(frontmatter)
		frontmatter = strings.TrimRight(frontmatter, " \t\r\n") + "\n" + block
		updated = "---" + frontmatter + "---" + content[end+3:]
	} else {
		updated = "---\n" + block + "---\n" + content
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat article: %w", err)
	}
	tmp := path + ".embedding.tmp"
	if err := os.WriteFile(tmp, []byte(updated), info.Mode().Perm()); err != nil {
		return fmt.Errorf("write article embedding: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace article embedding: %w", err)
	}
	return nil
}

func formatEmbeddingBlock(embedding parser.FrontmatterEmbedding) string {
	return fmt.Sprintf("embedding:\n  version: %d\n  model: %s\n  dimensions: %d\n  hash: %s\n  vector: %s\n  scale: %s\n",
		embedding.Version,
		strconv.Quote(embedding.Model),
		embedding.Dimensions,
		strconv.Quote(embedding.Hash),
		strconv.Quote(embedding.Vector),
		strconv.FormatFloat(float64(embedding.Scale), 'g', -1, 32),
	)
}

func removeEmbeddingBlock(frontmatter string) string {
	lines := strings.Split(frontmatter, "\n")
	out := make([]string, 0, len(lines))
	removing := false
	for _, line := range lines {
		if !removing && strings.HasPrefix(line, "embedding:") {
			removing = true
			continue
		}
		if removing {
			if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				continue
			}
			removing = false
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
