package application

import (
	"context"
	"testing"

	"erp/services/stock-service/internal/domain"
)

type memProducts struct{ byID map[string]domain.Product }

func (m *memProducts) Create(_ context.Context, p domain.Product) (domain.Product, error) {
	m.byID[p.ID] = p
	return p, nil
}
func (m *memProducts) Update(context.Context, domain.Product) error { return nil }
func (m *memProducts) Get(_ context.Context, id string) (domain.Product, error) {
	p, ok := m.byID[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	return p, nil
}
func (m *memProducts) List(context.Context) ([]domain.Product, error) { return nil, nil }
func (m *memProducts) Delete(context.Context, string) error           { return nil }
func (m *memProducts) SetSalePrice(context.Context, string, float64) error {
	return nil
}
func (m *memProducts) SetPurchasePrice(context.Context, string, float64) error {
	return nil
}
func (m *memProducts) CountByCategory(context.Context, string) (int64, error) { return 0, nil }

type memStock struct {
	wh  map[string]domain.Warehouse
	bal map[string]float64
	mov []domain.Movement

	lastReservedOrderID     string
	lastReservedWarehouseID string
	lastReservedItems       []domain.OrderItem
	reservations            []domain.Reservation
}

func balKey(p, w string) string { return p + "|" + w }

func (m *memStock) CreateWarehouse(_ context.Context, w domain.Warehouse) (domain.Warehouse, error) {
	return w, nil
}
func (m *memStock) UpdateWarehouse(context.Context, domain.Warehouse) error { return nil }
func (m *memStock) ListWarehouses(context.Context) ([]domain.Warehouse, error) {
	return nil, nil
}
func (m *memStock) GetWarehouse(_ context.Context, id string) (domain.Warehouse, error) {
	w, ok := m.wh[id]
	if !ok {
		return domain.Warehouse{}, domain.ErrNotFound
	}
	return w, nil
}
func (m *memStock) UpsertBalance(context.Context, string, string, float64) (domain.Balance, error) {
	return domain.Balance{}, nil
}
func (m *memStock) GetBalance(_ context.Context, productID, warehouseID string) (domain.Balance, error) {
	return domain.Balance{ProductID: productID, WarehouseID: warehouseID, QuantityAvailable: m.bal[balKey(productID, warehouseID)]}, nil
}
func (m *memStock) ListBalances(context.Context) ([]domain.Balance, error) { return nil, nil }
func (m *memStock) ReplaceReservations(_ context.Context, orderID, warehouseID string, items []domain.OrderItem) error {
	m.lastReservedOrderID = orderID
	m.lastReservedWarehouseID = warehouseID
	m.lastReservedItems = items
	return nil
}
func (m *memStock) ReleaseReservations(context.Context, string) error { return nil }
func (m *memStock) ConfirmReservations(context.Context, string) ([]domain.Reservation, error) {
	return nil, nil
}
func (m *memStock) ListOpenReservations(_ context.Context, productID string) ([]domain.Reservation, error) {
	out := []domain.Reservation{}
	for _, rv := range m.reservations {
		if rv.ProductID == productID && rv.Status == "OPEN" {
			out = append(out, rv)
		}
	}
	return out, nil
}
func (m *memStock) Reserve(context.Context, string, string, float64) error { return nil }
func (m *memStock) ConfirmSale(context.Context, string, string, float64) error {
	return nil
}
func (m *memStock) AddAvailable(_ context.Context, productID, warehouseID string, qty float64) error {
	m.bal[balKey(productID, warehouseID)] += qty
	return nil
}
func (m *memStock) RemoveAvailable(_ context.Context, productID, warehouseID string, qty float64) error {
	k := balKey(productID, warehouseID)
	if m.bal[k] < qty {
		return domain.ErrInsufficient
	}
	m.bal[k] -= qty
	return nil
}
func (m *memStock) HasMovement(_ context.Context, docType, docID, movementType string) (bool, error) {
	for _, mv := range m.mov {
		if mv.ReferenceDocType == docType && mv.ReferenceDocID == docID && mv.MovementType == movementType {
			return true, nil
		}
	}
	return false, nil
}
func (m *memStock) ProductInUse(context.Context, string) (bool, error) { return false, nil }
func (m *memStock) InsertMovement(_ context.Context, mv domain.Movement) error {
	m.mov = append(m.mov, mv)
	return nil
}
func (m *memStock) ListMovements(context.Context) ([]domain.Movement, error) { return m.mov, nil }

type memCatalog struct{ assemblies []domain.Assembly }

func (m *memCatalog) InsertSaleHistory(context.Context, domain.PriceHistory) (domain.PriceHistory, error) {
	return domain.PriceHistory{}, nil
}
func (m *memCatalog) InsertPurchaseHistory(context.Context, domain.PriceHistory) (domain.PriceHistory, error) {
	return domain.PriceHistory{}, nil
}
func (m *memCatalog) ListSaleHistory(context.Context, string) ([]domain.PriceHistory, error) {
	return nil, nil
}
func (m *memCatalog) ListPurchaseHistory(context.Context, string) ([]domain.PriceHistory, error) {
	return nil, nil
}
func (m *memCatalog) CreateAssembly(_ context.Context, a domain.Assembly) (domain.Assembly, error) {
	return a, nil
}
func (m *memCatalog) UpdateAssembly(context.Context, domain.Assembly) error { return nil }
func (m *memCatalog) GetAssembly(_ context.Context, id string) (domain.Assembly, error) {
	for _, a := range m.assemblies {
		if a.ID == id {
			return a, nil
		}
	}
	return domain.Assembly{}, domain.ErrNotFound
}
func (m *memCatalog) ListAssemblies(context.Context) ([]domain.Assembly, error) {
	return m.assemblies, nil
}
func (m *memCatalog) ProductInUse(context.Context, string) (bool, error) { return false, nil }

type nopPublisher struct{}

func (nopPublisher) PublishReserved(context.Context, domain.OrderEvent) error { return nil }

type nopLock struct{}

func (nopLock) Lock(context.Context, string) (func(context.Context), error) {
	return func(context.Context) {}, nil
}

type nopCache struct{}

func (nopCache) SetBalance(context.Context, domain.Balance) error { return nil }
func (nopCache) GetBalance(context.Context, string, string) (domain.Balance, bool, error) {
	return domain.Balance{}, false, nil
}

func stockSvc() (*Service, *memStock) {
	svc, st, _ := stockSvcWithCatalog(nil)
	return svc, st
}

func stockSvcWithCatalog(assemblies []domain.Assembly) (*Service, *memStock, *memCatalog) {
	st := &memStock{
		wh:  map[string]domain.Warehouse{"w1": {ID: "w1", Code: "000001", Name: "TESTE"}},
		bal: map[string]float64{},
	}
	cat := &memCatalog{assemblies: assemblies}
	svc := New(
		&memProducts{byID: map[string]domain.Product{"p1": {ID: "p1", Name: "Banana"}}},
		st, cat, nopPublisher{}, nopLock{}, nopCache{}, nil, nil, nil, nil,
	)
	return svc, st, cat
}

func TestCreateMovementIn(t *testing.T) {
	svc, st := stockSvc()
	if err := svc.CreateMovement(context.Background(), "p1", "w1", "IN", 20); err != nil {
		t.Fatal(err)
	}
	if st.bal[balKey("p1", "w1")] != 20 {
		t.Fatalf("%v", st.bal)
	}
	if len(st.mov) != 1 || st.mov[0].MovementType != "MANUAL_IN" || st.mov[0].ReferenceDocType != "MANUAL" || st.mov[0].ReferenceDocID != "" {
		t.Fatalf("%+v", st.mov)
	}
}

func TestCreateMovementOut(t *testing.T) {
	svc, st := stockSvc()
	st.bal[balKey("p1", "w1")] = 10
	if err := svc.CreateMovement(context.Background(), "p1", "w1", "OUT", 4); err != nil {
		t.Fatal(err)
	}
	if st.bal[balKey("p1", "w1")] != 6 || st.mov[0].MovementType != "MANUAL_OUT" {
		t.Fatalf("%v %+v", st.bal, st.mov)
	}
}

func TestCreateMovementRejectsEmpty(t *testing.T) {
	svc, _ := stockSvc()
	if err := svc.CreateMovement(context.Background(), "", "w1", "IN", 1); err != domain.ErrInvalid {
		t.Fatalf("%v", err)
	}
	if err := svc.CreateMovement(context.Background(), "p1", "w1", "IN", 0); err != domain.ErrInvalid {
		t.Fatalf("%v", err)
	}
	if err := svc.CreateMovement(context.Background(), "p1", "w1", "X", 1); err != domain.ErrInvalid {
		t.Fatalf("%v", err)
	}
}

func TestReceivePurchaseAddsStock(t *testing.T) {
	svc, st := stockSvc()
	items := []domain.OrderItem{{ProductID: "p1", Quantity: 5}}
	if err := svc.ReceivePurchase(context.Background(), "po1", "w1", items); err != nil {
		t.Fatal(err)
	}
	if st.bal[balKey("p1", "w1")] != 5 || st.mov[0].MovementType != "PURCHASE_IN" || st.mov[0].ReferenceDocID != "po1" {
		t.Fatalf("%v %+v", st.bal, st.mov)
	}
	if err := svc.ReceivePurchase(context.Background(), "po1", "w1", items); err != nil {
		t.Fatal(err)
	}
	if st.bal[balKey("p1", "w1")] != 5 || len(st.mov) != 1 {
		t.Fatalf("not idempotent: %v %+v", st.bal, st.mov)
	}
}

func TestReceivePurchaseConvertsPurchaseUoM(t *testing.T) {
	svc, st := stockSvc()
	svc.products.(*memProducts).byID["p1"] = domain.Product{
		ID: "p1", Name: "Banana", PurchaseUoM: "CX", SaleUoM: "KG", StockUoM: "KG",
		Conversions: []domain.UoMConversion{{FromUoM: "CX", ToUoM: "KG", Factor: 20}},
	}
	items := []domain.OrderItem{{ProductID: "p1", Quantity: 1}}
	if err := svc.ReceivePurchase(context.Background(), "po1", "w1", items); err != nil {
		t.Fatal(err)
	}
	if st.bal[balKey("p1", "w1")] != 20 {
		t.Fatalf("got %v want 20 KG", st.bal)
	}
}

func TestReserveOrderExpandsKitIntoComponents(t *testing.T) {
	kit := domain.Assembly{
		ID: "asm1", ProductID: "cesta3",
		Items: []domain.AssemblyItem{{ProductID: "apple", Quantity: 2, Role: domain.RoleComponent}},
	}
	svc, st, _ := stockSvcWithCatalog([]domain.Assembly{kit})

	ev := domain.OrderEvent{
		OrderID: "o1", WarehouseID: "w1",
		Items: []domain.OrderItem{{ProductID: "cesta3", Quantity: 3}},
	}
	if err := svc.ReserveOrder(context.Background(), ev); err != nil {
		t.Fatalf("ReserveOrder: %v", err)
	}
	if st.lastReservedOrderID != "o1" || st.lastReservedWarehouseID != "w1" {
		t.Fatalf("wrong order/warehouse recorded: %+v", st)
	}
	if len(st.lastReservedItems) != 1 || st.lastReservedItems[0].ProductID != "apple" || st.lastReservedItems[0].Quantity != 6 {
		t.Fatalf("expected reservation for 6 apples (2/kit * 3 kits), got: %+v", st.lastReservedItems)
	}
}

func TestReserveOrderHonorsCustomerSubstitution(t *testing.T) {
	kit := domain.Assembly{
		ID: "asm1", ProductID: "cesta3",
		Items: []domain.AssemblyItem{{ProductID: "apple", Quantity: 1, Role: domain.RoleComponent}},
	}
	svc, st, _ := stockSvcWithCatalog([]domain.Assembly{kit})

	ev := domain.OrderEvent{
		OrderID: "o1", WarehouseID: "w1",
		Items: []domain.OrderItem{{
			ProductID: "cesta3", Quantity: 1,
			Components: []domain.OrderItemComponent{{ProductID: "grape", Quantity: 2}},
		}},
	}
	if err := svc.ReserveOrder(context.Background(), ev); err != nil {
		t.Fatalf("ReserveOrder: %v", err)
	}
	if len(st.lastReservedItems) != 1 || st.lastReservedItems[0].ProductID != "grape" || st.lastReservedItems[0].Quantity != 2 {
		t.Fatalf("expected the substituted grape reservation, got: %+v", st.lastReservedItems)
	}
}

func TestReserveOrderPlainProductUnaffected(t *testing.T) {
	svc, st := stockSvc()
	ev := domain.OrderEvent{
		OrderID: "o1", WarehouseID: "w1",
		Items: []domain.OrderItem{{ProductID: "p1", Quantity: 4}},
	}
	if err := svc.ReserveOrder(context.Background(), ev); err != nil {
		t.Fatalf("ReserveOrder: %v", err)
	}
	if len(st.lastReservedItems) != 1 || st.lastReservedItems[0].ProductID != "p1" || st.lastReservedItems[0].Quantity != 4 {
		t.Fatalf("plain product line should pass through unchanged, got: %+v", st.lastReservedItems)
	}
}

func TestListOpenReservationsFiltersByProductAndStatus(t *testing.T) {
	svc, st := stockSvc()
	st.reservations = []domain.Reservation{
		{ID: "r1", SalesOrderID: "so1", ProductID: "p1", Quantity: 2, Status: "OPEN"},
		{ID: "r2", SalesOrderID: "so2", ProductID: "p1", Quantity: 3, Status: "OPEN"},
		{ID: "r3", SalesOrderID: "so3", ProductID: "p1", Quantity: 5, Status: "CONFIRMED"},
		{ID: "r4", SalesOrderID: "so4", ProductID: "p2", Quantity: 1, Status: "OPEN"},
	}
	out, err := svc.ListOpenReservations(context.Background(), "p1")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 open reservations for p1, got: %+v", out)
	}
	for _, rv := range out {
		if rv.ProductID != "p1" || rv.Status != "OPEN" {
			t.Fatalf("wrong reservation returned: %+v", rv)
		}
	}
}

func TestListOpenReservationsRequiresProductID(t *testing.T) {
	svc, _ := stockSvc()
	if _, err := svc.ListOpenReservations(context.Background(), ""); err != domain.ErrInvalid {
		t.Fatalf("%v", err)
	}
}
