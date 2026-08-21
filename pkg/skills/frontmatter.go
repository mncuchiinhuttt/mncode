package skills

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseFrontmatter extracts YAML frontmatter and body from markdown content
func ParseFrontmatter(content []byte, target interface{}) (string, error) {
	str := string(content)
	str = strings.TrimSpace(str)

	if !strings.HasPrefix(str, "---") {
		// No frontmatter, entire content is body
		return str, nil
	}

	// Find the closing ---
	rest := str[3:]
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		// No closing delimiter found
		return str, nil
	}

	frontmatterText := rest[:idx]
	body := strings.TrimSpace(rest[idx+4:])

	// Decode YAML into target if target is provided
	if target != nil {
		decoder := yaml.NewDecoder(bytes.NewBufferString(frontmatterText))
		if err := decoder.Decode(target); err != nil {
			return body, err
		}
	}

	return body, nil
}
