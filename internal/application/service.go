package application

import (
	"context"
	"fmt"
	"strings"

	"erp/pkg/codes"
	"erp/services/stock-service/internal/domain"

	"github.com/google/uuid"
)

type Service struct {
	products   domain.ProductRepository
	stock      domain.StockRepository
	catalog    domain.CatalogRepository
	pub        domain.EventPublisher
	lock       domain.Locker
	cache      domain.Cache
	categories domain.CategoryRepository
	skuSeq     codes.Sequence
	whSeq      codes.Sequence
	asmSeq     codes.Sequence
}

func New(products domain.ProductRepository, stock domain.StockRepository, catalog domain.CatalogRepository, pub domain.EventPublisher, lock domain.Locker, cache domain.Cache, categories domain.CategoryRepository, skuSeq, whSeq, asmSeq codes.Sequence) *Service {
	return &Service{products: products, stock: stock, catalog: catalog, pub: pub, lock: lock, cache: cache, categories: categories, skuSeq: skuSeq, whSeq: whSeq, asmSeq: asmSeq}
}

func (s *Service) CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	p = normalizeProduct(p)
	if err := validateProduct(p); err != nil {
		return domain.Product{}, err
	}
	sku, err := codes.Assign(ctx, p.SKU, s.skuSeq)
	if err != nil {
		return domain.Product{}, err
	}
	p.SKU = sku
	if err := s.ensureCategoryParent(ctx, p.CategoryID); err != nil {
		return domain.Product{}, err
	}
	return s.products.Create(ctx, p)
}

func (s *Service) UpdateProduct(ctx context.Context, p domain.Product) error {
	cur, err := s.products.Get(ctx, p.ID)
	if err != nil {
		return err
	}
	p.SKU = cur.SKU
	p = normalizeProduct(p)
	if err := validateProduct(p); err != nil {
		return err
	}
	if err := s.ensureCategoryParent(ctx, p.CategoryID); err != nil {
		return err
	}
	return s.products.Update(ctx, p)
}

func (s *Service) CreateCategory(ctx context.Context, c domain.Category) (domain.Category, error) {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return domain.Category{}, domain.ErrInvalid
	}
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if err := s.ensureCategoryParent(ctx, c.ParentID); err != nil {
		return domain.Category{}, err
	}
	return s.categories.Create(ctx, c)
}

func (s *Service) UpdateCategory(ctx context.Context, c domain.Category) error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return domain.ErrInvalid
	}
	items, err := s.categories.List(ctx)
	if err != nil {
		return err
	}
	if domain.CategoryParentWouldCycle(items, c.ID, c.ParentID) {
		return domain.ErrInvalid
	}
	if err := s.ensureCategoryParent(ctx, c.ParentID); err != nil {
		return err
	}
	return s.categories.Update(ctx, c)
}

func (s *Service) ListCategories(ctx context.Context) ([]domain.Category, error) {
	items, err := s.categories.List(ctx)
	if err != nil {
		return nil, err
	}
	return domain.BuildCategoryTree(items), nil
}

func (s *Service) DeleteCategory(ctx context.Context, id string) error {
	has, err := s.categories.HasChildren(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return domain.ErrInUse
	}
	n, err := s.products.CountByCategory(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return domain.ErrInUse
	}
	return s.categories.Delete(ctx, id)
}

func (s *Service) ensureCategoryParent(ctx context.Context, parentID string) error {
	if parentID == "" {
		return nil
	}
	_, err := s.categories.Get(ctx, parentID)
	return err
}

func (s *Service) GetProduct(ctx context.Context, id string) (domain.Product, error) {
	return s.products.Get(ctx, id)
}

func (s *Service) ListProducts(ctx context.Context) ([]domain.Product, error) {
	return s.products.List(ctx)
}

func (s *Service) DeleteProduct(ctx context.Context, id string) error {
	used, err := s.stock.ProductInUse(ctx, id)
	if err != nil {
		return err
	}
	if used {
		return domain.ErrInUse
	}
	used, err = s.catalog.ProductInUse(ctx, id)
	if err != nil {
		return err
	}
	if used {
		return domain.ErrInUse
	}
	return s.products.Delete(ctx, id)
}

func (s *Service) CreateWarehouse(ctx context.Context, w domain.Warehouse) (domain.Warehouse, error) {
	w.Name = strings.TrimSpace(w.Name)
	if w.Name == "" {
		return domain.Warehouse{}, domain.ErrInvalid
	}
	code, err := codes.Assign(ctx, w.Code, s.whSeq)
	if err != nil {
		return domain.Warehouse{}, err
	}
	w.Code = code
	return s.stock.CreateWarehouse(ctx, w)
}

func (s *Service) UpdateWarehouse(ctx context.Context, w domain.Warehouse) error {
	return s.stock.UpdateWarehouse(ctx, w)
}

func (s *Service) ListWarehouses(ctx context.Context) ([]domain.Warehouse, error) {
	return s.stock.ListWarehouses(ctx)
}

func (s *Service) SetBalance(ctx context.Context, productID, warehouseID string, qty float64) (domain.Balance, error) {
	b, err := s.stock.UpsertBalance(ctx, productID, warehouseID, qty)
	if err != nil {
		return domain.Balance{}, err
	}
	_ = s.cache.SetBalance(ctx, b)
	return b, nil
}

func (s *Service) ListBalances(ctx context.Context) ([]domain.Balance, error) {
	return s.stock.ListBalances(ctx)
}

func (s *Service) ListMovements(ctx context.Context) ([]domain.Movement, error) {
	return s.stock.ListMovements(ctx)
}

func (s *Service) ListOpenReservations(ctx context.Context, productID string) ([]domain.Reservation, error) {
	if productID == "" {
		return nil, domain.ErrInvalid
	}
	return s.stock.ListOpenReservations(ctx, productID)
}

func (s *Service) ReserveOrder(ctx context.Context, ev domain.OrderEvent) error {
	unlock, err := s.lock.Lock(ctx, "reserve:"+ev.OrderID)
	if err != nil {
		return err
	}
	defer unlock(ctx)
	if ev.OrderID == "" {
		return domain.ErrInvalid
	}
	if len(ev.Items) == 0 {
		return s.stock.ReleaseReservations(ctx, ev.OrderID)
	}
	assemblies, err := s.catalog.ListAssemblies(ctx)
	if err != nil {
		return err
	}
	if err := s.stock.ReplaceReservations(ctx, ev.OrderID, ev.WarehouseID, domain.ExpandKitItems(ev.Items, assemblies)); err != nil {
		return err
	}
	return s.pub.PublishReserved(ctx, ev)
}

func (s *Service) ConfirmInvoice(ctx context.Context, ev domain.InvoiceEvent) error {
	if ev.Direction == "IN" {
		return s.confirmInbound(ctx, ev)
	}
	return s.confirmOutbound(ctx, ev)
}

func (s *Service) CreateMovement(ctx context.Context, productID, warehouseID, direction string, qty float64) error {
	if qty <= 0 || productID == "" || warehouseID == "" {
		return domain.ErrInvalid
	}
	if _, err := s.products.Get(ctx, productID); err != nil {
		return err
	}
	if _, err := s.stock.GetWarehouse(ctx, warehouseID); err != nil {
		return err
	}
	typ := "MANUAL_IN"
	if strings.ToUpper(direction) == "OUT" {
		typ = "MANUAL_OUT"
	} else if strings.ToUpper(direction) != "IN" {
		return domain.ErrInvalid
	}
	return s.applyQty(ctx, productID, warehouseID, typ, qty, "MANUAL", "")
}

func (s *Service) ReceivePurchase(ctx context.Context, orderID, warehouseID string, items []domain.OrderItem) error {
	if orderID == "" || warehouseID == "" {
		return domain.ErrInvalid
	}
	if _, err := s.stock.GetWarehouse(ctx, warehouseID); err != nil {
		return err
	}
	done, err := s.stock.HasMovement(ctx, "PURCHASE_ORDER", orderID, "PURCHASE_IN")
	if err != nil {
		return err
	}
	if done {
		return nil
	}
	for _, it := range items {
		if it.ProductID == "" || it.Quantity <= 0 {
			continue
		}
		qty, err := s.toStockFromPurchase(ctx, it.ProductID, it.Quantity)
		if err != nil {
			return err
		}
		if err := s.applyQty(ctx, it.ProductID, warehouseID, "PURCHASE_IN", qty, "PURCHASE_ORDER", orderID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) confirmInbound(ctx context.Context, ev domain.InvoiceEvent) error {
	done, err := s.stock.HasMovement(ctx, "INVOICE", ev.InvoiceID, "PURCHASE_IN")
	if err != nil {
		return err
	}
	if done {
		return nil
	}
	if ev.WarehouseID == "" || len(ev.Items) == 0 {
		return domain.ErrInvalid
	}
	for _, it := range ev.Items {
		if it.ProductID == "" || it.Quantity <= 0 {
			continue
		}
		qty, err := s.toStockFromPurchase(ctx, it.ProductID, it.Quantity)
		if err != nil {
			return err
		}
		if err := s.applyQty(ctx, it.ProductID, ev.WarehouseID, "PURCHASE_IN", qty, "INVOICE", ev.InvoiceID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) confirmOutbound(ctx context.Context, ev domain.InvoiceEvent) error {
	done, err := s.stock.HasMovement(ctx, "INVOICE", ev.InvoiceID, "SALE_OUT")
	if err != nil {
		return err
	}
	if done {
		return nil
	}
	if ev.SalesOrderID != "" {
		rsv, err := s.stock.ConfirmReservations(ctx, ev.SalesOrderID)
		if err != nil {
			return err
		}
		if rsv != nil {
			if len(rsv) == 0 {
				return nil
			}
			return s.applyConfirmedReservations(ctx, ev, rsv)
		}
		orderDone, err := s.stock.HasMovement(ctx, "SALES_ORDER", ev.SalesOrderID, "RESERVATION_ADD")
		if err != nil {
			return err
		}
		if orderDone {
			return s.confirmReservedSale(ctx, ev)
		}
	}
	if ev.WarehouseID == "" || len(ev.Items) == 0 {
		return domain.ErrNotFound
	}
	assemblies, err := s.catalog.ListAssemblies(ctx)
	if err != nil {
		return err
	}
	for _, it := range domain.ExpandKitItems(ev.Items, assemblies) {
		if it.ProductID == "" || it.Quantity <= 0 {
			continue
		}
		if err := s.applyQty(ctx, it.ProductID, ev.WarehouseID, "SALE_OUT", it.Quantity, "INVOICE", ev.InvoiceID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) applyConfirmedReservations(ctx context.Context, ev domain.InvoiceEvent, rsv []domain.Reservation) error {
	for _, rv := range rsv {
		wh := rv.WarehouseID
		if wh == "" {
			wh = ev.WarehouseID
		}
		if wh == "" || rv.Quantity <= 0 {
			continue
		}
		key := fmt.Sprintf("stock:%s:%s", rv.ProductID, wh)
		unlock, err := s.lock.Lock(ctx, key)
		if err != nil {
			return err
		}
		if err := s.stock.RemoveAvailable(ctx, rv.ProductID, wh, rv.Quantity); err != nil {
			unlock(ctx)
			return err
		}
		_ = s.stock.InsertMovement(ctx, domain.Movement{
			ProductID: rv.ProductID, WarehouseID: wh, MovementType: "RESERVATION_RELEASE",
			Quantity: rv.Quantity, ReferenceDocType: "INVOICE", ReferenceDocID: ev.InvoiceID,
		})
		_ = s.stock.InsertMovement(ctx, domain.Movement{
			ProductID: rv.ProductID, WarehouseID: wh, MovementType: "SALE_OUT",
			Quantity: rv.Quantity, ReferenceDocType: "INVOICE", ReferenceDocID: ev.InvoiceID,
		})
		if b, err := s.stock.GetBalance(ctx, rv.ProductID, wh); err == nil {
			_ = s.cache.SetBalance(ctx, b)
		}
		unlock(ctx)
	}
	return nil
}

func (s *Service) confirmReservedSale(ctx context.Context, ev domain.InvoiceEvent) error {
	movs, err := s.stock.ListMovements(ctx)
	if err != nil {
		return err
	}
	for _, m := range movs {
		if m.ReferenceDocType != "SALES_ORDER" || m.ReferenceDocID != ev.SalesOrderID || m.MovementType != "RESERVATION_ADD" {
			continue
		}
		key := fmt.Sprintf("stock:%s:%s", m.ProductID, m.WarehouseID)
		unlock, err := s.lock.Lock(ctx, key)
		if err != nil {
			return err
		}
		if err := s.stock.ConfirmSale(ctx, m.ProductID, m.WarehouseID, m.Quantity); err != nil {
			unlock(ctx)
			return err
		}
		_ = s.stock.InsertMovement(ctx, domain.Movement{
			ProductID: m.ProductID, WarehouseID: m.WarehouseID, MovementType: "RESERVATION_RELEASE",
			Quantity: m.Quantity, ReferenceDocType: "INVOICE", ReferenceDocID: ev.InvoiceID,
		})
		_ = s.stock.InsertMovement(ctx, domain.Movement{
			ProductID: m.ProductID, WarehouseID: m.WarehouseID, MovementType: "SALE_OUT",
			Quantity: m.Quantity, ReferenceDocType: "INVOICE", ReferenceDocID: ev.InvoiceID,
		})
		if b, err := s.stock.GetBalance(ctx, m.ProductID, m.WarehouseID); err == nil {
			_ = s.cache.SetBalance(ctx, b)
		}
		unlock(ctx)
	}
	return nil
}

func (s *Service) toStockFromPurchase(ctx context.Context, productID string, qty float64) (float64, error) {
	p, err := s.products.Get(ctx, productID)
	if err != nil {
		return 0, err
	}
	p = normalizeProduct(p)
	return p.ToStockQty(qty, p.PurchaseUoM)
}

func (s *Service) applyQty(ctx context.Context, productID, warehouseID, typ string, qty float64, docType, docID string) error {
	key := fmt.Sprintf("stock:%s:%s", productID, warehouseID)
	unlock, err := s.lock.Lock(ctx, key)
	if err != nil {
		return err
	}
	defer unlock(ctx)
	if typ == "MANUAL_IN" || typ == "PURCHASE_IN" {
		if err := s.stock.AddAvailable(ctx, productID, warehouseID, qty); err != nil {
			return err
		}
	} else {
		if err := s.stock.RemoveAvailable(ctx, productID, warehouseID, qty); err != nil {
			return err
		}
	}
	if err := s.stock.InsertMovement(ctx, domain.Movement{
		ProductID: productID, WarehouseID: warehouseID, MovementType: typ,
		Quantity: qty, ReferenceDocType: docType, ReferenceDocID: docID,
	}); err != nil {
		return err
	}
	if b, err := s.stock.GetBalance(ctx, productID, warehouseID); err == nil {
		_ = s.cache.SetBalance(ctx, b)
	}
	return nil
}

func normalizeProduct(p domain.Product) domain.Product {
	if p.PurchaseUoM == "" && p.SaleUoM != "" {
		p.PurchaseUoM = p.SaleUoM
	}
	if p.SaleUoM == "" && p.PurchaseUoM != "" {
		p.SaleUoM = p.PurchaseUoM
	}
	if p.PurchaseUoM == "" {
		p.PurchaseUoM = "UN"
	}
	if p.SaleUoM == "" {
		p.SaleUoM = p.PurchaseUoM
	}
	p.StockUoM = p.SaleUoM
	p.UnitOfMeasure = p.SaleUoM
	if p.PurchaseUoM == p.SaleUoM {
		p.Conversions = []domain.UoMConversion{}
	} else if p.Conversions == nil {
		p.Conversions = []domain.UoMConversion{}
	}
	if p.Kind == "" {
		p.Kind = domain.KindFinal
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Barcode = strings.TrimSpace(p.Barcode)
	p.NCM = strings.TrimSpace(p.NCM)
	p.CategoryID = strings.TrimSpace(p.CategoryID)
	return p
}

func validateProduct(p domain.Product) error {
	if strings.TrimSpace(p.Name) == "" {
		return domain.ErrInvalid
	}
	if p.Kind != domain.KindFinal && p.Kind != domain.KindSupport && p.Kind != domain.KindFixedAsset {
		return domain.ErrInvalid
	}
	return validateConversions(p)
}

func validateAssembly(a domain.Assembly) error {
	if a.Code == "" || a.Name == "" || len(a.Items) == 0 {
		return domain.ErrInvalid
	}
	for _, it := range a.Items {
		if it.ProductID == "" || it.Quantity <= 0 {
			return domain.ErrInvalid
		}
		if it.Role != domain.RoleComponent && it.Role != domain.RoleSupport {
			return domain.ErrInvalid
		}
	}
	return nil
}

func (s *Service) RegisterSalePrice(ctx context.Context, productID string, newPrice float64) (domain.PriceHistory, error) {
	p, err := s.products.Get(ctx, productID)
	if err != nil {
		return domain.PriceHistory{}, err
	}
	if err := s.products.SetSalePrice(ctx, p.ID, newPrice); err != nil {
		return domain.PriceHistory{}, err
	}
	return s.catalog.InsertSaleHistory(ctx, domain.PriceHistory{
		ProductID:     p.ID,
		SKU:           p.SKU,
		PreviousPrice: p.SalePrice,
		NewPrice:      newPrice,
	})
}

func (s *Service) RegisterPurchasePrice(ctx context.Context, productID string, newPrice float64, refDocID string) (domain.PriceHistory, error) {
	p, err := s.products.Get(ctx, productID)
	if err != nil {
		return domain.PriceHistory{}, err
	}
	if err := s.products.SetPurchasePrice(ctx, p.ID, newPrice); err != nil {
		return domain.PriceHistory{}, err
	}
	return s.catalog.InsertPurchaseHistory(ctx, domain.PriceHistory{
		ProductID:      p.ID,
		SKU:            p.SKU,
		PreviousPrice:  p.PurchasePrice,
		NewPrice:       newPrice,
		ReferenceDocID: refDocID,
	})
}

func (s *Service) ListSaleHistory(ctx context.Context, productID string) ([]domain.PriceHistory, error) {
	return s.catalog.ListSaleHistory(ctx, productID)
}

func (s *Service) ListPurchaseHistory(ctx context.Context, productID string) ([]domain.PriceHistory, error) {
	return s.catalog.ListPurchaseHistory(ctx, productID)
}

func (s *Service) CreateAssembly(ctx context.Context, a domain.Assembly) (domain.Assembly, error) {
	code, err := codes.Assign(ctx, a.Code, s.asmSeq)
	if err != nil {
		return domain.Assembly{}, err
	}
	a.Code = code
	a.Active = true
	if err := validateAssembly(a); err != nil {
		return domain.Assembly{}, err
	}
	out, err := s.catalog.CreateAssembly(ctx, a)
	if err != nil {
		return domain.Assembly{}, err
	}
	return s.RecalculateAssembly(ctx, out.ID)
}

func (s *Service) UpdateAssembly(ctx context.Context, a domain.Assembly) (domain.Assembly, error) {
	if err := validateAssembly(a); err != nil {
		return domain.Assembly{}, err
	}
	if err := s.catalog.UpdateAssembly(ctx, a); err != nil {
		return domain.Assembly{}, err
	}
	return s.RecalculateAssembly(ctx, a.ID)
}

func (s *Service) GetAssembly(ctx context.Context, id string) (domain.Assembly, error) {
	return s.catalog.GetAssembly(ctx, id)
}

func (s *Service) ListAssemblies(ctx context.Context) ([]domain.Assembly, error) {
	return s.catalog.ListAssemblies(ctx)
}

func (s *Service) RecalculateAssembly(ctx context.Context, id string) (domain.Assembly, error) {
	a, err := s.catalog.GetAssembly(ctx, id)
	if err != nil {
		return domain.Assembly{}, err
	}
	var cost float64
	var priceSum float64
	var hasItemPrices bool
	for _, it := range a.Items {
		p, err := s.products.Get(ctx, it.ProductID)
		if err != nil {
			return domain.Assembly{}, err
		}
		qty := it.Quantity
		from := p.StockUoM
		if from == "" {
			from = p.SaleUoM
		}
		if from == "" {
			from = p.UnitOfMeasure
		}
		if from != "" && p.PurchaseUoM != "" && from != p.PurchaseUoM {
			if converted, cErr := domain.Convert(qty, from, p.PurchaseUoM, p.Conversions); cErr == nil {
				qty = converted
			}
		}
		cost += qty * p.PurchasePrice
		priceSum += it.UnitPrice
		if it.UnitPrice > 0 {
			hasItemPrices = true
		}
	}
	a.Cost = cost
	// Each item carries its own sale price (set independently on the Montagem screen), so the
	// kit's suggested price is their sum, not cost×margin — margin_percent becomes a derived,
	// informational blend of that sum over cost. Assemblies with no item prices set yet (created
	// before this field existed) keep the old cost×margin formula.
	if hasItemPrices {
		a.SuggestedPrice = priceSum
		if cost > 0 {
			a.MarginPercent = (priceSum/cost - 1) * 100
		}
	} else {
		a.SuggestedPrice = cost * (1 + a.MarginPercent/100)
	}
	if err := s.catalog.UpdateAssembly(ctx, a); err != nil {
		return domain.Assembly{}, err
	}
	return a, nil
}

func (s *Service) ApplyAssemblyPrice(ctx context.Context, id string) (domain.Assembly, error) {
	a, err := s.RecalculateAssembly(ctx, id)
	if err != nil {
		return domain.Assembly{}, err
	}
	if a.ProductID == "" {
		return domain.Assembly{}, domain.ErrInvalid
	}
	if _, err := s.RegisterSalePrice(ctx, a.ProductID, a.SuggestedPrice); err != nil {
		return domain.Assembly{}, err
	}
	return a, nil
}

func validateConversions(p domain.Product) error {
	if p.PurchaseUoM == p.SaleUoM {
		return nil
	}
	_, err := domain.Convert(1, p.PurchaseUoM, p.SaleUoM, p.Conversions)
	return err
}
