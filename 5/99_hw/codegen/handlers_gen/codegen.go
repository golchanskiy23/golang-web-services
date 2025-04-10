package main

import (
	"fmt"
	"log"
	"os"
)

type Parser struct {
}

type CodeGenerator struct {
}

type ParsedFile struct {
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
	return nil
}

func (p *Parser) Parse(inFile string) (*ParsedFile, error) {

	//fmt.Print(parse)
	return nil, nil
}

func NewCodeGenerator(parsedFile *ParsedFile, out *os.File) *CodeGenerator {
	return nil
}

func (cg *CodeGenerator) Generate() {

}
