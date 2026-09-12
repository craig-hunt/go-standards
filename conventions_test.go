// Package conventions_test enforces the standards a compiler and linter cannot:
// where literals may live, what packages may be named, and which way domain
// imports may point.
package conventions_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

type sourceFile struct {
	path string
	file *ast.File
	fset *token.FileSet
}

func parseModule(t *testing.T) []sourceFile {
	t.Helper()
	var files []sourceFile
	err := filepath.WalkDir(moduleRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && slices.Contains(skippedDirectories, entry.Name()) {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Ext(path) != goExtension {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		files = append(files, sourceFile{path: path, file: file, fset: fset})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func markLiterals(node ast.Node, allowed map[*ast.BasicLit]bool) {
	ast.Inspect(node, func(inner ast.Node) bool {
		if literal, ok := inner.(*ast.BasicLit); ok {
			allowed[literal] = true
		}
		return true
	})
}

// permittedLiterals collects the places a literal is itself the named value:
// constant and import declarations, struct tags, package-level variables in a
// constants file, and the names of subtests and table cases.
func permittedLiterals(source sourceFile) map[*ast.BasicLit]bool {
	allowed := map[*ast.BasicLit]bool{}
	base := filepath.Base(source.path)
	isTest := strings.HasSuffix(base, testFileSuffix)
	isConstants := base == constantsFile || base == constantsTestFile

	for _, declaration := range source.file.Decls {
		if general, ok := declaration.(*ast.GenDecl); ok && general.Tok == token.VAR && isConstants {
			markLiterals(general, allowed)
		}
	}

	ast.Inspect(source.file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.GenDecl:
			if typed.Tok == token.CONST || typed.Tok == token.IMPORT {
				markLiterals(typed, allowed)
			}
		case *ast.Field:
			if typed.Tag != nil {
				allowed[typed.Tag] = true
			}
		case *ast.KeyValueExpr:
			if key, ok := typed.Key.(*ast.Ident); ok && isTest && key.Name == tableNameKey {
				markLiterals(typed.Value, allowed)
			}
		case *ast.CallExpr:
			if selector, ok := typed.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == subtestMethod && len(typed.Args) == subtestArguments {
				markLiterals(typed.Args[0], allowed)
			}
		}
		return true
	})
	return allowed
}

func trivial(literal *ast.BasicLit) bool {
	switch literal.Kind {
	case token.STRING:
		return literal.Value == emptyString
	case token.INT:
		return literal.Value == literalZero || literal.Value == literalOne
	default:
		return false
	}
}

func TestLiteralsLiveInNamedConstants(t *testing.T) {
	for _, source := range parseModule(t) {
		allowed := permittedLiterals(source)
		ast.Inspect(source.file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if ok && !allowed[literal] && !trivial(literal) {
				t.Errorf(literalFormat, source.fset.Position(literal.Pos()), literal.Value)
			}
			return true
		})
	}
}

func TestPackagesAreNamedForWhatTheyDo(t *testing.T) {
	for _, source := range parseModule(t) {
		if slices.Contains(grabBagNames, source.file.Name.Name) {
			t.Errorf(grabBagFormat, source.path, source.file.Name.Name)
		}
	}
}

func TestDomainPackagesDoNotDependOnInfrastructure(t *testing.T) {
	for _, source := range parseModule(t) {
		inDomain := slices.ContainsFunc(domainPackages, func(name string) bool {
			return strings.HasPrefix(filepath.ToSlash(source.path), fmt.Sprintf(domainRootFormat, name))
		})
		if !inDomain {
			continue
		}
		for _, spec := range source.file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range infrastructure {
				if path == forbidden || strings.HasPrefix(path, forbidden+importSeparator) {
					t.Errorf(importFormat, source.path, path)
				}
			}
		}
	}
}
