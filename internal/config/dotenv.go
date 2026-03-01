package config

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

var (
	dotEnvMu          sync.Mutex
	dotEnvManagedKeys = make(map[string]struct{})
)

// LoadDotEnv loads environment variables from a .env file path.
// Existing process environment variables are not overridden.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:eq])
		if key == "" {
			continue
		}

		val := parseDotEnvValue(line[eq+1:])
		dotEnvMu.Lock()
		_, managed := dotEnvManagedKeys[key]
		dotEnvMu.Unlock()

		if !managed {
			if _, exists := os.LookupEnv(key); exists {
				continue
			}
		}
		_ = os.Setenv(key, val)
		dotEnvMu.Lock()
		dotEnvManagedKeys[key] = struct{}{}
		dotEnvMu.Unlock()
	}

	return scanner.Err()
}

func parseDotEnvValue(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}

	// Quoted values keep inner spaces and # chars.
	if strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") && len(v) >= 2 {
		return unescapeDoubleQuoted(v[1 : len(v)-1])
	}
	if strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'") && len(v) >= 2 {
		return v[1 : len(v)-1]
	}

	// Strip trailing inline comment for unquoted values: VALUE # comment.
	if idx := strings.Index(v, " #"); idx >= 0 {
		v = v[:idx]
	}
	return strings.TrimSpace(v)
}

func unescapeDoubleQuoted(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	escaped := false
	for _, r := range s {
		if escaped {
			switch r {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			default:
				b.WriteRune(r)
			}
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		b.WriteRune(r)
	}
	if escaped {
		b.WriteByte('\\')
	}
	return b.String()
}
