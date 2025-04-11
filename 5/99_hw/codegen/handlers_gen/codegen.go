package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

type Parser struct {
	ApiPrefix      string
	MatchValidator regexp.Regexp
}

type CodeGenerator struct {
	InputFile  *ParsedFile
	OutputFile *os.File
}

type ParsedFile struct {
	PackageName string
	ApiHandler  map[string]ApiHandler
	ApiStructs  map[string]ApiStruct
}

type ApiHandler struct {
	Handler    string
	ApiMethods []ApiMethod
}

type ApiMethod struct {
	Name        string
	HandlerName string
	RequestName string
	Api         ApiMetaInformation
}

type ApiMetaInformation struct {
	URL    string
	Auth   bool
	Method string
}

type ApiStruct struct {
	Name   string
	Fields []StructField
}

type StructField struct {
	Name            string
	Type            string
	StructValueTags structValueTag
}

type structValueTag struct {
	ParamName string
	Required  bool
	Min       bool
	MinValue  int
	Max       bool
	MaxValue  int
	Enum      []string
	Default   string
}

func main() {
	inFile, outFile := os.Args[1], os.Args[2]
	parser := NewParser("// apigen:api", "`apivalidator:\"(.*)\"`")
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

func NewParser(APIPrefix, APIValidator string) *Parser {
	return &Parser{
		ApiPrefix:      APIPrefix,
		MatchValidator: *regexp.MustCompile(APIValidator),
	}
}

func (p *Parser) GetFunctionReceiver(node *ast.FuncDecl) string {
	if node.Recv != nil {
		for _, field := range node.Recv.List {
			if f, ok := field.Type.(*ast.StarExpr); ok {
				if fi, ok := f.X.(*ast.Ident); ok {
					return fi.Name
				}
			}

			if fi, ok := field.Type.(*ast.Ident); ok {
				return fi.Name
			}

		}
	}
	return ""
}

func (p *Parser) ParseFunc(file *ParsedFile, decl *ast.FuncDecl) {
	if decl.Doc != nil {
		var meta ApiMetaInformation
		for _, comment := range decl.Doc.List {
			if strings.HasPrefix(comment.Text, p.ApiPrefix) {
				jsonStr := comment.Text[len(p.ApiPrefix):]
				if err := json.Unmarshal([]byte(jsonStr), &meta); err != nil {
					break
				}
			}
		}

		if meta.URL != "" {
			if receiver := p.GetFunctionReceiver(decl); receiver != "" {
				if _, exists := file.ApiHandler[receiver]; !exists {
					file.ApiHandler[receiver] = ApiHandler{
						Handler: receiver,
					}
				}

				if reqType, ok := decl.Type.Params.List[1].Type.(*ast.Ident); ok {
					handler := file.ApiHandler[receiver]
					handler.ApiMethods = append(handler.ApiMethods, ApiMethod{
						Name:        decl.Name.Name,
						HandlerName: receiver,
						RequestName: reqType.Name,
						Api:         meta,
					})
					file.ApiHandler[receiver] = handler
				}
			}
		}

	}
}

func (p *Parser) ParseStruct(file *ParsedFile, structName string, tt *ast.StructType) {
	for _, field := range tt.Fields.List {
		if field.Tag != nil {
			if matches := p.MatchValidator.FindStringSubmatch(field.Tag.Value); len(matches) > 0 {
				if _, exists := file.ApiStructs[structName]; !exists {
					file.ApiStructs[structName] = ApiStruct{
						Name: structName,
					}
				}

				fieldTag := structValueTag{
					ParamName: strings.ToLower(field.Names[0].Name),
				}

				structFieldTags := strings.Split(matches[1], ",")
				for _, structFieldTag := range structFieldTags {
					t := strings.Split(structFieldTag, "=")
					switch t[0] {
					case "required":
						fieldTag.Required = true
					case "min":
						fieldTag.Min = true
						fieldTag.MinValue, _ = strconv.Atoi(t[1])
					case "max":
						fieldTag.Max = true
						fieldTag.MaxValue, _ = strconv.Atoi(t[1])
					case "paramname":
						fieldTag.ParamName = t[1]
					case "enum":
						fieldTag.Enum = strings.Split(t[1], "|")
					case "default":
						fieldTag.Default = t[1]
					}
				}
				currStruct := file.ApiStructs[structName]
				currStruct.Fields = append(currStruct.Fields, StructField{
					Name:            field.Names[0].Name,
					Type:            field.Type.(*ast.Ident).Name,
					StructValueTags: fieldTag,
				})
				file.ApiStructs[structName] = currStruct

			}
		}
	}
}

func (p *Parser) Parse(inFile string) (*ParsedFile, error) {
	fs := token.NewFileSet()
	nodes, err := parser.ParseFile(fs, inFile, nil, parser.ParseComments)
	if err != nil {
		fmt.Errorf("parsing error: %s\n", err)
	}

	result := &ParsedFile{
		PackageName: nodes.Name.Name,
		ApiHandler:  make(map[string]ApiHandler),
		ApiStructs:  make(map[string]ApiStruct),
	}

	for _, decl := range nodes.Decls {
		switch decl.(type) {
		case *ast.FuncDecl:
			p.ParseFunc(result, decl.(*ast.FuncDecl))
		case *ast.GenDecl:
			for _, t := range decl.(*ast.GenDecl).Specs {
				if tt, ok := t.(*ast.TypeSpec); ok {
					if ttt, ok := tt.Type.(*ast.StructType); ok {
						p.ParseStruct(result, tt.Name.Name, ttt)
					}
				}
			}
		}
	}
	return result, nil
}

func NewCodeGenerator(parsedFile *ParsedFile, out *os.File) *CodeGenerator {
	return &CodeGenerator{
		InputFile:  parsedFile,
		OutputFile: out,
	}
}

func (cg *CodeGenerator) WriteHeader() {
	cg.OutputFile.WriteString("// Code generated by go generate; DO NOT EDIT\n")
	// fill package and imports
}

func (c *CodeGenerator) generateServe() *template.Template {
	return nil
}

func (c *CodeGenerator) generateStructValidation() *template.Template {
	return nil
}

func (c *CodeGenerator) generateWrapper() *template.Template {
	return nil
}

func (c *CodeGenerator) Generate() {
	c.WriteHeader()
	ServeTmpl, ValidationTmpl, WrapperTmpl := c.generateServe(), c.generateStructValidation(), c.generateWrapper()
	for _, handler := range c.InputFile.ApiHandler {
		ServeTmpl.Execute(c.OutputFile, handler)
		for _, method := range handler.ApiMethods {
			WrapperTmpl.Execute(c.OutputFile, method)
		}
	}

	for _, apiStruct := range c.InputFile.ApiStructs {
		ValidationTmpl.Execute(c.OutputFile, apiStruct)
	}
}
