package domain

import "testing"

func kitAssembly() Assembly {
	return Assembly{
		ID: "asm-cesta3", ProductID: "prod-cesta3",
		Items: []AssemblyItem{
			{ProductID: "prod-apple", Quantity: 1, Role: RoleComponent},
			{ProductID: "prod-bag", Quantity: 1, Role: RoleSupport},
		},
	}
}

func TestExpandKitItemsPassesThroughPlainProducts(t *testing.T) {
	items := []OrderItem{{ProductID: "prod-apple", Quantity: 3}}
	out := ExpandKitItems(items, []Assembly{kitAssembly()})
	if len(out) != 1 || out[0].ProductID != "prod-apple" || out[0].Quantity != 3 {
		t.Fatalf("plain product should pass through unchanged, got: %+v", out)
	}
}

func TestExpandKitItemsUsesDefaultRecipe(t *testing.T) {
	items := []OrderItem{{ProductID: "prod-cesta3", Quantity: 2}}
	out := ExpandKitItems(items, []Assembly{kitAssembly()})

	if len(out) != 2 {
		t.Fatalf("expected 2 expanded lines (component + support), got: %+v", out)
	}
	byProduct := map[string]float64{}
	for _, it := range out {
		byProduct[it.ProductID] = it.Quantity
		if it.ProductID == "prod-cesta3" {
			t.Fatalf("kit's own product must never appear in expanded output, got: %+v", out)
		}
	}
	if byProduct["prod-apple"] != 2 {
		t.Fatalf("apple qty = %v, want 2 (1 per kit * 2 kits)", byProduct["prod-apple"])
	}
	if byProduct["prod-bag"] != 2 {
		t.Fatalf("bag (support role) qty = %v, want 2 — support items must be consumed too", byProduct["prod-bag"])
	}
}

func TestExpandKitItemsCustomerSubstitutionWinsOverRecipe(t *testing.T) {
	items := []OrderItem{{
		ProductID: "prod-cesta3", Quantity: 2,
		Components: []OrderItemComponent{{ProductID: "prod-grape", Quantity: 3}},
	}}
	out := ExpandKitItems(items, []Assembly{kitAssembly()})

	if len(out) != 1 || out[0].ProductID != "prod-grape" || out[0].Quantity != 3 {
		t.Fatalf("substitution should fully replace the default recipe, got: %+v", out)
	}
}

func TestExpandKitItemsMixedOrderLines(t *testing.T) {
	items := []OrderItem{
		{ProductID: "prod-cesta3", Quantity: 1},
		{ProductID: "prod-banana", Quantity: 5},
	}
	out := ExpandKitItems(items, []Assembly{kitAssembly()})

	byProduct := map[string]float64{}
	for _, it := range out {
		byProduct[it.ProductID] += it.Quantity
	}
	if byProduct["prod-banana"] != 5 {
		t.Fatalf("unrelated line got altered: %+v", out)
	}
	if byProduct["prod-apple"] != 1 || byProduct["prod-bag"] != 1 {
		t.Fatalf("kit line not expanded correctly: %+v", out)
	}
}

func TestExpandKitItemsSkipsInvalidComponentRows(t *testing.T) {
	items := []OrderItem{{
		ProductID: "prod-cesta3", Quantity: 1,
		Components: []OrderItemComponent{{ProductID: "", Quantity: 1}, {ProductID: "prod-grape", Quantity: 0}},
	}}
	out := ExpandKitItems(items, []Assembly{kitAssembly()})
	if len(out) != 0 {
		t.Fatalf("invalid substitution rows should be dropped, got: %+v", out)
	}
}

func TestExpandKitItemsNoAssembliesLeavesEverythingUnchanged(t *testing.T) {
	items := []OrderItem{{ProductID: "prod-cesta3", Quantity: 1}}
	out := ExpandKitItems(items, nil)
	if len(out) != 1 || out[0].ProductID != "prod-cesta3" {
		t.Fatalf("with no assemblies registered, nothing can be identified as a kit: %+v", out)
	}
}
