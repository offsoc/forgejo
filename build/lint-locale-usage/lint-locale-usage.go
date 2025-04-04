// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"go/ast"
	goParser "go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	tmplParser "text/template/parse"

	"forgejo.org/modules/container"
	fjTemplates "forgejo.org/modules/templates"
	"forgejo.org/modules/translation/localeiter"
	"forgejo.org/modules/util"
)

// this works by first gathering all valid source string IDs from `en-US` reference files
// and then checking if all used source strings are actually defined

type LocatedError struct {
	Location string
	Kind     string
	Err      error
}

func (e LocatedError) Error() string {
	var sb strings.Builder

	sb.WriteString(e.Location)
	sb.WriteString(":\t")
	if e.Kind != "" {
		sb.WriteString(e.Kind)
		sb.WriteString(": ")
	}
	sb.WriteString("ERROR: ")
	sb.WriteString(e.Err.Error())

	return sb.String()
}

func InitLocaleTrFunctions() map[string][]uint {
	ret := make(map[string][]uint)

	f0 := []uint{0}
	ret["Tr"] = f0
	ret["TrString"] = f0
	ret["TrHTML"] = f0

	ret["TrPluralString"] = []uint{1}
	ret["TrN"] = []uint{1, 2}

	return ret
}

type Handler struct {
	OnMsgid            func(fset *token.FileSet, pos token.Pos, msgid string)
	OnUnexpectedInvoke func(fset *token.FileSet, pos token.Pos, funcname string, argc int)
	LocaleTrFunctions  map[string][]uint
}

// the `Handle*File` functions follow the following calling convention:
// * `fname` is the name of the input file
// * `src` is either `nil` (then the function invokes `ReadFile` to read the file)
//   or the contents of the file as {`[]byte`, or a `string`}

func (handler Handler) HandleGoFile(fname string, src any) error {
	fset := token.NewFileSet()
	node, err := goParser.ParseFile(fset, fname, src, goParser.SkipObjectResolution)
	if err != nil {
		return LocatedError{
			Location: fname,
			Kind:     "Go parser",
			Err:      err,
		}
	}

	ast.Inspect(node, func(n ast.Node) bool {
		// search for function calls of the form `anything.Tr(any-string-lit, ...)`

		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 1 {
			return true
		}

		funSel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ltf, ok := handler.LocaleTrFunctions[funSel.Sel.Name]
		if !ok {
			return true
		}

		var gotUnexpectedInvoke *int

		for _, argNum := range ltf {
			if len(call.Args) >= int(argNum+1) {
				argLit, ok := call.Args[int(argNum)].(*ast.BasicLit)
				if !ok || argLit.Kind != token.STRING {
					continue
				}

				// extract string content
				arg, err := strconv.Unquote(argLit.Value)
				if err == nil {
					// found interesting strings
					handler.OnMsgid(fset, argLit.ValuePos, arg)
				}
			} else {
				argc := len(call.Args)
				gotUnexpectedInvoke = &argc
			}
		}

		if gotUnexpectedInvoke != nil {
			handler.OnUnexpectedInvoke(fset, funSel.Sel.NamePos, funSel.Sel.Name, *gotUnexpectedInvoke)
		}

		return true
	})

	return nil
}

// derived from source: modules/templates/scopedtmpl/scopedtmpl.go, L169-L213
func (handler Handler) handleTemplateNode(fset *token.FileSet, node tmplParser.Node) {
	switch node.Type() {
	case tmplParser.NodeAction:
		handler.handleTemplatePipeNode(fset, node.(*tmplParser.ActionNode).Pipe)
	case tmplParser.NodeList:
		nodeList := node.(*tmplParser.ListNode)
		handler.handleTemplateFileNodes(fset, nodeList.Nodes)
	case tmplParser.NodePipe:
		handler.handleTemplatePipeNode(fset, node.(*tmplParser.PipeNode))
	case tmplParser.NodeTemplate:
		handler.handleTemplatePipeNode(fset, node.(*tmplParser.TemplateNode).Pipe)
	case tmplParser.NodeIf:
		nodeIf := node.(*tmplParser.IfNode)
		handler.handleTemplateBranchNode(fset, nodeIf.BranchNode)
	case tmplParser.NodeRange:
		nodeRange := node.(*tmplParser.RangeNode)
		handler.handleTemplateBranchNode(fset, nodeRange.BranchNode)
	case tmplParser.NodeWith:
		nodeWith := node.(*tmplParser.WithNode)
		handler.handleTemplateBranchNode(fset, nodeWith.BranchNode)

	case tmplParser.NodeCommand:
		nodeCommand := node.(*tmplParser.CommandNode)

		handler.handleTemplateFileNodes(fset, nodeCommand.Args)

		if len(nodeCommand.Args) < 2 {
			return
		}

		nodeChain, ok := nodeCommand.Args[0].(*tmplParser.ChainNode)
		if !ok {
			return
		}

		nodeIdent, ok := nodeChain.Node.(*tmplParser.IdentifierNode)
		if !ok || nodeIdent.Ident != "ctx" || len(nodeChain.Field) != 2 || nodeChain.Field[0] != "Locale" {
			return
		}

		ltf, ok := handler.LocaleTrFunctions[nodeChain.Field[1]]
		if !ok {
			return
		}

		var gotUnexpectedInvoke *int

		for _, argNum := range ltf {
			if len(nodeCommand.Args) >= int(argNum+2) {
				nodeString, ok := nodeCommand.Args[int(argNum+1)].(*tmplParser.StringNode)
				if ok {
					// found interesting strings
					// the column numbers are a bit "off", but much better than nothing
					handler.OnMsgid(fset, token.Pos(nodeString.Pos), nodeString.Text)
				}
			} else {
				argc := len(nodeCommand.Args) - 1
				gotUnexpectedInvoke = &argc
			}
		}

		if gotUnexpectedInvoke != nil {
			handler.OnUnexpectedInvoke(fset, token.Pos(nodeChain.Pos), nodeChain.Field[1], *gotUnexpectedInvoke)
		}

	default:
	}
}

func (handler Handler) handleTemplatePipeNode(fset *token.FileSet, pipeNode *tmplParser.PipeNode) {
	if pipeNode == nil {
		return
	}

	// NOTE: we can't pass `pipeNode.Cmds` to handleTemplateFileNodes due to incompatible argument types
	for _, node := range pipeNode.Cmds {
		handler.handleTemplateNode(fset, node)
	}
}

func (handler Handler) handleTemplateBranchNode(fset *token.FileSet, branchNode tmplParser.BranchNode) {
	handler.handleTemplatePipeNode(fset, branchNode.Pipe)
	handler.handleTemplateFileNodes(fset, branchNode.List.Nodes)
	if branchNode.ElseList != nil {
		handler.handleTemplateFileNodes(fset, branchNode.ElseList.Nodes)
	}
}

func (handler Handler) handleTemplateFileNodes(fset *token.FileSet, nodes []tmplParser.Node) {
	for _, node := range nodes {
		handler.handleTemplateNode(fset, node)
	}
}

func (handler Handler) HandleTemplateFile(fname string, src any) error {
	var tmplContent []byte
	switch src2 := src.(type) {
	case nil:
		var err error
		tmplContent, err = os.ReadFile(fname)
		if err != nil {
			return LocatedError{
				Location: fname,
				Kind:     "ReadFile",
				Err:      err,
			}
		}
	case []byte:
		tmplContent = src2
	case string:
		// SAFETY: we do not modify tmplContent below
		tmplContent = util.UnsafeStringToBytes(src2)
	default:
		panic("invalid type for 'src'")
	}

	fset := token.NewFileSet()
	fset.AddFile(fname, 1, len(tmplContent)).SetLinesForContent(tmplContent)
	// SAFETY: we do not modify tmplContent2 below
	tmplContent2 := util.UnsafeBytesToString(tmplContent)

	tmpl := template.New(fname)
	tmpl.Funcs(fjTemplates.NewFuncMap())
	tmplParsed, err := tmpl.Parse(tmplContent2)
	if err != nil {
		return LocatedError{
			Location: fname,
			Kind:     "Template parser",
			Err:      err,
		}
	}
	handler.handleTemplateFileNodes(fset, tmplParsed.Root.Nodes)
	return nil
}

// This command assumes that we get started from the project root directory
//
// Possible command line flags:
//
//	--allow-missing-msgids        don't return an error code if missing message IDs are found
//
// EXIT CODES:
//
//	0  success, no issues found
//	1  unable to walk directory tree
//	2  unable to parse locale ini/json files
//	3  unable to parse go or text/template files
//	4  found missing message IDs
//
//nolint:forbidigo
func main() {
	allowMissingMsgids := false
	for _, arg := range os.Args[1:] {
		if arg == "--allow-missing-msgids" {
			allowMissingMsgids = true
		}
	}

	onError := func(err error) {
		if err == nil {
			return
		}
		fmt.Println(err.Error())
		os.Exit(3)
	}

	msgids := make(container.Set[string])

	localeFile := filepath.Join(filepath.Join("options", "locale"), "locale_en-US.ini")
	localeContent, err := os.ReadFile(localeFile)
	if err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	if err = localeiter.IterateMessagesContent(localeContent, func(trKey, trValue string) error {
		msgids[trKey] = struct{}{}
		return nil
	}); err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	localeFile = filepath.Join(filepath.Join("options", "locale_next"), "locale_en-US.json")
	localeContent, err = os.ReadFile(localeFile)
	if err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	if err := localeiter.IterateMessagesNextContent(localeContent, func(trKey, pluralForm, trValue string) error {
		// ignore plural form
		msgids[trKey] = struct{}{}
		return nil
	}); err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	gotAnyMsgidError := false

	handler := Handler{
		OnMsgid: func(fset *token.FileSet, pos token.Pos, msgid string) {
			if !msgids.Contains(msgid) {
				gotAnyMsgidError = true
				fmt.Printf("%s:\tmissing msgid: %s\n", fset.Position(pos).String(), msgid)
			}
		},
		OnUnexpectedInvoke: func(fset *token.FileSet, pos token.Pos, funcname string, argc int) {
			gotAnyMsgidError = true
			fmt.Printf("%s:\tunexpected invocation of %s with %d arguments\n", fset.Position(pos).String(), funcname, argc)
		},
		LocaleTrFunctions: InitLocaleTrFunctions(),
	}

	if err := filepath.WalkDir(".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if name == "docker" || name == ".git" || name == "node_modules" {
				return fs.SkipDir
			}
		} else if name == "bindata.go" || fpath == "modules/translation/i18n/i18n_test.go" {
			// skip false positives
		} else if strings.HasSuffix(name, ".go") {
			onError(handler.HandleGoFile(fpath, nil))
		} else if strings.HasSuffix(name, ".tmpl") {
			if strings.HasPrefix(fpath, "tests") && strings.HasSuffix(name, ".ini.tmpl") {
				// skip false positives
			} else {
				onError(handler.HandleTemplateFile(fpath, nil))
			}
		}
		return nil
	}); err != nil {
		fmt.Printf("walkdir ERROR: %s\n", err.Error())
		os.Exit(1)
	}

	if !allowMissingMsgids && gotAnyMsgidError {
		os.Exit(4)
	}
}
