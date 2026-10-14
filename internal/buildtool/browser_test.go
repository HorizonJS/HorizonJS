package buildtool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserPackageVersionAndPatches(t *testing.T) {
	root := fixture(t)
	patches := []sourcePatch{{File: "main.js", Before: "var fixture = true;", After: "var fixture = 'module';", Count: 1}}
	data, err := json.Marshal(patches)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "overlay/module/patches.json"), string(data))
	if err := Build(context.Background(), Options{Root: root, Tag: "v1.2.3-beta.1", Docs: "bundled"}); err != nil {
		t.Fatal(err)
	}
	module, err := os.ReadFile(filepath.Join(root, "build/npm/index.mjs"))
	if err != nil || !strings.Contains(string(module), "var fixture = 'module';") {
		t.Fatalf("module patch missing: %v", err)
	}
	classic, err := os.ReadFile(filepath.Join(root, "build/build/ext-all-debug.js"))
	if err != nil || !strings.Contains(string(classic), "var fixture = true;") {
		t.Fatalf("module patch modified classic bundle: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(root, "build/npm/package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct{ Version string }
	if err := json.Unmarshal(data, &manifest); err != nil || manifest.Version != "1.2.3-beta.1" {
		t.Fatalf("npm release version: %s, %v", data, err)
	}
}

func TestBrowserPatchDriftPreservesOutput(t *testing.T) {
	for _, patch := range []sourcePatch{
		{File: "main.js", Before: "missing text", After: "replacement", Count: 1},
		{File: "not-included.js", Before: "var fixture = true;", After: "replacement", Count: 1},
	} {
		root := fixture(t)
		data, err := json.Marshal([]sourcePatch{patch})
		if err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(root, "overlay/module/patches.json"), string(data))
		mustWrite(t, filepath.Join(root, "build/previous.txt"), "previous output")
		if err := Build(context.Background(), Options{Root: root, Tag: "main", Docs: "bundled"}); err == nil || !strings.Contains(err.Error(), "source patch") {
			t.Fatalf("expected patch failure: %v", err)
		}
		previous, err := os.ReadFile(filepath.Join(root, "build/previous.txt"))
		if err != nil || string(previous) != "previous output" {
			t.Fatalf("failed module build replaced old output: %v", err)
		}
		assertNoTemporaryBuilds(t, root)
	}
}
