package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

type FileTreeNode struct {
	fileInfo  os.DirEntry
	fi        os.FileInfo
	ChildNode []FileTreeNode
}

func (node *FileTreeNode) size() (string, error) {
	if size := node.fi.Size(); size > 0 {
		s := int(size)
		ans := " (" + strconv.Itoa(s) + "b)"
		return ans, nil
	}
	return " (empty)", nil
}

func fillTree(path string, printFiles bool) ([]FileTreeNode, error) {
	//fmt.Println(path)
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var nodes []FileTreeNode
	for _, file := range files {
		if !printFiles && !file.IsDir() {
			continue
		}

		fi_, _ := file.Info()
		currNode := FileTreeNode{fileInfo: file, fi: fi_}

		if file.IsDir() {
			tree, err := fillTree(path+string(os.PathSeparator)+file.Name(), printFiles)
			if err != nil {
				return nil, err
			}
			currNode.ChildNode = tree
		}
		nodes = append(nodes, currNode)
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

		_, err := fmt.Fprint(out, output, prefix, node.fileInfo.Name())
		if err != nil {
			return
		}

		if !node.fileInfo.IsDir() {
			size, _ := node.size()
			fmt.Fprint(out, size, "\n")
		}

		if node.fileInfo.IsDir() {
			fmt.Fprint(out, "\n")
			printTree(out, node.ChildNode, output+suffix)
			/*size, err_ := node.size()
			if err_ != nil {
				return
			}
			fmt.Fprint(out, " (", size, "b)", "\n")*/
		}
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
