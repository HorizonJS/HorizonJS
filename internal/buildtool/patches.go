package buildtool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type sourcePatch struct {
	File, Before, After, AfterFile string
	Count                          int
}

func loadSourcePatches(path string) ([]sourcePatch, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var patches []sourcePatch
	if err := json.Unmarshal(data, &patches); err != nil {
		return nil, fmt.Errorf("patches %s: %w", path, err)
	}
	for index, patch := range patches {
		if patch.File == "" || strings.Contains(patch.File, `\`) || filepath.IsAbs(patch.File) || strings.HasPrefix(patch.File, "/") || filepath.ToSlash(filepath.Clean(patch.File)) != patch.File || strings.HasPrefix(patch.File, "../") || patch.File == ".." || patch.Before == "" || patch.Count < 1 {
			return nil, fmt.Errorf("invalid source patch for %q", patch.File)
		}
		if patch.AfterFile != "" {
			if patch.After != "" {
				return nil, fmt.Errorf("source patch for %s specifies both After and AfterFile", patch.File)
			}
			replacement, err := pathWithin(filepath.Dir(path), patch.AfterFile)
			if err != nil {
				return nil, err
			}
			info, err := os.Lstat(replacement)
			if err != nil {
				return nil, err
			}
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("replacement is not a regular file: %s", replacement)
			}
			realReplacement, err := filepath.EvalSymlinks(replacement)
			if err != nil {
				return nil, err
			}
			if _, err := pathWithin(filepath.Dir(path), realReplacement); err != nil {
				return nil, err
			}
			data, err := os.ReadFile(replacement)
			if err != nil {
				return nil, err
			}
			patches[index].After = strings.ReplaceAll(string(data), "\r\n", "\n")
		}
	}
	return patches, nil
}

// Apply reviewed changes before minification and source staging. Count checks
// prevent source drift from silently disabling a security fix.
func applySourcePatches(text, name string, patches []sourcePatch, applied []bool) (string, error) {
	for index, patch := range patches {
		if patch.File != name {
			continue
		}
		text = strings.ReplaceAll(text, "\r\n", "\n")
		if count := strings.Count(text, patch.Before); count != patch.Count {
			return "", fmt.Errorf("source patch for %s: expected %d occurrences, found %d", name, patch.Count, count)
		}
		text = strings.ReplaceAll(text, patch.Before, patch.After)
		applied[index] = true
	}
	return text, nil
}

func checkAppliedPatches(patches []sourcePatch, applied []bool) error {
	for index, patch := range patches {
		if !applied[index] {
			return fmt.Errorf("source patch for %s was not applied", patch.File)
		}
	}
	return nil
}
