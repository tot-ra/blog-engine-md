package embeddings

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tot-ra/blog-engine/internal/builder"
	"github.com/tot-ra/blog-engine/internal/parser"
)

// WriteFrontmatterEmbedding keeps the article itself as the portable source of
// truth. Path, URL, and language are deliberately derived during each build.
func WriteFrontmatterEmbedding(path string, contentType builder.ContentType, embedding parser.FrontmatterEmbedding) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read article frontmatter: %w", err)
	}
	content := string(data)
	block := formatEmbeddingBlock(embedding)

	var updated string
	if contentType == builder.TypeHTML {
		updated, err = writeHTMLFrontmatterEmbedding(content, block)
	} else {
		updated, err = writeYAMLFrontmatterEmbedding(content, block)
	}
	if err != nil {
		return err
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

func writeYAMLFrontmatterEmbedding(content, block string) (string, error) {
	if !strings.HasPrefix(content, "---") {
		return "---\n" + block + "---\n" + content, nil
	}
	end := strings.Index(content[3:], "---")
	if end < 0 {
		return "", fmt.Errorf("article has unterminated frontmatter")
	}
	end += 3
	frontmatter := removeEmbeddingBlock(content[3:end])
	frontmatter = strings.TrimRight(frontmatter, " \t\r\n") + "\n" + block
	return "---" + frontmatter + "---" + content[end+3:], nil
}

func writeHTMLFrontmatterEmbedding(content, block string) (string, error) {
	bom := ""
	trimmed := content
	if strings.HasPrefix(trimmed, "\ufeff") {
		bom = "\ufeff"
		trimmed = strings.TrimPrefix(trimmed, bom)
	}

	if strings.HasPrefix(trimmed, "<!--") {
		commentEnd := strings.Index(trimmed[4:], "-->")
		if commentEnd < 0 {
			return "", fmt.Errorf("article has unterminated HTML frontmatter comment")
		}
		commentEnd += 4
		comment := strings.TrimSpace(trimmed[4:commentEnd])
		if strings.HasPrefix(comment, "---") {
			updatedComment, err := writeYAMLFrontmatterEmbedding(comment, block)
			if err != nil {
				return "", err
			}
			// Keep metadata valid HTML while leaving the authored article body untouched.
			return bom + "<!--\n" + updatedComment + "\n-->" + trimmed[commentEnd+3:], nil
		}
	}

	return bom + "<!--\n---\n" + block + "---\n-->\n" + trimmed, nil
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
