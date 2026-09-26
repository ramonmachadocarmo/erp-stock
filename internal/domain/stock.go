package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrInsufficient     = errors.New("insufficient stock")
	ErrAlreadyProcessed = errors.New("already processed")
	ErrNoConversion     = errors.New("missing unit conversion")
	ErrInvalid          = errors.New("invalid")
	ErrInUse            = errors.New("cannot delete: record is in use")
	ErrConflict         = errors.New("already exists")
)

type Product struct {
	ID            string          `bson:"_id" json:"id"`
	SKU           string          `bson:"sku" json:"sku"`
	Barcode       string          `bson:"barcode" json:"barcode"`
	Name          string          `bson:"name" json:"name"`
	PopularName   string          `bson:"popular_name" json:"popular_name"`
	CategoryID    string          `bson:"category_id" json:"category_id"`
	NCM           string          `bson:"ncm" json:"ncm"`
	UnitOfMeasure string          `bson:"unit_of_measure" json:"unit_of_measure"`
	PurchaseUoM   string          `bson:"purchase_uom" json:"purchase_uom"`
	SaleUoM       string          `bson:"sale_uom" json:"sale_uom"`
	StockUoM      string          `bson:"stock_uom" json:"stock_uom"`
	Conversions   []UoMConversion `bson:"uom_conversions" json:"uom_conversions"`
	Kind          string          `bson:"kind" json:"kind"`
	SalePrice     float64         `bson:"sale_price" json:"sale_price"`
	PurchasePrice float64         `bson:"purchase_price" json:"purchase_price"`
	WeightKg      float64         `bson:"weight_kg" json:"weight_kg"`
	VolumeM3      float64         `bson:"volume_m3" json:"volume_m3"`
	Attributes    map[string]any  `bson:"attributes" json:"attributes"`
	CreatedAt     time.Time       `bson:"created_at" json:"created_at"`
}

type Category struct {
	ID       string     `bson:"_id" json:"id"`
	ParentID string     `bson:"parent_id" json:"parent_id"`
	Name     string     `bson:"name" json:"name"`
	Children []Category `bson:"-" json:"children"`
}

type UoMConversion struct {
	FromUoM string  `bson:"from_uom" json:"from_uom"`
	ToUoM   string  `bson:"to_uom" json:"to_uom"`
	Factor  float64 `bson:"factor" json:"factor"`
}

func Convert(qty float64, from, to string, convs []UoMConversion) (float64, error) {
	if from == "" || to == "" || from == to {
		return qty, nil
	}
	for _, c := range convs {
		if c.Factor <= 0 {
			continue
		}
		if c.FromUoM == from && c.ToUoM == to {
			return qty * c.Factor, nil
		}
		if c.FromUoM == to && c.ToUoM == from {
			return qty / c.Factor, nil
		}
	}
	return 0, ErrNoConversion
}

func (p Product) ToStockQty(qty float64, from string) (float64, error) {
	base := p.StockUoM
	if base == "" {
		base = p.UnitOfMeasure
	}
	return Convert(qty, from, base, p.Conversions)
}

const (
	KindFinal      = "FINAL"
	KindSupport    = "SUPPORT"
	KindFixedAsset = "FIXED_ASSET"
	RoleComponent  = "COMPONENT"
	RoleSupport    = "SUPPORT"
)

type PriceHistory struct {
	ID             string    `json:"id"`
	ProductID      string    `json:"product_id"`
	SKU            string    `json:"sku"`
	PreviousPrice  float64   `json:"previous_price"`
	NewPrice       float64   `json:"new_price"`
	ReferenceDocID string    `json:"reference_doc_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type AssemblyItem struct {
	ID        string  `json:"id,omitempty"`
	ProductID string  `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	Role      string  `json:"role"`
	UnitPrice float64 `json:"unit_price"`
}

type Assembly struct {
	ID             string         `json:"id"`
	Code           string         `json:"code"`
	Name           string         `json:"name"`
	ProductID      string         `json:"product_id"`
	MarginPercent  float64        `json:"margin_percent"`
	Cost           float64        `json:"cost"`
	SuggestedPrice float64        `json:"suggested_price"`
	Active         bool           `json:"active"`
	Items          []AssemblyItem `json:"items"`
	CreatedAt      time.Time      `json:"created_at"`
}

type CatalogRepository interface {
	InsertSaleHistory(ctx context.Context, h PriceHistory) (PriceHistory, error)
	InsertPurchaseHistory(ctx context.Context, h PriceHistory) (PriceHistory, error)
	ListSaleHistory(ctx context.Context, productID string) ([]PriceHistory, error)
	ListPurchaseHistory(ctx context.Context, productID string) ([]PriceHistory, error)
	CreateAssembly(ctx context.Context, a Assembly) (Assembly, error)
	UpdateAssembly(ctx context.Context, a Assembly) error
	GetAssembly(ctx context.Context, id string) (Assembly, error)
	ListAssemblies(ctx context.Context) ([]Assembly, error)
	ProductInUse(ctx context.Context, productID string) (bool, error)
}

type Warehouse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

const (
	ReservationOpen      = "OPEN"
	ReservationReleased  = "RELEASED"
	ReservationConfirmed = "CONFIRMED"
)

type Reservation struct {
	ID           string    `json:"id"`
	SalesOrderID string    `json:"sales_order_id"`
	ProductID    string    `json:"product_id"`
	WarehouseID  string    `json:"warehouse_id,omitempty"`
	Quantity     float64   `json:"quantity"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Balance struct {
	ID                string    `json:"id"`
	ProductID         string    `json:"product_id"`
	WarehouseID       string    `json:"warehouse_id"`
	QuantityAvailable float64   `json:"quantity_available"`
	QuantityReserved  float64   `json:"quantity_reserved"`
	UpdatedAt         time.Time `json:"updated_at"`
}

const (
	SubtypePurchase = "PURCHASE"
	SubtypeSale     = "SALE"
	SubtypeLoss     = "LOSS"
	SubtypeTransfer = "TRANSFER"
)

type Movement struct {
	ID               string    `json:"id"`
	ProductID        string    `json:"product_id"`
	WarehouseID      string    `json:"warehouse_id"`
	MovementType     string    `json:"movement_type"`
	Subtype          string    `json:"subtype"`
	Quantity         float64   `json:"quantity"`
	ReferenceDocType string    `json:"reference_doc_type"`
	ReferenceDocID   string    `json:"reference_doc_id"`
	CreatedAt        time.Time `json:"created_at"`
}

// MovementFilter narrows ListMovements. A zero-value filter preserves the
// original "recent activity" behavior (most recent movements, capped) used by
// the stock MFE's movement feed; setting any field switches to an unbounded,
// fully-filtered query, e.g. reports-service's loss report (Subtype: LOSS).
type MovementFilter struct {
	ProductID   string
	WarehouseID string
	Subtype     string
	From        *time.Time
	To          *time.Time
}

// OrderItemComponent overrides one product a kit line consumes from stock — set only when
// the customer substituted an item at sale time. See ExpandKitItems.
type OrderItemComponent struct {
	ProductID string  `json:"product_id"`
	Quantity  float64 `json:"quantity"`
}

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Subtotal  float64 `json:"subtotal"`
	// Components is only ever populated on a kit line (ProductID is some assembly's own
	// linked product) that the customer customized. Empty means "use the assembly's own
	// recipe" — see ExpandKitItems, which is what actually reads this field.
	Components []OrderItemComponent `json:"components,omitempty"`
}

type OrderEvent struct {
	OrderID     string      `json:"order_id"`
	WarehouseID string      `json:"warehouse_id"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
}

// ExpandKitItems turns each order line into what's actually consumed from stock: a plain
// product passes through unchanged. A kit line — ProductID matches some assembly's own
// linked final product — expands into its components instead, since the kit itself is
// never a stocked item, only assembled from these at the moment of sale (mirrors
// bi-service's identical rule for purchase planning: kitByProductID/effectiveKitQty).
// The customer's own substitution list (Components) wins when set; otherwise it falls back
// to the assembly's registered recipe (both COMPONENT and SUPPORT roles — packaging is
// consumed same as any ingredient), scaled by the line's quantity.
func ExpandKitItems(items []OrderItem, assemblies []Assembly) []OrderItem {
	kitByProduct := make(map[string]Assembly, len(assemblies))
	for _, a := range assemblies {
		if a.ProductID != "" {
			kitByProduct[a.ProductID] = a
		}
	}
	out := make([]OrderItem, 0, len(items))
	for _, it := range items {
		a, isKit := kitByProduct[it.ProductID]
		if !isKit {
			out = append(out, it)
			continue
		}
		if len(it.Components) > 0 {
			for _, c := range it.Components {
				if c.ProductID == "" || c.Quantity <= 0 {
					continue
				}
				out = append(out, OrderItem{ProductID: c.ProductID, Quantity: c.Quantity})
			}
			continue
		}
		for _, ai := range a.Items {
			if ai.ProductID == "" || ai.Quantity <= 0 {
				continue
			}
			out = append(out, OrderItem{ProductID: ai.ProductID, Quantity: ai.Quantity * it.Quantity})
		}
	}
	return out
}

type InvoiceEvent struct {
	InvoiceID       string      `json:"invoice_id"`
	SalesOrderID    string      `json:"sales_order_id"`
	PurchaseOrderID string      `json:"purchase_order_id"`
	Direction       string      `json:"direction"`
	WarehouseID     string      `json:"warehouse_id"`
	AccessKey       string      `json:"access_key"`
	Items           []OrderItem `json:"items"`
}

type ProductRepository interface {
	Create(ctx context.Context, p Product) (Product, error)
	Update(ctx context.Context, p Product) error
	Get(ctx context.Context, id string) (Product, error)
	List(ctx context.Context) ([]Product, error)
	Delete(ctx context.Context, id string) error
	SetSalePrice(ctx context.Context, id string, price float64) error
	SetPurchasePrice(ctx context.Context, id string, price float64) error
	CountByCategory(ctx context.Context, categoryID string) (int64, error)
}

type CategoryRepository interface {
	Create(ctx context.Context, c Category) (Category, error)
	Update(ctx context.Context, c Category) error
	Get(ctx context.Context, id string) (Category, error)
	List(ctx context.Context) ([]Category, error)
	Delete(ctx context.Context, id string) error
	HasChildren(ctx context.Context, id string) (bool, error)
}

type StockRepository interface {
	CreateWarehouse(ctx context.Context, w Warehouse) (Warehouse, error)
	UpdateWarehouse(ctx context.Context, w Warehouse) error
	ListWarehouses(ctx context.Context) ([]Warehouse, error)
	GetWarehouse(ctx context.Context, id string) (Warehouse, error)
	UpsertBalance(ctx context.Context, productID, warehouseID string, available float64) (Balance, error)
	GetBalance(ctx context.Context, productID, warehouseID string) (Balance, error)
	ListBalances(ctx context.Context) ([]Balance, error)
	ReplaceReservations(ctx context.Context, orderID, warehouseID string, items []OrderItem) error
	ReleaseReservations(ctx context.Context, orderID string) error
	ConfirmReservations(ctx context.Context, orderID string) ([]Reservation, error)
	// ListOpenReservations returns every still-OPEN reservation for a product — the orders
	// actually holding the "Reservado" quantity shown against a balance, for the Saldos
	// screen's info icon.
	ListOpenReservations(ctx context.Context, productID string) ([]Reservation, error)
	Reserve(ctx context.Context, productID, warehouseID string, qty float64) error
	ConfirmSale(ctx context.Context, productID, warehouseID string, qty float64) error
	AddAvailable(ctx context.Context, productID, warehouseID string, qty float64) error
	RemoveAvailable(ctx context.Context, productID, warehouseID string, qty float64) error
	HasMovement(ctx context.Context, docType, docID, movementType string) (bool, error)
	ProductInUse(ctx context.Context, productID string) (bool, error)
	InsertMovement(ctx context.Context, m Movement) error
	ListMovements(ctx context.Context, f MovementFilter) ([]Movement, error)
}

type EventPublisher interface {
	PublishReserved(ctx context.Context, ev OrderEvent) error
}

type Locker interface {
	Lock(ctx context.Context, key string) (unlock func(context.Context), err error)
}

type Cache interface {
	SetBalance(ctx context.Context, b Balance) error
	GetBalance(ctx context.Context, productID, warehouseID string) (Balance, bool, error)
}
