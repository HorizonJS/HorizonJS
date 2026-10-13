package buildtool

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type browserPatch struct {
	File, Before, After string
	Count               int
}

var regexpVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

// Generate a separate strict-mode distribution. The legacy bundles retain
// their original bytes; all module changes are reviewed in overlay/module.
func generateBrowserModule(root, library, dest string) error {
	data, err := os.ReadFile(filepath.Join(root, "extjs-2.0.2", "source", "ext.jsb"))
	if err != nil {
		return err
	}
	var manifest project
	if err := xml.Unmarshal(data, &manifest); err != nil {
		return err
	}
	data, err = os.ReadFile(filepath.Join(root, "overlay", "module", "patches.json"))
	if err != nil {
		return err
	}
	var patches []browserPatch
	if err := json.Unmarshal(data, &patches); err != nil {
		return fmt.Errorf("module patches: %w", err)
	}
	missing, err := legacyMissing()
	if err != nil {
		return err
	}
	allowedMissing := make(map[string]bool)
	for _, name := range missing["source/ext.jsb"] {
		allowedMissing[name] = true
	}
	applied := make([]bool, len(patches))
	var module strings.Builder
	header, err := os.ReadFile(filepath.Join(root, "overlay", "module", "header.js"))
	if err != nil {
		return err
	}
	module.Write(header)
	// Match output paths rather than target labels so the existing manifest
	// remains the authority on initialization order.
	for _, suffix := range []string{"/adapter/ext/ext-base.js", "/ext-all.js"} {
		found := false
		for _, target := range manifest.Targets {
			if !strings.HasSuffix(strings.ReplaceAll(target.File, `\`, "/"), suffix) {
				continue
			}
			found = true
			for _, include := range target.Includes {
				name := strings.ReplaceAll(include.Name, `\`, "/")
				path, err := pathWithin(filepath.Join(library, "source"), name)
				if err != nil {
					return err
				}
				source, err := os.ReadFile(path)
				if os.IsNotExist(err) && allowedMissing[name] {
					continue
				}
				if err != nil {
					return err
				}
				text := strings.ReplaceAll(string(source), "\r\n", "\n")
				for index, patch := range patches {
					if patch.File != name {
						continue
					}
					if patch.Before == "" || patch.Count < 1 || strings.Count(text, patch.Before) != patch.Count {
						return fmt.Errorf("module patch for %s: expected %d occurrences of %q", name, patch.Count, patch.Before)
					}
					text = strings.ReplaceAll(text, patch.Before, patch.After)
					applied[index] = true
				}
				module.WriteString("\n// " + name + "\n")
				module.WriteString(text)
				module.WriteString("\n;\n")
			}
		}
		if !found {
			return fmt.Errorf("module build: missing target %s", suffix)
		}
	}
	for index, patch := range patches {
		if !applied[index] {
			return fmt.Errorf("module patch for %s was not applied to any included source", patch.File)
		}
	}
	footer, err := os.ReadFile(filepath.Join(root, "overlay", "module", "footer.js"))
	if err != nil {
		return err
	}
	module.Write(footer)
	return writeFile(filepath.Join(dest, "index.mjs"), []byte(module.String()))
}

func packageBrowser(root, library, dest, tag string) error {
	if err := copyTree(library, dest); err != nil {
		return err
	}
	if err := copyTree(filepath.Join(root, "overlay", "npm"), dest); err != nil {
		return err
	}
	license, err := os.ReadFile(filepath.Join(root, "extjs-2.0.2", "LICENSE.txt"))
	if err != nil {
		return err
	}
	if err := writeFile(filepath.Join(dest, "LICENSE.txt"), license); err != nil {
		return err
	}
	// Release tags set the npm version. Non-release builds use the template's
	// development version and can still be packed and installed locally.
	version := strings.TrimPrefix(tag, "v")
	if regexpVersion.MatchString(version) {
		path := filepath.Join(dest, "package.json")
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var manifest map[string]any
		if err := json.Unmarshal(data, &manifest); err != nil {
			return err
		}
		manifest["version"] = version
		data, err = json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return err
		}
		return writeFile(path, append(data, '\n'))
	}
	return nil
}
