package buildtool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSharedPatchesReachClassicAndModule(t *testing.T) {
	root := fixture(t)
	mustWrite(t, filepath.Join(root, "overlay/source/replacement.js"), "var fixture = 'hardened';\n")
	mustWrite(t, filepath.Join(root, "overlay/source/patches.json"), `[{"File":"main.js","Before":"var fixture = true;","AfterFile":"replacement.js","Count":1}]`)
	if err := Build(context.Background(), Options{Root: root, Tag: "test", Docs: "bundled"}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"build/ext-all.js", "build/ext-all-debug.js", "build/ext-core.js", "build/ext-core-debug.js", "npm/index.mjs", "npm/ext-all.js"} {
		data, err := os.ReadFile(filepath.Join(root, "build", filepath.FromSlash(name)))
		if err != nil || !strings.Contains(string(data), "'hardened'") || strings.Contains(string(data), "fixture = true") {
			t.Fatalf("shared patch missing from %s: %v", name, err)
		}
	}
	original, err := os.ReadFile(filepath.Join(root, "extjs-2.0.2/source/main.js"))
	if err != nil || !strings.Contains(string(original), "fixture = true") {
		t.Fatal("shared patch changed upstream input")
	}
}

func TestSharedPatchFailuresPreserveOutput(t *testing.T) {
	for _, patch := range []sourcePatch{
		{File: "main.js", Before: "missing", After: "replacement", Count: 1},
		{File: "main.js", Before: "fixture", After: "replacement", Count: 3},
		{File: "missing.js", Before: "fixture", After: "replacement", Count: 1},
		{File: "../main.js", Before: "fixture", After: "replacement", Count: 1},
		{File: "main.js", Before: "", After: "replacement", Count: 1},
		{File: "main.js", Before: "fixture", After: "replacement", Count: 0},
		{File: "main.js", Before: "fixture", AfterFile: "missing.js", Count: 1},
		{File: "main.js", Before: "fixture", AfterFile: "../../README.md", Count: 1},
	} {
		t.Run(patch.File+patch.Before+patch.AfterFile, func(t *testing.T) {
			root := fixture(t)
			data, err := json.Marshal([]sourcePatch{patch})
			if err != nil {
				t.Fatal(err)
			}
			mustWrite(t, filepath.Join(root, "overlay/source/patches.json"), string(data))
			mustWrite(t, filepath.Join(root, "build/previous.txt"), "previous build")
			if err := Build(context.Background(), Options{Root: root, Tag: "main", Docs: "bundled"}); err == nil {
				t.Fatal("unsafe or stale patch accepted")
			}
			previous, err := os.ReadFile(filepath.Join(root, "build/previous.txt"))
			if err != nil || string(previous) != "previous build" {
				t.Fatal("patch failure replaced previous output")
			}
			assertNoTemporaryBuilds(t, root)
		})
	}
}
