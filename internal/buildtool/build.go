package buildtool

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"horizonjs/internal/extdoc"
)

type Options struct {
	Root, Out, Tag, Docs string
	Log                  io.Writer
}

var validTag = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func Build(ctx context.Context, opts Options) (resultErr error) {
	if !validTag.MatchString(opts.Tag) {
		return fmt.Errorf("invalid release tag %q", opts.Tag)
	}
	if opts.Docs != "go" && opts.Docs != "bundled" {
		return fmt.Errorf("invalid docs mode %q; use go or bundled", opts.Docs)
	}
	if opts.Log == nil {
		opts.Log = io.Discard
	}
	root, err := repositoryRoot(opts.Root)
	if err != nil {
		return err
	}
	out := opts.Out
	if out == "" {
		out = filepath.Join(root, "build")
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return err
	}
	if err := validateOutput(root, out); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(out), ".horizon-stage-")
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, os.RemoveAll(stage)) }()
	build := filepath.Join(stage, "build")
	library := filepath.Join(build, "build")
	if err := os.MkdirAll(library, 0755); err != nil {
		return err
	}
	missing, err := legacyMissing()
	if err != nil {
		return err
	}
	fmt.Fprintln(opts.Log, "Building library...")
	for _, relative := range []string{"source/ext.jsb", "resources/resources.jsb"} {
		if err := ctx.Err(); err != nil {
			return err
		}
		dest := library
		if strings.HasPrefix(relative, "resources/") {
			dest = filepath.Join(library, "resources")
		}
		if err := buildProject(filepath.Join(root, "extjs-2.0.2", filepath.FromSlash(relative)), dest, missing[relative], opts.Log); err != nil {
			return err
		}
	}
	fmt.Fprintf(opts.Log, "Preparing documentation (%s)...\n", opts.Docs)
	docs := filepath.Join(build, "docs")
	if opts.Docs == "go" {
		if err := extdoc.Generate(ctx, filepath.Join(library, "source"), filepath.Join(root, "assets", "docs"), docs, opts.Log); err != nil {
			return err
		}
	} else {
		if err := copyTree(filepath.Join(root, "extjs-2.0.2", "docs"), docs); err != nil {
			return err
		}
	}
	fmt.Fprintln(opts.Log, "Building browser module...")
	if err := generateBrowserModule(root, library, filepath.Join(build, "npm")); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := copyTree(filepath.Join(root, "extjs-2.0.2", "examples"), filepath.Join(build, "examples")); err != nil {
		return err
	}
	// Preserve the previous distribution layout, including the gray theme.
	for _, relative := range []string{"source", "build", "package", "resources/raw-images", "resources/license.txt", "resources/resources.jsb"} {
		if err := os.RemoveAll(filepath.Join(library, filepath.FromSlash(relative))); err != nil {
			return fmt.Errorf("cleanup %s: %w", relative, err)
		}
	}
	cssDir := filepath.Join(library, "resources", "css")
	cssFiles, err := os.ReadDir(cssDir)
	if err != nil {
		return err
	}
	for _, file := range cssFiles {
		if !file.IsDir() && file.Name() != "ext-all.css" && file.Name() != "xtheme-gray.css" {
			if err := os.Remove(filepath.Join(cssDir, file.Name())); err != nil {
				return err
			}
		}
	}
	for _, name := range []string{"build", "docs", "examples"} {
		if err := copyTree(filepath.Join(root, "overlay", name), filepath.Join(build, name)); err != nil {
			return fmt.Errorf("overlay %s: %w", name, err)
		}
	}
	if err := validateArtifacts(build); err != nil {
		return err
	}
	if err := packageBrowser(root, library, filepath.Join(build, "npm"), opts.Tag); err != nil {
		return err
	}
	fmt.Fprintln(opts.Log, "Packaging validated output...")
	for _, item := range []struct{ directory, filename string }{
		{"build", "horizonjs-" + opts.Tag + ".zip"},
		{"docs", "horizonjs-docs-" + opts.Tag + ".zip"},
		{"examples", "horizonjs-examples-" + opts.Tag + ".zip"},
	} {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := archive(filepath.Join(build, item.directory), filepath.Join(build, "zips", item.filename)); err != nil {
			return err
		}
	}
	if opts.Tag == "main" {
		website := filepath.Join(build, "website")
		for _, item := range []struct{ from, to string }{{"build", ""}, {"docs", "docs"}, {"examples", "examples"}, {"zips", "release"}} {
			if err := copyTree(filepath.Join(build, item.from), filepath.Join(website, item.to)); err != nil {
				return err
			}
		}
		if err := copyTree(filepath.Join(root, "overlay", "website"), website); err != nil {
			return err
		}
		index := filepath.Join(website, "index.html")
		content, err := os.ReadFile(index)
		if err != nil {
			return err
		}
		if len(content) == 0 {
			return fmt.Errorf("website index is empty")
		}
		content = bytes.ReplaceAll(content, []byte("__HORIZONJS_BUILD_DATE__"), []byte(time.Now().UTC().Format(time.RFC3339)))
		if err := writeFile(index, content); err != nil {
			return err
		}
		for _, name := range []string{"README.md", "docs/README.md", "examples/README.md"} {
			if err := os.Remove(filepath.Join(website, filepath.FromSlash(name))); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		for _, name := range []string{"README.md", "EXTJS_2_0_2_USAGE.md", "BUILD_TOOL.md"} {
			content, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				return err
			}
			if err := writeFile(filepath.Join(website, name), content); err != nil {
				return err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := publish(build, out); err != nil {
		return err
	}
	if err := os.RemoveAll(stage); err != nil {
		return fmt.Errorf("remove staging directory: %w", err)
	}
	fmt.Fprintf(opts.Log, "Build completed successfully: %s\n", out)
	return nil
}

func repositoryRoot(root string) (string, error) {
	if root != "" {
		resolved, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		resolved, err = filepath.EvalSymlinks(resolved)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(filepath.Join(resolved, "extjs-2.0.2", "source", "ext.jsb")); err != nil {
			return "", fmt.Errorf("repository root: %w", err)
		}
		return resolved, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(cwd, "extjs-2.0.2", "source", "ext.jsb")); err == nil {
			return repositoryRoot(cwd)
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			return "", fmt.Errorf("cannot find HorizonJS repository; pass --root")
		}
		cwd = parent
	}
}

func validateOutput(root, out string) error {
	// Resolve the existing parent to prevent a symlink from redirecting staging
	// or replacement into the source tree. The final destination cannot be a link.
	parent := filepath.Dir(out)
	for {
		if _, err := os.Lstat(parent); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return err
		}
		parent = filepath.Dir(parent)
	}
	realParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(parent, out)
	if err != nil {
		return err
	}
	resolved := filepath.Join(realParent, rel)
	if info, err := os.Lstat(out); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("output must be a directory, not a file or symlink: %s", out)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	// Output must not contain the repository or overlap any of its inputs.
	if _, err := pathWithin(resolved, root); err == nil {
		return fmt.Errorf("output %s contains the repository", out)
	}
	for _, name := range []string{"extjs-2.0.2", "assets", "overlay", "cmd", "internal", "scripts", "tasks", ".git", ".github"} {
		input := filepath.Join(root, name)
		if _, err := pathWithin(input, resolved); err == nil {
			return fmt.Errorf("output overlaps input %s", input)
		}
		if _, err := pathWithin(resolved, input); err == nil {
			return fmt.Errorf("output contains input %s", input)
		}
	}
	return nil
}

func copyTree(source, dest string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsupported symlink: %s", path)
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported file: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return writeFile(target, data)
	})
}

func validateArtifacts(root string) error {
	for _, relative := range []string{
		"build/ext-all.js", "build/ext-all-debug.js", "build/ext-core.js", "build/ext-core-debug.js",
		"build/adapter/ext/ext-base.js", "build/resources/css/ext-all.css", "build/resources/css/xtheme-gray.css",
		"build/resources/images/default/s.gif", "docs/index.html", "docs/output/Ext.Panel.html",
		"docs/output/Ext.data.Store.html", "docs/output/Ext.grid.GridPanel.html", "examples/index.html",
	} {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return fmt.Errorf("required artifact %s: %w", relative, err)
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return fmt.Errorf("required artifact %s is empty or not a regular file", relative)
		}
	}
	return nil
}

func archive(source, dest string) (resultErr error) {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	writer := zip.NewWriter(file)
	defer func() { resultErr = errors.Join(resultErr, writer.Close()) }()
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("cannot archive non-regular file %s", path)
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name, header.Method = filepath.ToSlash(rel), zip.Deflate
		output, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = output.Write(data)
		return err
	})
}

func publish(stage, out string) error {
	if _, err := os.Lstat(out); os.IsNotExist(err) {
		return os.Rename(stage, out)
	} else if err != nil {
		return err
	}
	backup, err := os.MkdirTemp(filepath.Dir(out), ".horizon-backup-")
	if err != nil {
		return err
	}
	if err := os.Remove(backup); err != nil {
		return err
	}
	if err := os.Rename(out, backup); err != nil {
		return fmt.Errorf("preserve existing build: %w", err)
	}
	if err := os.Rename(stage, out); err != nil {
		restoreErr := os.Rename(backup, out)
		if restoreErr != nil {
			return fmt.Errorf("publish: %w; restore failed: %v; previous build retained at %s", err, restoreErr, backup)
		}
		return fmt.Errorf("publish: %w", err)
	}
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("remove build backup %s: %w", backup, err)
	}
	return nil
}
