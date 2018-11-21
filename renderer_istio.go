package gendoc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

var (
	api = map[string]bool{
		"Gateway":        true,
		"ServiceEntry":   true,
		"VirtualService": true,
	}
	extensions = parser.CommonExtensions | parser.AutoHeadingIDs
)

type istioRenderer struct{}

func (r *istioRenderer) Apply(template *Template) ([]byte, error) {
	out := make(map[string]interface{}, 1024)

	for _, file := range template.Files {
		s := strings.Split(file.Package, ".")
		if len(s) != 3 {
			continue
		}

		locator := []byte(fmt.Sprintf("# See more info at https://istio.io/docs/reference/config/%s", file.Package))

		m := make(map[string]string, 1024)
		messages := ([]*Message)(file.Messages)
		for _, message := range messages {
			name := message.Name
			if _, ok := api[name]; !ok {
				continue
			}

			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintln(os.Stderr, string(message.Description))
				}
			}()

			visitor := newIstioVisitor()
			p := parser.NewWithExtensions(extensions)
			ast.Walk(p.Parse([]byte(message.Description)), visitor)

			m["scope"] = "yaml"
			m["prefix"] = strings.ToLower(s[0] + message.Name)

			// body := bytes.Replace(visitor.description, []byte("\n"), []byte("\n# "), -1)
			// body = append([]byte("#"), body...)
			// body = append(body, []byte("\n")...)
			body := make([]byte, 0, 1024)
			body = append(body, locator...)
			body = append(body, []byte("\n")...)
			body = append(body, visitor.codeblock...)

			index := bytes.Index(visitor.description, []byte("."))
			if index != -1 {
				visitor.description = visitor.description[:index]
			}

			m["description"] = string(visitor.description)
			m["body"] = string(body)

			out[m["prefix"]] = m
		}
	}

	return json.MarshalIndent(out, "", "\t")
}

type vsCodeVisitor struct {
	description []byte
	codeblock   []byte
}

func newIstioVisitor() *vsCodeVisitor {
	return &vsCodeVisitor{
		description: make([]byte, 0, 1024),
		codeblock:   make([]byte, 0, 1024),
	}
}

func (v *vsCodeVisitor) Visit(node ast.Node, entering bool) ast.WalkStatus {
	// fmt.Println(getType(node))
	// fmt.Println(node.)
	switch node.(type) {
	case *ast.Text:
		if len(v.description) == 0 {
			v.description = node.AsLeaf().Literal
		}
	case *ast.CodeBlock:
		if len(v.codeblock) == 0 {
			v.codeblock = node.AsLeaf().Literal
		}
	}

	return ast.GoToNext
}
