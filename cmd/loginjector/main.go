package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file.go>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Failed to parse file: %v", err)
	}

	modified := false
	astutil.Apply(node, func(cursor *astutil.Cursor) bool {
		if fn, ok := cursor.Node().(*ast.FuncDecl); ok {
			if hasLogComment(node, fn) {
				removeLogComment(node, fn)
				injectLogging(fn)
				modified = true
			}
		}
		return true
	}, nil)

	if !modified {
		fmt.Printf("No functions with //dd:log comment found in %s\n", filename)
		return
	}

	// Add required imports and init function
	addRequiredImports(fset, node)
	addLoggerInit(node)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		log.Fatalf("Failed to format code: %v", err)
	}

	outputFile := filename + ".generated"
	if err := os.WriteFile(outputFile, buf.Bytes(), 0644); err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	fmt.Printf("Generated %s with logging injected\n", outputFile)
}

// hasLogComment checks if the function has a //dd:log comment in its preceding comments
func hasLogComment(file *ast.File, fn *ast.FuncDecl) bool {
	fnPos := fn.Pos()
	for _, commentGroup := range file.Comments {
		// Only check comments that end before the function starts
		if commentGroup.End() < fnPos {
			for _, comment := range commentGroup.List {
				if strings.Contains(comment.Text, "dd:log") {
					return true
				}
			}
		}
	}
	return false
}

// removeLogComment removes the //dd:log comment from the file
func removeLogComment(file *ast.File, fn *ast.FuncDecl) {
	fnPos := fn.Pos()
	var newComments []*ast.CommentGroup

	for _, commentGroup := range file.Comments {
		if commentGroup.End() < fnPos {
			var newComments2 []*ast.Comment
			for _, comment := range commentGroup.List {
				if !strings.Contains(comment.Text, "dd:log") {
					newComments2 = append(newComments2, comment)
				}
			}
			if len(newComments2) > 0 {
				commentGroup.List = newComments2
				newComments = append(newComments, commentGroup)
			}
		} else {
			newComments = append(newComments, commentGroup)
		}
	}
	file.Comments = newComments
}

// injectLogging modifies the function to add logging statements
func injectLogging(fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}

	funcName := fn.Name.Name

	startDecl := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("start")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("time"),
					Sel: ast.NewIdent("Now"),
				},
			},
		},
	}

	entryArgs := []ast.Expr{
		&ast.BasicLit{Kind: token.STRING, Value: `"function entry"`},
		&ast.BasicLit{Kind: token.STRING, Value: `"func"`},
		&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf(`"%s"`, funcName)},
	}

	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			if param.Names != nil {
				for _, name := range param.Names {
					entryArgs = append(entryArgs,
						&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf(`"%s"`, name.Name)},
						ast.NewIdent(name.Name),
					)
				}
			}
		}
	}

	entryLog := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("slog"),
				Sel: ast.NewIdent("Info"),
			},
			Args: entryArgs,
		},
	}

	deferLog := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{
							X: &ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   ast.NewIdent("slog"),
									Sel: ast.NewIdent("Info"),
								},
								Args: []ast.Expr{
									&ast.BasicLit{Kind: token.STRING, Value: `"function exit"`},
									&ast.BasicLit{Kind: token.STRING, Value: `"func"`},
									&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf(`"%s"`, funcName)},
									&ast.BasicLit{Kind: token.STRING, Value: `"duration"`},
									&ast.CallExpr{
										Fun: &ast.SelectorExpr{
											X:   ast.NewIdent("time"),
											Sel: ast.NewIdent("Since"),
										},
										Args: []ast.Expr{ast.NewIdent("start")},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	newStmts := []ast.Stmt{startDecl, entryLog, deferLog}
	newStmts = append(newStmts, fn.Body.List...)
	fn.Body.List = newStmts
}

// addRequiredImports ensures the required imports are present
func addRequiredImports(fset *token.FileSet, file *ast.File) {
	astutil.AddNamedImport(fset, file, "slog", "log/slog")
	astutil.AddImport(fset, file, "os")
	astutil.AddImport(fset, file, "time")
}

// addLoggerInit adds an init function to configure the slog logger
func addLoggerInit(file *ast.File) {
	// Check if init function already exists
	initExists := false
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		if fn, ok := cursor.Node().(*ast.FuncDecl); ok && fn.Name.Name == "init" {
			initExists = true
			return false // Stop traversal early
		}
		return true
	}, nil)

	if initExists {
		return
	}

	// Create init function
	initFunc := createInitFunction()

	// Insert init function after imports
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		if genDecl, ok := cursor.Node().(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			cursor.InsertAfter(initFunc)
			return false // Stop traversal after first import declaration
		}
		return true
	}, nil)
}

// createInitFunction creates the init function AST node
func createInitFunction() *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent("init"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				// opts := &slog.HandlerOptions{Level: slog.LevelInfo}
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("opts")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.UnaryExpr{
							Op: token.AND,
							X: &ast.CompositeLit{
								Type: &ast.SelectorExpr{
									X:   ast.NewIdent("slog"),
									Sel: ast.NewIdent("HandlerOptions"),
								},
								Elts: []ast.Expr{
									&ast.KeyValueExpr{
										Key: ast.NewIdent("Level"),
										Value: &ast.SelectorExpr{
											X:   ast.NewIdent("slog"),
											Sel: ast.NewIdent("LevelInfo"),
										},
									},
								},
							},
						},
					},
				},
				// handler := slog.NewJSONHandler(os.Stdout, opts)
				&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("handler")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("slog"),
								Sel: ast.NewIdent("NewJSONHandler"),
							},
							Args: []ast.Expr{
								&ast.SelectorExpr{
									X:   ast.NewIdent("os"),
									Sel: ast.NewIdent("Stdout"),
								},
								ast.NewIdent("opts"),
							},
						},
					},
				},
				// slog.SetDefault(slog.New(handler))
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("slog"),
							Sel: ast.NewIdent("SetDefault"),
						},
						Args: []ast.Expr{
							&ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   ast.NewIdent("slog"),
									Sel: ast.NewIdent("New"),
								},
								Args: []ast.Expr{ast.NewIdent("handler")},
							},
						},
					},
				},
			},
		},
	}
}
