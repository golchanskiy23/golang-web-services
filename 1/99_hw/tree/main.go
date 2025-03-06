package main

import (
	"fmt"
	"io"
	"os"
)

type FileTreeNode struct {
	fileInfo  os.DirEntry
	ChildNode []FileTreeNode
}

func fillTree(path string, printFiles bool) ([]FileTreeNode, error) {
	//fmt.Println(path)
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var nodes []FileTreeNode
	for _, file := range files {

		if file.IsDir() {
			currNode := FileTreeNode{fileInfo: file}
			tree, err := fillTree(path+string(os.PathSeparator)+file.Name(), printFiles)
			if err != nil {
				return nil, err
			}

			currNode.ChildNode = tree
			nodes = append(nodes, currNode)
		}
	}
	return nodes, nil
}

func printTree(out io.Writer, tree []FileTreeNode, output string) {
	var (
		prefix  = "├───"
		suffix  = "│\t"
		lastIdx = len(tree) - 1
	)
	for i, node := range tree {
		if i == lastIdx {
			prefix = "└───"
			suffix = "\t"
		}

		_, err := fmt.Fprint(out, output, prefix, node.fileInfo.Name(), "\n")
		if err != nil {
			return
		}

		printTree(out, node.ChildNode, output+suffix)
	}
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	tree, err_ := fillTree(path, printFiles)
	if err_ != nil {
		return err_
	}
	printTree(out, tree, "")
	return nil
}

func main() {
	// определяем структуру дерева
	// заполняем массивы в узлах другими узлами
	// выводим узлы

	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	//tree, _ := fillTree("testdata", false)
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
