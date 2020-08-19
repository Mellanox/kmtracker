package main

import (
	"fmt"
)

type BstTree struct {
	Root *Symbol
}

func (current *Symbol) BstNodeInsert(node *Symbol) {

	if current == nil {
		return
	}

	switch {
	case node.StartAddress == current.StartAddress:
		return
	case node.StartAddress < current.StartAddress:
		if current.Left == nil {
			current.Left = node
		} else {
			current.Left.BstNodeInsert(node)
		}
	case node.StartAddress > current.StartAddress:
		if current.Right == nil {
			current.Right = node
		} else {
			current.Right.BstNodeInsert(node)
		}
	}
}

func (t *BstTree) BstNodeInsert(node *Symbol) {
	if t.Root == nil {
		t.Root = node
	} else {
		t.Root.BstNodeInsert(node)
	}
}

func (current *Symbol) BstNodePrint(Level uint64, Direction string) {
	fmt.Printf("Level %v dir=%v name=%v module=%v addr=%x len=%v\n",
		Level, Direction, current.Name, current.Module, current.StartAddress, current.Length)

	if current.Left != nil {
		current.Left.BstNodePrint(Level+1, "left")
	}
	if current.Right != nil {
		current.Right.BstNodePrint(Level+1, "right")
	}
}

func (t *BstTree) BstTreePrint() {
	var Level uint64

	if t.Root == nil {
		return
	}
	Level = 0
	t.Root.BstNodePrint(Level, "root")
}

func (t *BstTree) SymbolFind(Address uint64) (node *Symbol) {
	if t.Root == nil {
		return nil
	}

	for current := t.Root; current != nil; {
		if Address == current.StartAddress {
			return current
		} else if Address < current.StartAddress {
			current = current.Left
			continue
		} else if Address > current.StartAddress && Address < current.StartAddress+current.Length-1 {
			return current
		} else {
			current = current.Right
			continue
		}
	}
	return nil
}
