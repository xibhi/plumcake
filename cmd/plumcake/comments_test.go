package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestZeroCommentsInSource(t *testing.T) {
	fileSet := token.NewFileSet()
	rootDirectory := "../.."

	err := filepath.WalkDir(rootDirectory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		astFile, parseError := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
		if parseError != nil {
			t.Fatalf("failed parsing %s: %v", path, parseError)
		}

		for _, commentGroup := range astFile.Comments {
			for _, comment := range commentGroup.List {
				text := strings.TrimSpace(comment.Text)
				if strings.HasPrefix(text, "//go:build") || strings.HasPrefix(text, "//go:embed") {
					continue
				}
				t.Errorf("%s:%d: forbidden comment found: %s",
					path, fileSet.Position(comment.Pos()).Line, text)
			}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("failed walking directory: %v", err)
	}
}
