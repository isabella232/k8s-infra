/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 */

package astbuilder

import (
	ast "github.com/dave/dst"
)

// CallFunc creates an expression to call a function with specified arguments, generating code
// like:
// <funcName>(<arguments>...)
func CallFunc(funcName string, arguments ...ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun:  ast.NewIdent(funcName),
		Args: arguments,
	}
}

// CallQualifiedFunc creates an expression to call a qualified function with the specified
// arguments, generating code like:
// <qualifier>.<funcName>(arguments...)
func CallQualifiedFunc(qualifier string, funcName string, arguments ...ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent(qualifier),
			Sel: ast.NewIdent(funcName),
		},
		Args: arguments,
	}
}

// InvokeFunc creates a statement to invoke a function with specified arguments, generating code
// like
// <funcName>(arguments...)
// If you want to use the result of the function call as a value, use CallFunc() instead
func InvokeFunc(funcName string, arguments ...ast.Expr) ast.Stmt {
	return &ast.ExprStmt{
		X: CallFunc(funcName, arguments...),
	}
}

// InvokeQualifiedFunc creates a statement to invoke a qualified function with specified
// arguments, generating code like:
// <qualifier>.<funcName>(arguments...)
// If you want to use the result of the function call as a value, use CallQualifiedFunc() instead
func InvokeQualifiedFunc(qualifier string, funcName string, arguments ...ast.Expr) ast.Stmt {
	return &ast.ExprStmt{
		X: CallQualifiedFunc(qualifier, funcName, arguments...),
	}
}
