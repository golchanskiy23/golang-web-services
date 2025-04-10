package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
)

type Parser struct {
}

type CodeGenerator struct {
}

type ParsedFile struct {
	PackageName string
	ApiMethods  map[string]ApiMethod
	ApiStructs  map[string]ApiStruct
}

type ApiMethod struct {
}

type ApiStruct struct {
}

func main() {
	inFile, outFile := os.Args[1], os.Args[2]
	fmt.Println(len(os.Args))
	parser := NewParser()
	parsedInFile, err := parser.Parse(inFile)
	if err != nil {
		log.Fatalf("Error happened: %s\n", err)
	}
	out, err := os.Create(outFile)
	if err != nil {
		log.Fatalf("Error creating file: %s", err)
	}
	defer out.Close()

	codeGenerator := NewCodeGenerator(parsedInFile, out)
	codeGenerator.Generate()
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseFunc(file *ParsedFile, decl ast.Decl) {

}

func (p *Parser) ParseStruct(file *ParsedFile, tt *ast.StructType) {

}

func (p *Parser) Parse(inFile string) (*ParsedFile, error) {
	fs := token.NewFileSet()
	nodes, err := parser.ParseFile(fs, inFile, nil, parser.ParseComments)
	if err != nil {
		fmt.Errorf("parsing error: %s\n", err)
	}

	result := &ParsedFile{
		PackageName: nodes.Name.Name,
		ApiMethods:  make(map[string]ApiMethod),
		ApiStructs:  make(map[string]ApiStruct),
	}

	for _, decl := range nodes.Decls {
		switch decl.(type) {
		case *ast.FuncDecl:
			p.ParseFunc(result, decl)
		case *ast.GenDecl:
			for _, t := range decl.(*ast.GenDecl).Specs {
				if tt, ok := t.(*ast.TypeSpec); ok {
					if ttt, ok := tt.Type.(*ast.StructType); ok {
						p.ParseStruct(result, ttt)
					}
				}
			}
		}
	}
	return result, nil
}

func NewCodeGenerator(parsedFile *ParsedFile, out *os.File) *CodeGenerator {
	return nil
}

func (cg *CodeGenerator) Generate() {

}
