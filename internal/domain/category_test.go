package domain

import "testing"

func TestFormatSKU(t *testing.T) {
	if got := FormatSKU(1); got != "000001" {
		t.Fatalf("got %s", got)
	}
	if got := FormatSKU(42); got != "000042" {
		t.Fatalf("got %s", got)
	}
}

func TestBuildCategoryTree(t *testing.T) {
	items := []Category{
		{ID: "folhas", ParentID: "legumes", Name: "Folhas"},
		{ID: "legumes", ParentID: "", Name: "Legumes"},
		{ID: "frutas", ParentID: "", Name: "Frutas"},
		{ID: "citricas", ParentID: "frutas", Name: "Cítricas"},
	}
	tree := BuildCategoryTree(items)
	if len(tree) != 2 {
		t.Fatalf("roots=%d", len(tree))
	}
	byName := map[string]Category{}
	for _, c := range tree {
		byName[c.Name] = c
	}
	if len(byName["Legumes"].Children) != 1 || byName["Legumes"].Children[0].Name != "Folhas" {
		t.Fatalf("%+v", byName["Legumes"])
	}
	if len(byName["Frutas"].Children) != 1 || byName["Frutas"].Children[0].Name != "Cítricas" {
		t.Fatalf("%+v", byName["Frutas"])
	}
}

func TestCategoryParentWouldCycle(t *testing.T) {
	items := []Category{
		{ID: "a", Name: "A"},
		{ID: "b", ParentID: "a", Name: "B"},
		{ID: "c", ParentID: "b", Name: "C"},
	}
	if CategoryParentWouldCycle(items, "a", "") {
		t.Fatal("empty parent")
	}
	if !CategoryParentWouldCycle(items, "a", "a") {
		t.Fatal("self")
	}
	if !CategoryParentWouldCycle(items, "a", "c") {
		t.Fatal("descendant")
	}
	if CategoryParentWouldCycle(items, "c", "a") {
		t.Fatal("valid parent")
	}
}
