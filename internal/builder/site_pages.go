package builder

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tot-ra/blog-engine/internal/parser"
)

func (b *SiteBuilder) buildPages(files []ContentFile, titleToURL, pathToURL map[string]map[string]string, explicitIndexDirs map[string]struct{}) ([]*Page, []error) {
	if len(files) == 0 {
		return nil, nil
	}

	pages := make([]*Page, len(files))
	workers := b.workerCount()
	if workers > len(files) {
		workers = len(files)
	}

	type buildJob struct {
		idx  int
		file ContentFile
	}

	jobs := make(chan buildJob, len(files))
	errCh := make(chan error, len(files))
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		builder := NewPageBuilder(b.config.Site.URL, b.config.I18n.Default, b.languages)
		builder.urlGen.SetExplicitIndexDirs(explicitIndexDirs)

		wg.Add(1)
		go func(pb *PageBuilder) {
			defer wg.Done()
			for job := range jobs {
				lang, _ := detectLanguageAndContentPath(job.file.RelativePath, b.config.I18n.Default, b.languages)
				pageMap := titleToURL[lang]
				pathMap := pathToURL[lang]
				pb.SetPageResolver(func(title string) (string, bool) {
					if url, ok := pageMap[title]; ok {
						return url, true
					}
					slug := parser.GenerateSlug(title)
					if url, ok := pageMap[slug]; ok {
						return url, true
					}
					return "", false
				})
				pb.SetMarkdownLinkResolver(func(destination, pageRelPath string) (string, bool) {
					return resolveLocalMarkdownLink(destination, pageRelPath, pathMap)
				})
				page, err := pb.Build(job.file)
				if err != nil {
					errCh <- fmt.Errorf("%s: %w", job.file.Path, err)
					continue
				}
				pages[job.idx] = page
			}
		}(builder)
	}

	for i, file := range files {
		jobs <- buildJob{idx: i, file: file}
	}
	close(jobs)
	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	result := make([]*Page, 0, len(files))
	for _, page := range pages {
		if page != nil {
			result = append(result, page)
		}
	}
	return result, errs
}

func resolveLocalMarkdownLink(destination, pageRelPath string, pathToURL map[string]string) (string, bool) {
	if len(pathToURL) == 0 || destination == "" || strings.HasPrefix(destination, "#") {
		return "", false
	}
	if strings.HasPrefix(destination, "/") || strings.HasPrefix(destination, "//") {
		return "", false
	}
	if u, err := url.Parse(destination); err == nil && u.IsAbs() {
		return "", false
	}

	pathPart := destination
	suffix := ""
	if idx := strings.IndexAny(pathPart, "?#"); idx >= 0 {
		suffix = pathPart[idx:]
		pathPart = pathPart[:idx]
	}
	if decoded, err := url.PathUnescape(pathPart); err == nil {
		pathPart = decoded
	}
	if _, ok := markdownExtensions[strings.ToLower(filepath.Ext(pathPart))]; !ok {
		return "", false
	}

	pageDir := filepath.ToSlash(filepath.Dir(pageRelPath))
	if pageDir == "." {
		pageDir = ""
	}
	candidates := []string{
		normalizeMarkdownLinkPath(filepath.ToSlash(filepath.Join(pageDir, pathPart))),
		normalizeMarkdownLinkPath(pathPart),
	}
	for _, candidate := range candidates {
		if url, ok := pathToURL[candidate]; ok {
			return url + suffix, true
		}
	}
	return "", false
}

func normalizeMarkdownLinkPath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "./")
	return strings.Trim(path, "/")
}

func addMarkdownLinkPathAliases(pathToURL map[string]string, relPath, pageURL string) {
	normalized := normalizeMarkdownLinkPath(relPath)
	if normalized == "" {
		return
	}
	pathToURL[normalized] = pageURL

	parts := strings.Split(normalized, "/")
	if len(parts) > 1 {
		withoutTopSection := strings.Join(parts[1:], "/")
		if withoutTopSection != "" {
			pathToURL[withoutTopSection] = pageURL
		}
	}

func collectExplicitIndexDirs(files []ContentFile) map[string]struct{} {
	if len(files) == 0 {
		return nil
	}
	dirs := make(map[string]struct{})
	for _, file := range files {
		filename := filepath.Base(file.RelativePath)
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)
		if name != "index" && name != "README" {
			continue
		}
		dir := normalizeMarkdownLinkPath(filepath.Dir(file.RelativePath))
		if dir == "." {
			dir = ""
		}
		dirs[dir] = struct{}{}
	}
	return dirs
}
}
