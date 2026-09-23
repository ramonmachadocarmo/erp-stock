package httpadapter

import (
	"errors"
	"net/http"
	"time"

	"erp/pkg/httpserver"
	"erp/services/stock-service/internal/application"
	"erp/services/stock-service/internal/domain"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *application.Service
}

func New(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.Engine, jwt gin.HandlerFunc) {
	api := r.Group("/", jwt)
	api.GET("/products", h.listProducts)
	api.POST("/products", h.createProduct)
	api.GET("/products/:id", h.getProduct)
	api.PUT("/products/:id", h.updateProduct)
	api.DELETE("/products/:id", h.deleteProduct)
	api.GET("/categories", h.listCategories)
	api.POST("/categories", h.createCategory)
	api.PUT("/categories/:id", h.updateCategory)
	api.DELETE("/categories/:id", h.deleteCategory)
	api.GET("/warehouses", h.listWarehouses)
	api.POST("/warehouses", h.createWarehouse)
	api.PUT("/warehouses/:id", h.updateWarehouse)
	api.GET("/balances", h.listBalances)
	api.PUT("/balances", h.setBalance)
	api.GET("/movements", h.listMovements)
	api.POST("/movements", h.createMovement)
	api.POST("/movements/transfer", h.transferStock)
	api.GET("/reservations", h.listOpenReservations)
	api.POST("/purchases/receive", h.receivePurchase)
	api.GET("/prices/sale", h.listSalePrices)
	api.POST("/prices/sale", h.createSalePrice)
	api.GET("/prices/purchase", h.listPurchasePrices)
	api.POST("/prices/purchase", h.createPurchasePrice)
	api.GET("/assemblies", h.listAssemblies)
	api.POST("/assemblies", h.createAssembly)
	api.GET("/assemblies/:id", h.getAssembly)
	api.PUT("/assemblies/:id", h.updateAssembly)
	api.POST("/assemblies/:id/recalculate", h.recalculateAssembly)
	api.POST("/assemblies/:id/apply-price", h.applyAssemblyPrice)
}

func (h *Handler) listProducts(c *gin.Context) {
	out, err := h.svc.ListProducts(c.Request.Context())
	if err != nil {
		httpserver.Error(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) createProduct(c *gin.Context) {
	var p domain.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	p.CreatedAt = time.Now().UTC()
	out, err := h.svc.CreateProduct(c.Request.Context(), p)
	if err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) getProduct(c *gin.Context) {
	out, err := h.svc.GetProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpserver.Error(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) updateProduct(c *gin.Context) {
	var p domain.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	p.ID = c.Param("id")
	if err := h.svc.UpdateProduct(c.Request.Context(), p); err != nil {
		if errors.Is(err, domain.ErrNoConversion) || errors.Is(err, domain.ErrInvalid) {
			httpserver.Error(c, http.StatusBadRequest, err)
			return
		}
		httpserver.Error(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) listCategories(c *gin.Context) {
	out, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		httpserver.Error(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) createCategory(c *gin.Context) {
	var cat domain.Category
	if err := c.ShouldBindJSON(&cat); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	out, err := h.svc.CreateCategory(c.Request.Context(), cat)
	if err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) updateCategory(c *gin.Context) {
	var cat domain.Category
	if err := c.ShouldBindJSON(&cat); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	cat.ID = c.Param("id")
	if err := h.svc.UpdateCategory(c.Request.Context(), cat); err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusOK, cat)
}

func (h *Handler) deleteCategory(c *gin.Context) {
	if err := h.svc.DeleteCategory(c.Request.Context(), c.Param("id")); err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) deleteProduct(c *gin.Context) {
	if err := h.svc.DeleteProduct(c.Request.Context(), c.Param("id")); err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listWarehouses(c *gin.Context) {
	out, err := h.svc.ListWarehouses(c.Request.Context())
	if err != nil {
		httpserver.Error(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) createWarehouse(c *gin.Context) {
	var w domain.Warehouse
	if err := c.ShouldBindJSON(&w); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	out, err := h.svc.CreateWarehouse(c.Request.Context(), w)
	if err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) updateWarehouse(c *gin.Context) {
	var w domain.Warehouse
	if err := c.ShouldBindJSON(&w); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	w.ID = c.Param("id")
	if err := h.svc.UpdateWarehouse(c.Request.Context(), w); err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusOK, w)
}

func (h *Handler) listBalances(c *gin.Context) {
	out, err := h.svc.ListBalances(c.Request.Context())
	if err != nil {
		httpserver.Error(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

type setBalanceIn struct {
	ProductID   string  `json:"product_id" binding:"required"`
	WarehouseID string  `json:"warehouse_id" binding:"required"`
	Quantity    float64 `json:"quantity"`
}

func (h *Handler) setBalance(c *gin.Context) {
	var in setBalanceIn
	if err := c.ShouldBindJSON(&in); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	out, err := h.svc.SetBalance(c.Request.Context(), in.ProductID, in.WarehouseID, in.Quantity)
	if err != nil {
		httpserver.Error(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) listMovements(c *gin.Context) {
	f := domain.MovementFilter{
		ProductID:   c.Query("product_id"),
		WarehouseID: c.Query("warehouse_id"),
		Subtype:     c.Query("subtype"),
	}
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = &t
		}
	}
	out, err := h.svc.ListMovements(c.Request.Context(), f)
	if err != nil {
		httpserver.Error(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) listOpenReservations(c *gin.Context) {
	out, err := h.svc.ListOpenReservations(c.Request.Context(), c.Query("product_id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, domain.ErrInvalid) {
			status = http.StatusBadRequest
		}
		httpserver.Error(c, status, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

type movementIn struct {
	ProductID   string  `json:"product_id" binding:"required"`
	WarehouseID string  `json:"warehouse_id" binding:"required"`
	Direction   string  `json:"direction" binding:"required"`
	Subtype     string  `json:"subtype"`
	Quantity    float64 `json:"quantity" binding:"required"`
}

type transferIn struct {
	ProductID       string  `json:"product_id" binding:"required"`
	FromWarehouseID string  `json:"from_warehouse_id" binding:"required"`
	ToWarehouseID   string  `json:"to_warehouse_id" binding:"required"`
	Quantity        float64 `json:"quantity" binding:"required"`
}

func (h *Handler) transferStock(c *gin.Context) {
	var in transferIn
	if err := c.ShouldBindJSON(&in); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.TransferStock(c.Request.Context(), in.ProductID, in.FromWarehouseID, in.ToWarehouseID, in.Quantity); err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

func (h *Handler) createMovement(c *gin.Context) {
	var in movementIn
	if err := c.ShouldBindJSON(&in); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.CreateMovement(c.Request.Context(), in.ProductID, in.WarehouseID, in.Direction, in.Subtype, in.Quantity); err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

type receivePurchaseIn struct {
	OrderID     string             `json:"order_id" binding:"required"`
	WarehouseID string             `json:"warehouse_id" binding:"required"`
	Items       []domain.OrderItem `json:"items"`
}

func (h *Handler) receivePurchase(c *gin.Context) {
	var in receivePurchaseIn
	if err := c.ShouldBindJSON(&in); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.ReceivePurchase(c.Request.Context(), in.OrderID, in.WarehouseID, in.Items); err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

type priceIn struct {
	ProductID      string  `json:"product_id" binding:"required"`
	NewPrice       float64 `json:"new_price"`
	ReferenceDocID string  `json:"reference_doc_id"`
}

func (h *Handler) createSalePrice(c *gin.Context) {
	var in priceIn
	if err := c.ShouldBindJSON(&in); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	out, err := h.svc.RegisterSalePrice(c.Request.Context(), in.ProductID, in.NewPrice)
	if err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) listSalePrices(c *gin.Context) {
	out, err := h.svc.ListSaleHistory(c.Request.Context(), c.Query("product_id"))
	if err != nil {
		httpserver.Error(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) createPurchasePrice(c *gin.Context) {
	var in priceIn
	if err := c.ShouldBindJSON(&in); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	out, err := h.svc.RegisterPurchasePrice(c.Request.Context(), in.ProductID, in.NewPrice, in.ReferenceDocID)
	if err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) listPurchasePrices(c *gin.Context) {
	out, err := h.svc.ListPurchaseHistory(c.Request.Context(), c.Query("product_id"))
	if err != nil {
		httpserver.Error(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) listAssemblies(c *gin.Context) {
	out, err := h.svc.ListAssemblies(c.Request.Context())
	if err != nil {
		httpserver.Error(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) createAssembly(c *gin.Context) {
	var a domain.Assembly
	if err := c.ShouldBindJSON(&a); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	out, err := h.svc.CreateAssembly(c.Request.Context(), a)
	if err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) getAssembly(c *gin.Context) {
	out, err := h.svc.GetAssembly(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) updateAssembly(c *gin.Context) {
	var a domain.Assembly
	if err := c.ShouldBindJSON(&a); err != nil {
		httpserver.Error(c, http.StatusBadRequest, err)
		return
	}
	a.ID = c.Param("id")
	out, err := h.svc.UpdateAssembly(c.Request.Context(), a)
	if err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) recalculateAssembly(c *gin.Context) {
	out, err := h.svc.RecalculateAssembly(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) applyAssemblyPrice(c *gin.Context) {
	out, err := h.svc.ApplyAssemblyPrice(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpserver.Error(c, statusFrom(err), err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func statusFrom(err error) int {
	if errors.Is(err, domain.ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, domain.ErrInUse) || errors.Is(err, domain.ErrConflict) || errors.Is(err, domain.ErrInsufficient) {
		return http.StatusConflict
	}
	if errors.Is(err, domain.ErrInvalid) || errors.Is(err, domain.ErrNoConversion) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
