package buildtool

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	contents := map[string]string{
		"extjs-2.0.2/source/main.js":                 "/* fixture */\nvar fixture = true;\n",
		"extjs-2.0.2/source/ext.jsb":                 `<project name="fixture" author="test" version="1" copyright="fixture" source="True" source-dir="$output/source" minify="True" min-dir="$output/build"><file name="main.js" path=""/><target name="all" file="$output/ext-all.js" debug="True"><include name="main.js"/></target><target name="core" file="$output/ext-core.js" debug="True"><include name="main.js"/></target><target name="adapter" file="$output/adapter/ext/ext-base.js"><include name="main.js"/></target></project>`,
		"extjs-2.0.2/resources/resources.jsb":        `<project name="resources" author="test" version="1" copyright="fixture" source="True" source-dir="$output" minify="False" min-dir="$output/build"><file name="css/core.css" path="css"/><file name="css/xtheme-gray.css" path="css"/><file name="images/default/s.gif" path="images/default"/><target name="css" file="$output/css/ext-all.css" debug="True"><include name="css/core.css"/></target></project>`,
		"extjs-2.0.2/resources/css/core.css":         "body { color: black; }",
		"extjs-2.0.2/resources/css/xtheme-gray.css":  "body { color: gray; }",
		"extjs-2.0.2/resources/images/default/s.gif": "image",
		"extjs-2.0.2/examples/index.html":            "examples",
		"overlay/build/README.md":                    "library overlay",
		"overlay/docs/README.md":                     "docs overlay",
		"overlay/examples/README.md":                 "examples overlay",
		"overlay/website/index.html":                 "<html>__HORIZONJS_BUILD_DATE__</html>",
		"README.md":                                  "readme",
		"extjs-2.0.2/LICENSE.txt":                    "fixture license",
		"overlay/module/header.js":                   "var Ext;\n",
		"overlay/module/footer.js":                   "export default Ext;\n",
		"overlay/module/patches.json":                "[]",
		"overlay/npm/package.json":                   `{"name":"@horizonjs/core","version":"0.1.0-dev.0"}`,
		"EXTJS_2_0_2_USAGE.md":                       "usage",
	}
	for name, content := range contents {
		mustWrite(t, filepath.Join(root, filepath.FromSlash(name)), content)
	}
	fixtureDocs(t, filepath.Join(root, "extjs-2.0.2", "docs"))
	mustWrite(t, filepath.Join(root, "BUILD_TOOL.md"), "build tool guide")
	return root
}

func fixtureDocs(t *testing.T, root string) {
	t.Helper()
	for _, name := range []string{"index.html", "output/Ext.Panel.html", "output/Ext.data.Store.html", "output/Ext.grid.GridPanel.html"} {
		mustWrite(t, filepath.Join(root, filepath.FromSlash(name)), "<html>docs</html>")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := writeFile(path, []byte(content)); err != nil {
		t.Fatal(err)
	}
}

func TestBuildPublishesCompleteOutput(t *testing.T) {
	root := fixture(t)
	mustWrite(t, filepath.Join(root, "build", "old.txt"), "previous build")
	var log bytes.Buffer
	if err := Build(context.Background(), Options{Root: root, Tag: "main", Docs: "bundled", Log: &log}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "build", "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("stale file remains: %v", err)
	}
	if err := validateArtifacts(filepath.Join(root, "build")); err != nil {
		t.Fatal(err)
	}
	website, err := os.ReadFile(filepath.Join(root, "build", "website", "index.html"))
	if err != nil || strings.Contains(string(website), "__HORIZONJS_BUILD_DATE__") {
		t.Fatalf("website stamp: %s, %v", website, err)
	}
	for _, name := range []string{"docs/index.html", "examples/index.html", "release/horizonjs-main.zip", "README.md", "BUILD_TOOL.md"} {
		if _, err := os.Stat(filepath.Join(root, "build", "website", filepath.FromSlash(name))); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"index.mjs", "package.json", "LICENSE.txt", "resources/images/default/s.gif", "resources/css/ext-all.css"} {
		if _, err := os.Stat(filepath.Join(root, "build", "npm", filepath.FromSlash(name))); err != nil {
			t.Fatal(err)
		}
	}
	reader, err := zip.OpenReader(filepath.Join(root, "build", "zips", "horizonjs-main.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	files := map[string]bool{}
	for _, file := range reader.File {
		if strings.Contains(file.Name, `\`) || strings.HasPrefix(file.Name, "source/") || strings.HasPrefix(file.Name, "package/") {
			t.Fatalf("unexpected archive entry %q", file.Name)
		}
		contents, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		_, readErr := io.Copy(io.Discard, contents)
		closeErr := contents.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("invalid zip member: %v %v", readErr, closeErr)
		}
		files[file.Name] = true
	}
	for _, name := range []string{"ext-all.js", "ext-all-debug.js", "adapter/ext/ext-base.js", "resources/css/ext-all.css"} {
		if !files[name] {
			t.Fatalf("missing zip entry %q", name)
		}
	}
	assertNoTemporaryBuilds(t, root)
}

func TestBuildFailuresPreservePreviousOutput(t *testing.T) {
	for _, test := range []struct {
		name   string
		damage func(*testing.T, string)
	}{
		{"missing source", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "extjs-2.0.2", "source", "main.js")); err != nil {
				t.Fatal(err)
			}
		}},
		{"invalid javascript", func(t *testing.T, root string) {
			mustWrite(t, filepath.Join(root, "extjs-2.0.2", "source", "main.js"), "var broken = 'unterminated;")
		}},
		{"missing resource", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "extjs-2.0.2", "resources", "images", "default", "s.gif")); err != nil {
				t.Fatal(err)
			}
		}},
		{"empty overlay artifact", func(t *testing.T, root string) {
			mustWrite(t, filepath.Join(root, "overlay", "build", "ext-all.js"), "")
		}},
		{"missing documentation", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "extjs-2.0.2", "docs", "output", "Ext.Panel.html")); err != nil {
				t.Fatal(err)
			}
		}},
		{"missing website", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "overlay", "website", "index.html")); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := fixture(t)
			mustWrite(t, filepath.Join(root, "build", "previous.txt"), "previous build")
			test.damage(t, root)
			var log bytes.Buffer
			if err := Build(context.Background(), Options{Root: root, Tag: "main", Docs: "bundled", Log: &log}); err == nil {
				t.Fatal("expected failure")
			}
			previous, err := os.ReadFile(filepath.Join(root, "build", "previous.txt"))
			if err != nil || string(previous) != "previous build" {
				t.Fatalf("previous output lost: %v", err)
			}
			if strings.Contains(log.String(), "completed successfully") {
				t.Fatal("reported success")
			}
			if _, err := os.Stat(filepath.Join(root, "build", "zips")); !os.IsNotExist(err) {
				t.Fatal("published incomplete archives")
			}
			assertNoTemporaryBuilds(t, root)
		})
	}
}

func TestRemovedDocsModePreservesOutput(t *testing.T) {
	root := fixture(t)
	mustWrite(t, filepath.Join(root, "build", "previous.txt"), "previous build")
	var log bytes.Buffer
	err := Build(context.Background(), Options{Root: root, Tag: "main", Docs: "docker", Log: &log})
	if err == nil || !strings.Contains(err.Error(), "invalid docs mode") {
		t.Fatalf("expected removed docs mode to be rejected: %v", err)
	}
	previous, err := os.ReadFile(filepath.Join(root, "build", "previous.txt"))
	if err != nil || string(previous) != "previous build" {
		t.Fatal("invalid docs mode lost previous build")
	}
	if log.Len() != 0 {
		t.Fatal("invalid docs mode started the build")
	}
	assertNoTemporaryBuilds(t, root)
}

func TestCancellationAndUnsafeOutput(t *testing.T) {
	root := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Build(ctx, Options{Root: root, Tag: "main", Docs: "bundled"}); err == nil {
		t.Fatal("cancelled build succeeded")
	}
	for _, out := range []string{root, filepath.Dir(root), filepath.Join(root, "extjs-2.0.2"), filepath.Join(root, "overlay", "nested"), filepath.Join(root, "assets", "docs")} {
		if err := Build(context.Background(), Options{Root: root, Out: out, Tag: "main", Docs: "bundled"}); err == nil {
			t.Fatalf("unsafe output accepted: %s", out)
		}
	}
	assertNoTemporaryBuilds(t, root)
}

func TestManifestRejectsInvalidInputs(t *testing.T) {
	for _, test := range []struct {
		name, files, includes string
		allow                 []string
	}{
		{"missing include", `<file name="present.js"/><file name="missing.js"/>`, `<include name="present.js"/><include name="missing.js"/>`, nil},
		{"all missing", `<file name="missing.js"/>`, `<include name="missing.js"/>`, nil},
		{"all allowed missing", `<file name="missing.js"/>`, `<include name="missing.js"/>`, []string{"missing.js"}},
		{"undeclared include", `<file name="present.js"/>`, `<include name="missing.js"/>`, nil},
		{"empty target", `<file name="present.js"/>`, ``, nil},
		{"traversal", `<file name="../outside.js"/>`, `<include name="../outside.js"/>`, nil},
		{"duplicate input", `<file name="present.js"/><file name="present.js"/>`, `<include name="present.js"/>`, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			mustWrite(t, filepath.Join(root, "present.js"), "var present=true;")
			manifest := `<project source="True" source-dir="$output/source" minify="True" min-dir="$output/min">` + test.files + `<target name="bundle" file="$output/bundle.js">` + test.includes + `</target></project>`
			mustWrite(t, filepath.Join(root, "project.jsb"), manifest)
			if err := buildProject(filepath.Join(root, "project.jsb"), filepath.Join(root, "out"), test.allow, io.Discard); err == nil {
				t.Fatal("expected failure")
			}
			if _, err := os.Stat(filepath.Join(root, "out", "bundle.js")); !os.IsNotExist(err) {
				t.Fatal("wrote incomplete bundle")
			}
		})
	}
}

func TestJSMinMalformedInput(t *testing.T) {
	for _, input := range []string{"/* unfinished", "var x='unfinished", "var x=/unfinished"} {
		if _, err := minifyJS(input); err == nil {
			t.Fatalf("accepted malformed source %q", input)
		}
	}
	input := "// comment\r\nvar url = 'https://example.test/a'; var re = /a\\/b/;"
	output, err := minifyJS(input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "https://example.test/a") || !strings.Contains(output, `/a\/b/`) {
		t.Fatalf("corrupted literals: %s", output)
	}
}

func TestOriginalBundleCompatibility(t *testing.T) {
	root, err := repositoryRoot("")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "build")
	if err := Build(context.Background(), Options{Root: root, Out: out, Tag: "test", Docs: "go"}); err != nil {
		t.Fatal(err)
	}
	// SHA-256 values captured from the original C# builder before migration,
	// normalizing CRLF to LF because Git checkouts differ across platforms.
	golden := map[string]string{
		"ext-all.js":                    "6673c539f303175033ed782314561cdda71bd60c506afb0c2f5bedfab0781371",
		"ext-all-debug.js":              "db42c5771d69d77d34bcf0aa1dc4074f558af37b9dc232b79cf4660bed4bc165",
		"ext-core.js":                   "8ec8bf7a63377f3e4d8f5bd8706270aa2053f5332b88b7499a7fa08e332762ce",
		"ext-core-debug.js":             "0a2585d2ee0472f5f9ef4f94e3eb02174accac3e591dabf8ec1a6dfff6a010c0",
		"adapter/ext/ext-base.js":       "cff4e3bea8486ad62e40b055b03be8b839b9b2f1fc37e640d81f6bca360c06ce",
		"resources/css/ext-all.css":     "f844f2ab0510754cbb18ccba97f3f915c40fb1d59bc9573794ee4347432c832e",
		"resources/css/xtheme-gray.css": "fc89822c7db8abdd8f6e1f4f370394b16e76dfe14ca1b7384dc25778adf12b0f",
	}
	for name, expected := range golden {
		data, err := os.ReadFile(filepath.Join(out, "build", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		actual := fmt.Sprintf("%x", sha256.Sum256(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))))
		if actual != expected {
			t.Errorf("%s differs from C# baseline: got %s", name, actual)
		}
	}
}

func TestNativeDocsFailurePreservesOutput(t *testing.T) {
	root := fixture(t)
	mustWrite(t, filepath.Join(root, "build", "previous.txt"), "previous build")
	mustWrite(t, filepath.Join(root, "extjs-2.0.2", "source", "main.js"), "/** @class Demo @extends Demo */ var Demo = {};\n")
	var log bytes.Buffer
	err := Build(context.Background(), Options{Root: root, Tag: "main", Docs: "go", Log: &log})
	if err == nil || !strings.Contains(err.Error(), "inheritance cycle") {
		t.Fatalf("expected native docs failure: %v", err)
	}
	previous, err := os.ReadFile(filepath.Join(root, "build", "previous.txt"))
	if err != nil || string(previous) != "previous build" {
		t.Fatal("native docs failure lost previous build")
	}
	if strings.Contains(log.String(), "Packaging") || strings.Contains(log.String(), "completed successfully") {
		t.Fatal("native docs failure proceeded to packaging")
	}
	assertNoTemporaryBuilds(t, root)
}

func assertNoTemporaryBuilds(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".horizon-stage-") || strings.HasPrefix(entry.Name(), ".horizon-backup-") {
			t.Fatalf("left temporary build %s", entry.Name())
		}
	}
}
