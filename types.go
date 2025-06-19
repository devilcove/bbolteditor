package main

import (
	"log"
	"os"
	"strings"

	"cogentcore.org/core/core"
)

type Path [][]byte

type TreeNode struct {
	Name     []byte
	IsBucket bool
	Value    []byte
	Path     Path
	Children []*TreeNode
}

func getNode(m *core.Scene) TreeNode {
	name := strings.ReplaceAll(m.This.AsTree().Name, "-menu", "")
	node, ok := nodeMap[name]
	if !ok {
		logger := log.New(os.Stdout, "", log.Flags())
		_ = logger.Output(2, "invalid node "+name)
		node = selectedNode
	}
	return node
}

func pathToString(path Path) string {
	array := []string{}
	for _, part := range path {
		array = append(array, string(part))
	}
	return strings.Join(array, "/")
}

func stringToPath(s string) Path {
	path := Path{}
	array := strings.Split(s, "/")
	for _, part := range array {
		path = append(path, []byte(part))
	}
	return path
}
