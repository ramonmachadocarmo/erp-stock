package domain

import "fmt"

func FormatSKU(n int64) string {
	return fmt.Sprintf("%06d", n)
}

func BuildCategoryTree(items []Category) []Category {
	byID := make(map[string]*Category, len(items))
	nodes := make([]Category, len(items))
	copy(nodes, items)
	childIDs := make(map[string][]string, len(items))
	var roots []string
	for i := range nodes {
		nodes[i].Children = []Category{}
		byID[nodes[i].ID] = &nodes[i]
	}
	for i := range nodes {
		p := nodes[i].ParentID
		if p == "" || byID[p] == nil {
			roots = append(roots, nodes[i].ID)
			continue
		}
		childIDs[p] = append(childIDs[p], nodes[i].ID)
	}
	var attach func(id string) Category
	attach = func(id string) Category {
		n := *byID[id]
		n.Children = []Category{}
		for _, cid := range childIDs[id] {
			n.Children = append(n.Children, attach(cid))
		}
		return n
	}
	out := make([]Category, 0, len(roots))
	for _, id := range roots {
		out = append(out, attach(id))
	}
	return out
}

func CategoryParentWouldCycle(items []Category, id, parentID string) bool {
	if parentID == "" {
		return false
	}
	if parentID == id {
		return true
	}
	byID := make(map[string]Category, len(items))
	for _, c := range items {
		byID[c.ID] = c
	}
	seen := map[string]bool{}
	for p := parentID; p != ""; {
		if p == id {
			return true
		}
		if seen[p] {
			return true
		}
		seen[p] = true
		next, ok := byID[p]
		if !ok {
			break
		}
		p = next.ParentID
	}
	return false
}
