package extdoc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func put(t *testing.T, path, text string) {
	t.Helper()
	if err := write(path, []byte(text)); err != nil {
		t.Fatal(err)
	}
}

func TestParserInheritanceAndTags(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "a.js"), `/** @class Demo.Base Base description. @constructor Makes a base. @param {String} name Name. @xtype base */
Demo.Base = function(name){};
/** @cfg {Boolean} enabled (Optional) Whether enabled. */
/** @property value Value description. @type {Number} */ value: 1,
/** Runs. @param {String} text (Optional) Text. @return {Boolean} Success. @static */ run: function(text){},
/** Fires. @event change A change. @param {Object} sender Sender. */
/** @private @method secret */ secret: function(){},
/** @class Demo.Child Child description. @extends Demo.Base */
Demo.Child = function(){};
/** @cfg {Boolean} enabled @hide */
/** Runs differently. @method run @return {Number} Count. */ run: function(){},
/** @property value Child value. @type {String} */ value: "child",
/** @class Demo.Singleton @singleton @namespace Custom.Space @brick icon-special */
Demo.Singleton = {};
/** An alias. @member Demo.Base alias @method @return {Object} Alias. */ alias: function(){};
`)
	c, err := parse(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	base, child, singleton := c.ByName["Demo.Base"], c.ByName["Demo.Child"], c.ByName["Demo.Singleton"]
	if !base.HasConstructor || len(base.Params) != 1 || base.Params[0].Name != "name" {
		t.Fatal("constructor parameters lost")
	}
	if len(child.Ancestors) != 1 || child.Ancestors[0] != base {
		t.Fatal("inheritance lost")
	}
	found := map[string]*Member{}
	for _, m := range child.Members {
		found[m.Kind+":"+m.Name] = m
	}
	if found["cfg:enabled"] != nil || found["method:secret"] != nil {
		t.Fatal("hidden/private members leaked")
	}
	if found["method:run"] == nil || found["method:run"].Owner != child.Name || found["prop:value"].Type != "String" {
		t.Fatal("override lost")
	}
	if found["event:change"] == nil || found["event:change"].Owner != base.Name {
		t.Fatal("event inheritance lost")
	}
	if !singleton.Singleton || singleton.Package != "Custom.Space" || singleton.ShortName != "Singleton" {
		t.Fatal("namespace/singleton lost")
	}
	_, icons := navigation(c.Classes)
	if icons[singleton.Name] != "icon-special" {
		t.Fatal("custom icon lost")
	}
}

func TestStaticMethodsDoNotInherit(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "fixture.js"), `/** @class Base */ Base={};
/** Utility. @method utility @static */ utility:function(){},
/** @class Child @extends Base */ Child={};`)
	c, err := parse(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.ByName["Child"].Members) != 0 {
		t.Fatal("inherited a static method")
	}
}

func TestRepeatedClassAndMethod(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "fixture.js"), `/** @class Demo */ Demo={};
/** First. @method execute */ execute:function(){},
/** Second. @method execute */ execute:function(){},
/** @class Demo More detail. @constructor Construct. @param {Number} n Number. */ Demo=function(n){};`)
	c, err := parse(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	cls := c.ByName["Demo"]
	if len(c.Classes) != 1 || len(cls.DefinedIn) != 2 || !cls.HasConstructor || cls.Description != "More detail." {
		t.Fatal("class merging changed")
	}
	if len(cls.Members) != 1 || cls.Members[0].Description.Long != "Second." {
		t.Fatal("duplicate method did not replace previous definition")
	}
	count := 0
	for _, a := range c.Files[0].Anchors {
		if a.ID == "method-Demo-execute" {
			count++
		}
	}
	if count != 1 {
		t.Fatal("duplicate method source anchor")
	}
}

func TestLinksHighlightingAndLegacyDescriptions(t *testing.T) {
	d := describe(`See {@link Demo.Base#run the runner} and {@link #change}. <code>var text = 'hi'; // note
</code>`, "Demo.Child", true)
	if !strings.Contains(d.Long, `ext:cls="Demo.Base"`) || !strings.Contains(d.Long, `Demo.Child.html#Demo.Child-change`) {
		t.Fatal("inline member links lost")
	}
	if !strings.Contains(d.Long, "<b>var</b>") || !strings.Contains(d.Long, "<em>'hi'</em>") || !strings.Contains(d.Long, "<i>// note</i>") {
		t.Fatal("code highlighting changed")
	}
	if !d.HasShort || strings.Contains(d.Short, "<b>") {
		t.Fatal("short description changed")
	}
	if got := describe("See {@link Demo.Base unfinished", "Demo.Child", false).Long; got != "See Demo.Base unfinished" {
		t.Fatalf("legacy incomplete link: %q", got)
	}
	comment := parseComment("Text before @ suffix.\n@param {String} value Value.")
	if comment.Description != "Text before" || !comment.has("@") {
		t.Fatal("literal-at compatibility changed")
	}
	for _, name := range []string{"Ext.layout.BorderLayout.Region", "Ext.Updater.BasicRenderer"} {
		pkg, short := splitPackage(name)
		if pkg == "" || short == lastPart(name) {
			t.Fatalf("nested class package lost: %s", name)
		}
	}
}

func TestInputFailures(t *testing.T) {
	for _, test := range []struct{ name, source string }{
		{"unterminated", `/** @class Broken`},
		{"cycle", `/** @class A @extends B */ A={}; /** @class B @extends A */ B={};`},
		{"invalid class", `/** @class ../escape */ Escape={};`},
		{"orphan", `/** A public method. @method run */ run:function(){}`},
		{"no classes", `var x=1;`},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			put(t, filepath.Join(root, "fixture.js"), test.source)
			if _, err := parse(context.Background(), root); err == nil {
				t.Fatal("expected failure")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := parse(ctx, t.TempDir()); err == nil {
		t.Fatal("cancellation ignored")
	}
}

func TestSourcePagesEscapeCodeAndPreserveAnchors(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	assets := filepath.Join(root, "assets")
	out := filepath.Join(root, "out")
	put(t, filepath.Join(source, "a", "Same.js"), `/** @class Demo.A */ Demo.A={html:"<div>& text</div>"};`)
	put(t, filepath.Join(source, "b", "Same.js"), `/** @class Demo.B */ Demo.B={};`)
	for _, name := range []string{"index.html", "welcome.html", "license.txt", "resources/docs.css"} {
		put(t, filepath.Join(assets, name), "asset")
	}
	put(t, filepath.Join(assets, "source-wrap.html"), "<pre>###SOURCE###</pre>")
	if err := Generate(context.Background(), source, assets, out, io.Discard); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(out, "source", "Same.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `id="cls-Demo.A"`) || !strings.Contains(string(page), "&lt;div&gt;&amp; text&lt;/div&gt;") {
		t.Fatal("source escaping or anchors lost")
	}
	if _, err := os.Stat(filepath.Join(out, "source", "Same1.html")); err != nil {
		t.Fatal("duplicate source filename lost")
	}
	put(t, filepath.Join(assets, "source-wrap.html"), "missing marker")
	if err := Generate(context.Background(), source, assets, filepath.Join(root, "invalid"), io.Discard); err == nil {
		t.Fatal("invalid source wrapper accepted")
	}
}

type pageSnapshot struct {
	IDs   []string `json:"ids"`
	Links []string `json:"links"`
	Rows  []string `json:"rows"`
	Text  string   `json:"text"`
}

func pageFingerprint(data []byte) string {
	text := string(data)
	extract := func(pattern string) []string {
		result := []string{}
		for _, match := range regexp.MustCompile(pattern).FindAllStringSubmatch(text, -1) {
			result = append(result, match[1])
		}
		return result
	}
	var textParts []string
	for _, part := range regexp.MustCompile(`<[^>]*>`).Split(text, -1) {
		if normalized := strings.Join(strings.Fields(html.UnescapeString(part)), " "); normalized != "" {
			textParts = append(textParts, normalized)
		}
	}
	snapshot := pageSnapshot{IDs: extract(`\bid="([^"]*)"`), Links: extract(`<a\b[^>]*?\bhref="([^"]*)"`), Rows: extract(`<tr class="([^"]*)"`), Text: strings.Join(textParts, " ")}
	var encoded strings.Builder
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(snapshot)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.TrimSuffix(encoded.String(), "\n"))))
}

func TestJavaDocumentationBaseline(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("testdata", "java-pages.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]string
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "docs")
	if err := Generate(context.Background(), filepath.Join(root, "extjs-2.0.2", "source"), filepath.Join(root, "assets", "docs"), out, io.Discard); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(out, "output", "*.html"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != len(expected) {
		t.Fatalf("got %d class pages, Java had %d", len(files), len(expected))
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if actual := pageFingerprint(data); actual != expected[filepath.Base(file)] {
			t.Errorf("%s differs from Java baseline: %s", filepath.Base(file), actual)
		}
	}
	treeData, err := os.ReadFile(filepath.Join(out, "output", "tree.js"))
	if err != nil {
		t.Fatal(err)
	}
	treeText := strings.TrimPrefix(string(treeData), "Docs.classData = ")
	classJSON, remaining, found := strings.Cut(treeText, ";\nDocs.icons = ")
	if !found {
		t.Fatal("invalid navigation output")
	}
	var actual, reference struct {
		ClassData treeNode          `json:"classData"`
		Icons     map[string]string `json:"icons"`
	}
	if err := json.Unmarshal([]byte(classJSON), &actual.ClassData); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(strings.TrimSuffix(strings.TrimSpace(remaining), ";")), &actual.Icons); err != nil {
		t.Fatal(err)
	}
	referenceData, err := os.ReadFile(filepath.Join("testdata", "java-navigation.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(referenceData, &reference); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, reference) {
		t.Fatal("navigation and icons differ from Java reference")
	}
	// Every emitted source link must resolve to an actual anchor in its page.
	sourceLink := regexp.MustCompile(`href="source/([^"#]+)#([^"]*)"`)
	anchorPattern := regexp.MustCompile(`\bid="([^"]*)"`)
	anchors := map[string]map[string]bool{}
	for _, file := range files {
		data, _ := os.ReadFile(file)
		for _, link := range sourceLink.FindAllStringSubmatch(string(data), -1) {
			if anchors[link[1]] == nil {
				source, err := os.ReadFile(filepath.Join(out, "source", link[1]))
				if err != nil {
					t.Fatal(err)
				}
				anchors[link[1]] = map[string]bool{}
				for _, match := range anchorPattern.FindAllStringSubmatch(string(source), -1) {
					anchors[link[1]][match[1]] = true
				}
			}
			if !anchors[link[1]][link[2]] {
				t.Errorf("broken source anchor %s#%s", link[1], link[2])
			}
		}
	}
}
