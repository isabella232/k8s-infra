/*
* Copyright (c) Microsoft Corporation.
* Licensed under the MIT license.
 */

package astmodel

import (
	"bufio"
	"go/token"
	"io"
	"os"

	ast "github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"k8s.io/klog/v2"
)

// TestFileDefinition defines the content of a test file we're generating
type TestFileDefinition struct {
	// the package this file is in
	packageReference PackageReference
	// definitions containing test cases to include in this file
	definitions []TypeDefinition
	// other packages whose references may be needed for code generation
	generatedPackages map[PackageReference]*PackageDefinition
}

// NewFileDefinition creates a file definition containing test cases from the specified definitions
func NewTestFileDefinition(
	packageRef PackageReference,
	definitions []TypeDefinition,
	generatedPackages map[PackageReference]*PackageDefinition) *TestFileDefinition {

	// TODO: check that all definitions are from same package
	return &TestFileDefinition{packageRef, definitions, generatedPackages}
}

// SaveToWriter writes the file to the specifier io.Writer
func (file TestFileDefinition) SaveToWriter(dst io.Writer) error {
	content := file.AsAst()

	buf := bufio.NewWriter(dst)
	defer buf.Flush()

	err := decorator.Fprint(buf, content)
	return err
}

// SaveToFile writes this generated file to disk
func (file TestFileDefinition) SaveToFile(filePath string) error {

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer f.Close()

	return file.SaveToWriter(f)
}

// AsAst generates an array of declarations for the content of the file
func (file *TestFileDefinition) AsAst() *ast.File {

	// Create context from imports
	codeGenContext := NewCodeGenerationContext(file.packageReference, file.generateImports(), file.generatedPackages)

	// Emit all test cases:
	var testcases []ast.Decl
	for _, s := range file.definitions {
		definer, ok := s.Type().(TestCaseDefiner)
		if !ok {
			continue
		}

		for _, testcase := range definer.TestCases() {
			testcases = append(testcases, testcase.AsFuncs(s.name, codeGenContext)...)
		}
	}

	var decls []ast.Decl

	// Create import header if needed
	usedImports := codeGenContext.UsedPackageImports()
	if usedImports.Length() > 0 {
		decls = append(decls, &ast.GenDecl{Tok: token.IMPORT, Specs: file.generateImportSpecs(usedImports)})
	}

	decls = append(decls, testcases...)

	var header []string
	header = append(header, CodeGenerationComments...)
	header = append(header,
		"// Copyright (c) Microsoft Corporation.",
		"// Licensed under the MIT license.")

	packageName := file.packageReference.PackageName()

	result := &ast.File{
		Decs: ast.FileDecorations{
			NodeDecs: ast.NodeDecs{
				Start: header,
				After: ast.EmptyLine,
			},
		},
		Name:  ast.NewIdent(packageName),
		Decls: decls,
	}

	return result
}

// disambiguates any conflicts
func (file *TestFileDefinition) generateImports() *PackageImportSet {
	var requiredImports = NewPackageImportSet()

	for _, s := range file.definitions {
		definer, ok := s.Type().(TestCaseDefiner)
		if !ok {
			continue
		}

		for _, testCase := range definer.TestCases() {
			requiredImports.Merge(testCase.RequiredImports())
		}
	}

	// Don't need to import the current package
	selfImport := NewPackageImport(file.packageReference)
	requiredImports.Remove(selfImport)

	// Force local imports to have explicit names based on the service
	for _, imp := range requiredImports.AsSlice() {
		if IsLocalPackageReference(imp.packageReference) && !imp.HasExplicitName() {
			name := requiredImports.ServiceNameForImport(imp)
			requiredImports.AddImport(imp.WithName(name))
		}
	}

	// Resolve any conflicts and report any that couldn't be fixed up automatically
	err := requiredImports.ResolveConflicts()
	if err != nil {
		klog.Errorf("File %s: %v", file.packageReference, err)
	}

	return requiredImports
}

func (file *TestFileDefinition) generateImportSpecs(imports *PackageImportSet) []ast.Spec {
	var importSpecs []ast.Spec
	for _, requiredImport := range imports.AsSortedSlice() {
		importSpecs = append(importSpecs, requiredImport.AsImportSpec())
	}

	return importSpecs
}
