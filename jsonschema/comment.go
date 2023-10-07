package jsonschema

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

//nolint:gochecknoglobals
var commentMap map[string]string = make(map[string]string)

func getDescription(path string, typ string, field string) (string, bool) {
	key := fmt.Sprintf("%s.%s", path, typ)
	if field != "" {
		key += "." + field
	}

	if v, ok := commentMap[key]; ok {
		return v, true
	}

	return "", false
}

func getNodeDescription(doc *ast.CommentGroup, comment *ast.CommentGroup) string {
	if doc != nil && doc.Text() != "" {
		return doc.Text()
	}

	if comment != nil && comment.Text() != "" {
		return comment.Text()
	}

	return ""
}

// This can only retrieve comments up to one level deep.
func InitCommentMap(rootdir string, basePkg string) { //nolint:gocognit,cyclop
	fset := token.NewFileSet()
	dict := make(map[string][]*ast.Package)

	_ = filepath.Walk(rootdir, func(newPath string, info fs.FileInfo, _ error) error {
		if !info.IsDir() || strings.Contains(newPath, ".git") {
			return nil
		}
		d, _ := parser.ParseDir(fset, newPath, nil, parser.ParseComments)
		relPath, _ := filepath.Rel(rootdir, newPath)
		pkgName := filepath.Join(basePkg, relPath)
		for _, v := range d {
			dict[pkgName] = append(dict[pkgName], v)
		}

		return nil
	})

	var docPackage doc.Package

	for pkg, p := range dict {
		for _, f := range p {
			gtxt := ""
			typ := ""

			ast.Inspect(f, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.TypeSpec:
					if node.Name.IsExported() {
						typ = node.Name.String()
						txt := getNodeDescription(node.Doc, node.Comment)
						if txt == "" && gtxt != "" {
							txt = gtxt
							gtxt = ""
						}
						txt = docPackage.Synopsis(txt)
						if strings.TrimSpace(txt) != "" {
							commentMap[fmt.Sprintf("%s.%s", pkg, typ)] = strings.TrimSpace(txt)
						}
					}
				case *ast.Field:
					if txt := getNodeDescription(node.Doc, node.Comment); typ != "" && strings.TrimSpace(txt) != "" {
						for _, n := range node.Names {
							if n.IsExported() {
								commentMap[fmt.Sprintf("%s.%s.%s", pkg, typ, n)] = strings.TrimSpace(txt)
							}
						}
					}
				case *ast.GenDecl:
					if node.Doc != nil {
						gtxt = node.Doc.Text()
					}
				}

				return true
			})
		}
	}
}
