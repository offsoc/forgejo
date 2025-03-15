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

	"code.gitea.io/gitea/modules/container"
	"code.gitea.io/gitea/modules/locale"
	fjTemplates "code.gitea.io/gitea/modules/templates"
	"code.gitea.io/gitea/modules/util"
)

// this works by first gathering all valid source string IDs from `en-US` reference files
// and then checking if all used source strings are actually defined

type OnMsgidHandler func(fset *token.FileSet, pos token.Pos, msgid string)

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

func isLocaleTrFunction(funcname string) bool {
	return funcname == "Tr" || funcname == "TrN"
}

// the `Handle*File` functions follow the following calling convention:
// * `fname` is the name of the input file
// * `src` is either `nil` (then the function invokes `ReadFile` to read the file)
//   or the contents of the file as {`[]byte`, or a `string`}

func (omh OnMsgidHandler) HandleGoFile(fname string, src any) error {
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
		// search for function calls of the form `anything.Tr(any-string-lit)`

		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}

		funSel, ok := call.Fun.(*ast.SelectorExpr)
		if (!ok) || !isLocaleTrFunction(funSel.Sel.Name) {
			return true
		}

		argLit, ok := call.Args[0].(*ast.BasicLit)
		if (!ok) || argLit.Kind != token.STRING {
			return true
		}

		// extract string content
		arg, err := strconv.Unquote(argLit.Value)
		if err != nil {
			return true
		}

		// found interesting string
		omh(fset, argLit.ValuePos, arg)

		return true
	})

	return nil
}

// derived from source: modules/templates/scopedtmpl/scopedtmpl.go, L169-L213
func (omh OnMsgidHandler) handleTemplateNode(fset *token.FileSet, node tmplParser.Node) {
	switch node.Type() {
	case tmplParser.NodeAction:
		omh.handleTemplatePipeNode(fset, node.(*tmplParser.ActionNode).Pipe)
	case tmplParser.NodeList:
		nodeList := node.(*tmplParser.ListNode)
		omh.handleTemplateFileNodes(fset, nodeList.Nodes)
	case tmplParser.NodePipe:
		omh.handleTemplatePipeNode(fset, node.(*tmplParser.PipeNode))
	case tmplParser.NodeTemplate:
		omh.handleTemplatePipeNode(fset, node.(*tmplParser.TemplateNode).Pipe)
	case tmplParser.NodeIf:
		nodeIf := node.(*tmplParser.IfNode)
		omh.handleTemplateBranchNode(fset, nodeIf.BranchNode)
	case tmplParser.NodeRange:
		nodeRange := node.(*tmplParser.RangeNode)
		omh.handleTemplateBranchNode(fset, nodeRange.BranchNode)
	case tmplParser.NodeWith:
		nodeWith := node.(*tmplParser.WithNode)
		omh.handleTemplateBranchNode(fset, nodeWith.BranchNode)

	case tmplParser.NodeCommand:
		nodeCommand := node.(*tmplParser.CommandNode)

		omh.handleTemplateFileNodes(fset, nodeCommand.Args)

		if len(nodeCommand.Args) != 2 {
			return
		}

		nodeChain, ok := nodeCommand.Args[0].(*tmplParser.ChainNode)
		if !ok {
			return
		}

		nodeString, ok := nodeCommand.Args[1].(*tmplParser.StringNode)
		if !ok {
			return
		}

		nodeIdent, ok := nodeChain.Node.(*tmplParser.IdentifierNode)
		if !ok || nodeIdent.Ident != "ctx" {
			return
		}

		if len(nodeChain.Field) != 2 || nodeChain.Field[0] != "Locale" || !isLocaleTrFunction(nodeChain.Field[1]) {
			return
		}

		// found interesting string
		// the column numbers are a bit "off", but much better than nothing
		omh(fset, token.Pos(nodeString.Pos), nodeString.Text)

	default:
	}
}

func (omh OnMsgidHandler) handleTemplatePipeNode(fset *token.FileSet, pipeNode *tmplParser.PipeNode) {
	if pipeNode == nil {
		return
	}

	// NOTE: we can't pass `pipeNode.Cmds` to handleTemplateFileNodes due to incompatible argument types
	for _, node := range pipeNode.Cmds {
		omh.handleTemplateNode(fset, node)
	}
}

func (omh OnMsgidHandler) handleTemplateBranchNode(fset *token.FileSet, branchNode tmplParser.BranchNode) {
	omh.handleTemplatePipeNode(fset, branchNode.Pipe)
	omh.handleTemplateFileNodes(fset, branchNode.List.Nodes)
	if branchNode.ElseList != nil {
		omh.handleTemplateFileNodes(fset, branchNode.ElseList.Nodes)
	}
}

func (omh OnMsgidHandler) handleTemplateFileNodes(fset *token.FileSet, nodes []tmplParser.Node) {
	for _, node := range nodes {
		omh.handleTemplateNode(fset, node)
	}
}

func (omh OnMsgidHandler) HandleTemplateFile(fname string, src any) error {
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
	omh.handleTemplateFileNodes(fset, tmplParsed.Tree.Root.Nodes)
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
	onMsgid := func(trKey, trValue string) error {
		msgids[trKey] = struct{}{}
		return nil
	}

	localeFile := filepath.Join(filepath.Join("options", "locale"), "locale_en-US.ini")
	localeContent, err := os.ReadFile(localeFile)
	if err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	if err = locale.IterateMessagesContent(localeContent, onMsgid); err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	localeFile = filepath.Join(filepath.Join("options", "locale_next"), "locale_en-US.json")
	localeContent, err = os.ReadFile(localeFile)
	if err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	if err := locale.IterateMessagesNextContent(localeContent, onMsgid); err != nil {
		fmt.Printf("%s:\tERROR: %s\n", localeFile, err.Error())
		os.Exit(2)
	}

	gotAnyMsgidError := false

	omh := OnMsgidHandler(func(fset *token.FileSet, pos token.Pos, msgid string) {
		if !msgids.Contains(msgid) {
			gotAnyMsgidError = true
			fmt.Printf("%s:\tmissing msgid: %s\n", fset.Position(pos).String(), msgid)
		}
	})

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
		} else if name == "bindata.go" {
			// skip false positives
		} else if strings.HasSuffix(name, ".go") {
			onError(omh.HandleGoFile(fpath, nil))
		} else if strings.HasSuffix(name, ".tmpl") {
			if strings.HasPrefix(fpath, "tests") && strings.HasSuffix(name, ".ini.tmpl") {
				// skip false positives
			} else {
				onError(omh.HandleTemplateFile(fpath, nil))
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
