# Phase 3: Remaining Entities

Add all remaining models and handlers: BananaTree (with lifecycle status), Bunch, Banana (with hand_number), Tool, and Worker. Then wire up nested routes.

By the end of this phase, every entity has full CRUD and you can navigate the full relationship tree: Farm → Tree → Bunch → Banana.

---

## Step 16: Create `models/banana_tree.go`

**Delete `models/stubs.go`** — we're replacing all stubs with real models now.

```go
package models

import "time"

// BananaTree lifecycle statuses.
// A banana tree goes through these stages and dies after one harvest cycle.
const (
	TreeStatusPlanted   = "planted"
	TreeStatusGrowing   = "growing"
	TreeStatusFlowering = "flowering"
	TreeStatusFruiting  = "fruiting"
	TreeStatusHarvested = "harvested"
	TreeStatusDead      = "dead"
)

// BananaTree health statuses.
const (
	TreeHealthHealthy     = "healthy"
	TreeHealthDiseased    = "diseased"
	TreeHealthPestDamaged = "pest_damaged"
)

// BananaTree varieties.
const (
	VarietyCavendish  = "cavendish"
	VarietyPlantain   = "plantain"
	VarietyLadyFinger = "lady_finger"
	VarietyRed        = "red"
	VarietyBlueJava   = "blue_java"
)

// ValidTreeStatuses is the list of allowed tree statuses for validation.
var ValidTreeStatuses = []string{
	TreeStatusPlanted, TreeStatusGrowing, TreeStatusFlowering,
	TreeStatusFruiting, TreeStatusHarvested, TreeStatusDead,
}

// ValidTreeHealths is the list of allowed tree health values.
var ValidTreeHealths = []string{
	TreeHealthHealthy, TreeHealthDiseased, TreeHealthPestDamaged,
}

// ValidVarieties is the list of allowed banana tree varieties.
var ValidVarieties = []string{
	VarietyCavendish, VarietyPlantain, VarietyLadyFinger,
	VarietyRed, VarietyBlueJava,
}

// BananaTree represents a banana tree planted on a farm.
type BananaTree struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	FarmID    uint      `json:"farm_id" gorm:"not null;index" validate:"required"`
	Variety   string    `json:"variety" gorm:"not null" validate:"required,oneof=cavendish plantain lady_finger red blue_java"`
	PlantedAt time.Time `json:"planted_at" gorm:"not null" validate:"required"`
	Status    string    `json:"status" gorm:"not null;default:planted" validate:"required,oneof=planted growing flowering fruiting harvested dead"`
	Health    string    `json:"health" gorm:"not null;default:healthy" validate:"required,oneof=healthy diseased pest_damaged"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	// constraint:OnDelete:CASCADE — deleting a Farm deletes all its BananaTrees
	Farm    Farm     `json:"-" gorm:"foreignKey:FarmID;constraint:OnDelete:CASCADE"`
	Bunches []Bunch  `json:"bunches,omitempty" gorm:"foreignKey:BananaTreeID"`
}
```

### Key concepts:
- **Constants for enums** — Go doesn't have enums like TypeScript. The convention is string constants + validation via `oneof=` tag
- **`gorm:"index"`** on FarmID — creates a database index for faster queries when filtering trees by farm
- **`json:"-"`** on `Farm` — hides the parent from JSON output. Without this, loading a tree would recursively include the entire farm object. You already have `farm_id` in the response
- **`default:planted`** — GORM sets this as the column default in SQLite. New trees start as "planted"

---

## Step 17: Create `models/bunch.go`

```go
package models

import "time"

// Bunch represents a bunch of bananas on a tree.
// Each banana tree produces one or more bunches during its fruiting cycle.
type Bunch struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	BananaTreeID uint       `json:"banana_tree_id" gorm:"not null;index" validate:"required"`
	HarvestedAt  *time.Time `json:"harvested_at"` // nil means not yet harvested
	WeightKg     float64    `json:"weight_kg" gorm:"not null;default:0" validate:"gte=0"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relationships
	// constraint:OnDelete:CASCADE — deleting a BananaTree deletes all its Bunches
	BananaTree BananaTree `json:"-" gorm:"foreignKey:BananaTreeID;constraint:OnDelete:CASCADE"`
	Bananas    []Banana   `json:"bananas,omitempty" gorm:"foreignKey:BunchID"`
}
```

### Key concepts:
- **`*time.Time`** (pointer) for `HarvestedAt` — this is nullable. `nil` means the bunch hasn't been harvested yet. A non-nil value means it has been. In the JSON response, `null` vs a date string
- **`gorm:"default:0"`** on WeightKg — new bunches start at 0 weight until measured

---

## Step 18: Create `models/banana.go`

```go
package models

import "time"

// Banana ripeness levels.
const (
	RipenessGreen    = "green"
	RipenessTurning  = "turning"
	RipenessRipe     = "ripe"
	RipenessOverripe = "overripe"
)

// Banana sizes.
const (
	SizeSmall  = "small"
	SizeMedium = "medium"
	SizeLarge  = "large"
)

// Banana represents an individual banana in a bunch.
type Banana struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	BunchID     uint      `json:"bunch_id" gorm:"not null;index" validate:"required"`
	HandNumber  int       `json:"hand_number" gorm:"not null" validate:"required,gte=1,lte=20"`
	Size        string    `json:"size" gorm:"not null;default:medium" validate:"required,oneof=small medium large"`
	Ripeness    string    `json:"ripeness" gorm:"not null;default:green" validate:"required,oneof=green turning ripe overripe"`
	WeightGrams float64   `json:"weight_grams" gorm:"not null;default:0" validate:"gte=0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	// constraint:OnDelete:CASCADE — deleting a Bunch deletes all its Bananas
	Bunch Bunch `json:"-" gorm:"foreignKey:BunchID;constraint:OnDelete:CASCADE"`
}
```

### Key concepts:
- **`HandNumber`** — bananas grow in "hands" (clusters) on a bunch. Typically 5-20 hands per bunch. The `gte=1,lte=20` validation enforces this range
- **Defaults** — new bananas start as `medium`, `green`, 0 weight

---

## Step 19: Create `models/tool.go`

```go
package models

import "time"

// Tool types found on a banana farm.
const (
	ToolTypeMachete          = "machete"
	ToolTypePruningShears    = "pruning_shears"
	ToolTypeIrrigationPump   = "irrigation_pump"
	ToolTypeFertilizerSprayer = "fertilizer_sprayer"
	ToolTypeHarvestingKnife  = "harvesting_knife"
	ToolTypeWheelbarrow      = "wheelbarrow"
	ToolTypeBunchCover       = "bunch_cover"
)

// Tool conditions.
const (
	ConditionNew    = "new"
	ConditionGood   = "good"
	ConditionWorn   = "worn"
	ConditionBroken = "broken"
)

// Tool represents a tool used on a banana farm.
type Tool struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	FarmID    uint      `json:"farm_id" gorm:"not null;index" validate:"required"`
	Name      string    `json:"name" gorm:"not null" validate:"required,min=1,max=100"`
	Type      string    `json:"type" gorm:"not null" validate:"required,oneof=machete pruning_shears irrigation_pump fertilizer_sprayer harvesting_knife wheelbarrow bunch_cover"`
	Condition string    `json:"condition" gorm:"not null;default:new" validate:"required,oneof=new good worn broken"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	// constraint:OnDelete:CASCADE — deleting a Farm deletes all its Tools
	Farm Farm `json:"-" gorm:"foreignKey:FarmID;constraint:OnDelete:CASCADE"`
}
```

---

## Step 20: Create `models/worker.go`

```go
package models

import "time"

// Worker roles on a banana farm.
const (
	RoleFarmer     = "farmer"
	RoleHarvester  = "harvester"
	RoleIrrigator  = "irrigator"
	RoleSupervisor = "supervisor"
)

// Worker represents a person working on a banana farm.
type Worker struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	FarmID    uint      `json:"farm_id" gorm:"not null;index" validate:"required"`
	Name      string    `json:"name" gorm:"not null" validate:"required,min=1,max=100"`
	Role      string    `json:"role" gorm:"not null" validate:"required,oneof=farmer harvester irrigator supervisor"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	// constraint:OnDelete:CASCADE — deleting a Farm deletes all its Workers
	Farm Farm `json:"-" gorm:"foreignKey:FarmID;constraint:OnDelete:CASCADE"`
}
```

---

## Step 21: Create `handlers/banana_tree.go`

This follows the exact same pattern as `handlers/farm.go`. Here's the full implementation:

```go
package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
)

type BananaTreeHandler struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewBananaTreeHandler(db *gorm.DB) *BananaTreeHandler {
	return &BananaTreeHandler{
		DB:       db,
		Validate: validator.New(),
	}
}

type CreateBananaTreeRequest struct {
	FarmID  uint   `json:"farm_id" validate:"required"`
	Variety string `json:"variety" validate:"required,oneof=cavendish plantain lady_finger red blue_java"`
	PlantedAt string `json:"planted_at" validate:"required"`
	Status  string `json:"status" validate:"omitempty,oneof=planted growing flowering fruiting harvested dead"`
	Health  string `json:"health" validate:"omitempty,oneof=healthy diseased pest_damaged"`
}

type UpdateBananaTreeRequest struct {
	Status *string `json:"status" validate:"omitempty,oneof=planted growing flowering fruiting harvested dead"`
	Health *string `json:"health" validate:"omitempty,oneof=healthy diseased pest_damaged"`
}

// List handles GET /trees
// Supports filtering: ?status=flowering&variety=cavendish&farm_id=1
func (h *BananaTreeHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := helpers.ParsePagination(r)

	var trees []models.BananaTree
	var total int64

	query := h.DB.Model(&models.BananaTree{})

	// Filters
	if status := r.URL.Query().Get("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if variety := r.URL.Query().Get("variety"); variety != "" {
		query = query.Where("variety = ?", variety)
	}
	if farmID := r.URL.Query().Get("farm_id"); farmID != "" {
		query = query.Where("farm_id = ?", farmID)
	}

	query.Count(&total)

	result := query.Offset(pagination.Offset()).Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&trees)

	if result.Error != nil {
		slog.Error("failed to list trees", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to list trees")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, helpers.NewPaginatedResponse(trees, total, pagination))
}

// Create handles POST /trees
func (h *BananaTreeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateBananaTreeRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	// Verify the farm exists
	var farm models.Farm
	if result := h.DB.First(&farm, req.FarmID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusBadRequest, "farm not found")
			return
		}
		slog.Error("failed to find farm", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to verify farm")
		return
	}

	plantedAt, err := ParseDate(req.PlantedAt)
	if err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid date format for planted_at, use YYYY-MM-DD")
		return
	}

	tree := models.BananaTree{
		FarmID:    req.FarmID,
		Variety:   req.Variety,
		PlantedAt: plantedAt,
		Status:    req.Status,
		Health:    req.Health,
	}

	// Apply defaults if not provided
	if tree.Status == "" {
		tree.Status = models.TreeStatusPlanted
	}
	if tree.Health == "" {
		tree.Health = models.TreeHealthHealthy
	}

	if result := h.DB.Create(&tree); result.Error != nil {
		slog.Error("failed to create tree", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to create tree")
		return
	}

	slog.Info("tree created", "id", tree.ID, "farm_id", tree.FarmID, "variety", tree.Variety)
	helpers.RespondJSON(w, http.StatusCreated, tree)
}

// Get handles GET /trees/{id}
func (h *BananaTreeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var tree models.BananaTree
	result := h.DB.First(&tree, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "tree not found")
			return
		}
		slog.Error("failed to get tree", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to get tree")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, tree)
}

// Update handles PUT /trees/{id}
// Typically used to update lifecycle status and health.
func (h *BananaTreeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var tree models.BananaTree
	if result := h.DB.First(&tree, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "tree not found")
			return
		}
		slog.Error("failed to find tree", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find tree")
		return
	}

	var req UpdateBananaTreeRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	if req.Status != nil {
		tree.Status = *req.Status
	}
	if req.Health != nil {
		tree.Health = *req.Health
	}

	if result := h.DB.Save(&tree); result.Error != nil {
		slog.Error("failed to update tree", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to update tree")
		return
	}

	slog.Info("tree updated", "id", tree.ID, "status", tree.Status)
	helpers.RespondJSON(w, http.StatusOK, tree)
}

// Delete handles DELETE /trees/{id}
func (h *BananaTreeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var tree models.BananaTree
	if result := h.DB.First(&tree, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "tree not found")
			return
		}
		slog.Error("failed to find tree", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find tree")
		return
	}

	if result := h.DB.Delete(&tree); result.Error != nil {
		slog.Error("failed to delete tree", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to delete tree")
		return
	}

	slog.Info("tree deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

// ListBunches handles GET /trees/{id}/bunches
func (h *BananaTreeHandler) ListBunches(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pagination := helpers.ParsePagination(r)

	// Verify tree exists
	var tree models.BananaTree
	if result := h.DB.First(&tree, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "tree not found")
			return
		}
		slog.Error("failed to find tree", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find tree")
		return
	}

	var bunches []models.Bunch
	var total int64

	query := h.DB.Model(&models.Bunch{}).Where("banana_tree_id = ?", id)
	query.Count(&total)

	result := query.Offset(pagination.Offset()).Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&bunches)

	if result.Error != nil {
		slog.Error("failed to list bunches", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to list bunches")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, helpers.NewPaginatedResponse(bunches, total, pagination))
}
```

### Important pattern — verifying foreign keys:
```go
// Verify the farm exists before creating a tree
var farm models.Farm
if result := h.DB.First(&farm, req.FarmID); result.Error != nil {
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        helpers.RespondError(w, http.StatusBadRequest, "farm not found")
```
Always verify that the parent entity exists before creating a child. SQLite has foreign key support but GORM doesn't enable it by default, so we do this check manually. This gives the client a clear error message instead of a cryptic database constraint error.

---

## Step 22: Create `handlers/bunch.go`

```go
package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
)

type BunchHandler struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewBunchHandler(db *gorm.DB) *BunchHandler {
	return &BunchHandler{
		DB:       db,
		Validate: validator.New(),
	}
}

type CreateBunchRequest struct {
	BananaTreeID uint    `json:"banana_tree_id" validate:"required"`
	WeightKg     float64 `json:"weight_kg" validate:"gte=0"`
}

type UpdateBunchRequest struct {
	HarvestedAt *string  `json:"harvested_at"` // "YYYY-MM-DD" or null
	WeightKg    *float64 `json:"weight_kg" validate:"omitempty,gte=0"`
}

func (h *BunchHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := helpers.ParsePagination(r)

	var bunches []models.Bunch
	var total int64

	query := h.DB.Model(&models.Bunch{})

	if treeID := r.URL.Query().Get("banana_tree_id"); treeID != "" {
		query = query.Where("banana_tree_id = ?", treeID)
	}

	query.Count(&total)

	result := query.Offset(pagination.Offset()).Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&bunches)

	if result.Error != nil {
		slog.Error("failed to list bunches", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to list bunches")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, helpers.NewPaginatedResponse(bunches, total, pagination))
}

func (h *BunchHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateBunchRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	// Verify tree exists
	var tree models.BananaTree
	if result := h.DB.First(&tree, req.BananaTreeID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusBadRequest, "banana tree not found")
			return
		}
		slog.Error("failed to find tree", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to verify tree")
		return
	}

	bunch := models.Bunch{
		BananaTreeID: req.BananaTreeID,
		WeightKg:     req.WeightKg,
	}

	if result := h.DB.Create(&bunch); result.Error != nil {
		slog.Error("failed to create bunch", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to create bunch")
		return
	}

	slog.Info("bunch created", "id", bunch.ID, "tree_id", bunch.BananaTreeID)
	helpers.RespondJSON(w, http.StatusCreated, bunch)
}

func (h *BunchHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var bunch models.Bunch
	result := h.DB.First(&bunch, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "bunch not found")
			return
		}
		slog.Error("failed to get bunch", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to get bunch")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, bunch)
}

func (h *BunchHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var bunch models.Bunch
	if result := h.DB.First(&bunch, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "bunch not found")
			return
		}
		slog.Error("failed to find bunch", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find bunch")
		return
	}

	var req UpdateBunchRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	if req.HarvestedAt != nil {
		t, err := ParseDate(*req.HarvestedAt)
		if err != nil {
			helpers.RespondError(w, http.StatusBadRequest, "invalid date format for harvested_at, use YYYY-MM-DD")
			return
		}
		bunch.HarvestedAt = &t
	}
	if req.WeightKg != nil {
		bunch.WeightKg = *req.WeightKg
	}

	if result := h.DB.Save(&bunch); result.Error != nil {
		slog.Error("failed to update bunch", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to update bunch")
		return
	}

	slog.Info("bunch updated", "id", bunch.ID)
	helpers.RespondJSON(w, http.StatusOK, bunch)
}

func (h *BunchHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var bunch models.Bunch
	if result := h.DB.First(&bunch, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "bunch not found")
			return
		}
		slog.Error("failed to find bunch", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find bunch")
		return
	}

	if result := h.DB.Delete(&bunch); result.Error != nil {
		slog.Error("failed to delete bunch", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to delete bunch")
		return
	}

	slog.Info("bunch deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

// ListBananas handles GET /bunches/{id}/bananas
func (h *BunchHandler) ListBananas(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pagination := helpers.ParsePagination(r)

	var bunch models.Bunch
	if result := h.DB.First(&bunch, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "bunch not found")
			return
		}
		slog.Error("failed to find bunch", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find bunch")
		return
	}

	var bananas []models.Banana
	var total int64

	query := h.DB.Model(&models.Banana{}).Where("bunch_id = ?", id)
	query.Count(&total)

	result := query.Offset(pagination.Offset()).Limit(pagination.Limit).
		Order("hand_number ASC").
		Find(&bananas)

	if result.Error != nil {
		slog.Error("failed to list bananas", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to list bananas")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, helpers.NewPaginatedResponse(bananas, total, pagination))
}
```

---

## Step 23: Create `handlers/banana.go`

```go
package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
)

type BananaHandler struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewBananaHandler(db *gorm.DB) *BananaHandler {
	return &BananaHandler{
		DB:       db,
		Validate: validator.New(),
	}
}

type CreateBananaRequest struct {
	BunchID     uint    `json:"bunch_id" validate:"required"`
	HandNumber  int     `json:"hand_number" validate:"required,gte=1,lte=20"`
	Size        string  `json:"size" validate:"omitempty,oneof=small medium large"`
	Ripeness    string  `json:"ripeness" validate:"omitempty,oneof=green turning ripe overripe"`
	WeightGrams float64 `json:"weight_grams" validate:"gte=0"`
}

type UpdateBananaRequest struct {
	Size        *string  `json:"size" validate:"omitempty,oneof=small medium large"`
	Ripeness    *string  `json:"ripeness" validate:"omitempty,oneof=green turning ripe overripe"`
	WeightGrams *float64 `json:"weight_grams" validate:"omitempty,gte=0"`
}

func (h *BananaHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := helpers.ParsePagination(r)

	var bananas []models.Banana
	var total int64

	query := h.DB.Model(&models.Banana{})

	if ripeness := r.URL.Query().Get("ripeness"); ripeness != "" {
		query = query.Where("ripeness = ?", ripeness)
	}
	if size := r.URL.Query().Get("size"); size != "" {
		query = query.Where("size = ?", size)
	}
	if bunchID := r.URL.Query().Get("bunch_id"); bunchID != "" {
		query = query.Where("bunch_id = ?", bunchID)
	}

	query.Count(&total)

	result := query.Offset(pagination.Offset()).Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&bananas)

	if result.Error != nil {
		slog.Error("failed to list bananas", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to list bananas")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, helpers.NewPaginatedResponse(bananas, total, pagination))
}

func (h *BananaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateBananaRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	// Verify bunch exists
	var bunch models.Bunch
	if result := h.DB.First(&bunch, req.BunchID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusBadRequest, "bunch not found")
			return
		}
		slog.Error("failed to find bunch", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to verify bunch")
		return
	}

	banana := models.Banana{
		BunchID:     req.BunchID,
		HandNumber:  req.HandNumber,
		Size:        req.Size,
		Ripeness:    req.Ripeness,
		WeightGrams: req.WeightGrams,
	}

	if banana.Size == "" {
		banana.Size = models.SizeMedium
	}
	if banana.Ripeness == "" {
		banana.Ripeness = models.RipenessGreen
	}

	if result := h.DB.Create(&banana); result.Error != nil {
		slog.Error("failed to create banana", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to create banana")
		return
	}

	slog.Info("banana created", "id", banana.ID, "bunch_id", banana.BunchID)
	helpers.RespondJSON(w, http.StatusCreated, banana)
}

func (h *BananaHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var banana models.Banana
	result := h.DB.First(&banana, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "banana not found")
			return
		}
		slog.Error("failed to get banana", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to get banana")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, banana)
}

func (h *BananaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var banana models.Banana
	if result := h.DB.First(&banana, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "banana not found")
			return
		}
		slog.Error("failed to find banana", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find banana")
		return
	}

	var req UpdateBananaRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	if req.Size != nil {
		banana.Size = *req.Size
	}
	if req.Ripeness != nil {
		banana.Ripeness = *req.Ripeness
	}
	if req.WeightGrams != nil {
		banana.WeightGrams = *req.WeightGrams
	}

	if result := h.DB.Save(&banana); result.Error != nil {
		slog.Error("failed to update banana", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to update banana")
		return
	}

	slog.Info("banana updated", "id", banana.ID)
	helpers.RespondJSON(w, http.StatusOK, banana)
}

func (h *BananaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var banana models.Banana
	if result := h.DB.First(&banana, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "banana not found")
			return
		}
		slog.Error("failed to find banana", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find banana")
		return
	}

	if result := h.DB.Delete(&banana); result.Error != nil {
		slog.Error("failed to delete banana", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to delete banana")
		return
	}

	slog.Info("banana deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}
```

---

## Step 24: Create `handlers/tool.go`

```go
package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
)

type ToolHandler struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewToolHandler(db *gorm.DB) *ToolHandler {
	return &ToolHandler{
		DB:       db,
		Validate: validator.New(),
	}
}

type CreateToolRequest struct {
	FarmID    uint   `json:"farm_id" validate:"required"`
	Name      string `json:"name" validate:"required,min=1,max=100"`
	Type      string `json:"type" validate:"required,oneof=machete pruning_shears irrigation_pump fertilizer_sprayer harvesting_knife wheelbarrow bunch_cover"`
	Condition string `json:"condition" validate:"omitempty,oneof=new good worn broken"`
}

type UpdateToolRequest struct {
	Name      *string `json:"name" validate:"omitempty,min=1,max=100"`
	Condition *string `json:"condition" validate:"omitempty,oneof=new good worn broken"`
}

func (h *ToolHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := helpers.ParsePagination(r)

	var tools []models.Tool
	var total int64

	query := h.DB.Model(&models.Tool{})

	if toolType := r.URL.Query().Get("type"); toolType != "" {
		query = query.Where("type = ?", toolType)
	}
	if condition := r.URL.Query().Get("condition"); condition != "" {
		query = query.Where("condition = ?", condition)
	}
	if farmID := r.URL.Query().Get("farm_id"); farmID != "" {
		query = query.Where("farm_id = ?", farmID)
	}

	query.Count(&total)

	result := query.Offset(pagination.Offset()).Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&tools)

	if result.Error != nil {
		slog.Error("failed to list tools", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to list tools")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, helpers.NewPaginatedResponse(tools, total, pagination))
}

func (h *ToolHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateToolRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	// Verify farm exists
	var farm models.Farm
	if result := h.DB.First(&farm, req.FarmID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusBadRequest, "farm not found")
			return
		}
		slog.Error("failed to find farm", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to verify farm")
		return
	}

	tool := models.Tool{
		FarmID:    req.FarmID,
		Name:      req.Name,
		Type:      req.Type,
		Condition: req.Condition,
	}

	if tool.Condition == "" {
		tool.Condition = models.ConditionNew
	}

	if result := h.DB.Create(&tool); result.Error != nil {
		slog.Error("failed to create tool", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to create tool")
		return
	}

	slog.Info("tool created", "id", tool.ID, "name", tool.Name)
	helpers.RespondJSON(w, http.StatusCreated, tool)
}

func (h *ToolHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var tool models.Tool
	result := h.DB.First(&tool, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "tool not found")
			return
		}
		slog.Error("failed to get tool", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to get tool")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, tool)
}

func (h *ToolHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var tool models.Tool
	if result := h.DB.First(&tool, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "tool not found")
			return
		}
		slog.Error("failed to find tool", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find tool")
		return
	}

	var req UpdateToolRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	if req.Name != nil {
		tool.Name = *req.Name
	}
	if req.Condition != nil {
		tool.Condition = *req.Condition
	}

	if result := h.DB.Save(&tool); result.Error != nil {
		slog.Error("failed to update tool", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to update tool")
		return
	}

	slog.Info("tool updated", "id", tool.ID)
	helpers.RespondJSON(w, http.StatusOK, tool)
}

func (h *ToolHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var tool models.Tool
	if result := h.DB.First(&tool, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "tool not found")
			return
		}
		slog.Error("failed to find tool", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find tool")
		return
	}

	if result := h.DB.Delete(&tool); result.Error != nil {
		slog.Error("failed to delete tool", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to delete tool")
		return
	}

	slog.Info("tool deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}
```

---

## Step 25: Create `handlers/worker.go`

```go
package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
)

type WorkerHandler struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewWorkerHandler(db *gorm.DB) *WorkerHandler {
	return &WorkerHandler{
		DB:       db,
		Validate: validator.New(),
	}
}

type CreateWorkerRequest struct {
	FarmID uint   `json:"farm_id" validate:"required"`
	Name   string `json:"name" validate:"required,min=1,max=100"`
	Role   string `json:"role" validate:"required,oneof=farmer harvester irrigator supervisor"`
}

type UpdateWorkerRequest struct {
	Name *string `json:"name" validate:"omitempty,min=1,max=100"`
	Role *string `json:"role" validate:"omitempty,oneof=farmer harvester irrigator supervisor"`
}

func (h *WorkerHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := helpers.ParsePagination(r)

	var workers []models.Worker
	var total int64

	query := h.DB.Model(&models.Worker{})

	if role := r.URL.Query().Get("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	if farmID := r.URL.Query().Get("farm_id"); farmID != "" {
		query = query.Where("farm_id = ?", farmID)
	}

	query.Count(&total)

	result := query.Offset(pagination.Offset()).Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&workers)

	if result.Error != nil {
		slog.Error("failed to list workers", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to list workers")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, helpers.NewPaginatedResponse(workers, total, pagination))
}

func (h *WorkerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkerRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	// Verify farm exists
	var farm models.Farm
	if result := h.DB.First(&farm, req.FarmID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusBadRequest, "farm not found")
			return
		}
		slog.Error("failed to find farm", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to verify farm")
		return
	}

	worker := models.Worker{
		FarmID: req.FarmID,
		Name:   req.Name,
		Role:   req.Role,
	}

	if result := h.DB.Create(&worker); result.Error != nil {
		slog.Error("failed to create worker", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to create worker")
		return
	}

	slog.Info("worker created", "id", worker.ID, "name", worker.Name)
	helpers.RespondJSON(w, http.StatusCreated, worker)
}

func (h *WorkerHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var worker models.Worker
	result := h.DB.First(&worker, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "worker not found")
			return
		}
		slog.Error("failed to get worker", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to get worker")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, worker)
}

func (h *WorkerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var worker models.Worker
	if result := h.DB.First(&worker, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "worker not found")
			return
		}
		slog.Error("failed to find worker", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find worker")
		return
	}

	var req UpdateWorkerRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	if req.Name != nil {
		worker.Name = *req.Name
	}
	if req.Role != nil {
		worker.Role = *req.Role
	}

	if result := h.DB.Save(&worker); result.Error != nil {
		slog.Error("failed to update worker", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to update worker")
		return
	}

	slog.Info("worker updated", "id", worker.ID)
	helpers.RespondJSON(w, http.StatusOK, worker)
}

func (h *WorkerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var worker models.Worker
	if result := h.DB.First(&worker, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "worker not found")
			return
		}
		slog.Error("failed to find worker", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find worker")
		return
	}

	if result := h.DB.Delete(&worker); result.Error != nil {
		slog.Error("failed to delete worker", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to delete worker")
		return
	}

	slog.Info("worker deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}
```

---

## Step 26: Add nested routes to Farm handler

Add these methods to `handlers/farm.go`:

```go
// ListTrees handles GET /farms/{id}/trees
func (h *FarmHandler) ListTrees(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pagination := helpers.ParsePagination(r)

	// Verify farm exists
	var farm models.Farm
	if result := h.DB.First(&farm, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "farm not found")
			return
		}
		slog.Error("failed to find farm", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find farm")
		return
	}

	var trees []models.BananaTree
	var total int64

	query := h.DB.Model(&models.BananaTree{}).Where("farm_id = ?", id)

	// Support filtering on nested route too
	if status := r.URL.Query().Get("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	result := query.Offset(pagination.Offset()).Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&trees)

	if result.Error != nil {
		slog.Error("failed to list farm trees", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to list trees")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, helpers.NewPaginatedResponse(trees, total, pagination))
}

// ListWorkers handles GET /farms/{id}/workers
func (h *FarmHandler) ListWorkers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pagination := helpers.ParsePagination(r)

	var farm models.Farm
	if result := h.DB.First(&farm, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "farm not found")
			return
		}
		slog.Error("failed to find farm", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find farm")
		return
	}

	var workers []models.Worker
	var total int64

	query := h.DB.Model(&models.Worker{}).Where("farm_id = ?", id)

	if role := r.URL.Query().Get("role"); role != "" {
		query = query.Where("role = ?", role)
	}

	query.Count(&total)

	result := query.Offset(pagination.Offset()).Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&workers)

	if result.Error != nil {
		slog.Error("failed to list farm workers", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to list workers")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, helpers.NewPaginatedResponse(workers, total, pagination))
}

// ListTools handles GET /farms/{id}/tools
func (h *FarmHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pagination := helpers.ParsePagination(r)

	var farm models.Farm
	if result := h.DB.First(&farm, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "farm not found")
			return
		}
		slog.Error("failed to find farm", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find farm")
		return
	}

	var tools []models.Tool
	var total int64

	query := h.DB.Model(&models.Tool{}).Where("farm_id = ?", id)

	if toolType := r.URL.Query().Get("type"); toolType != "" {
		query = query.Where("type = ?", toolType)
	}

	query.Count(&total)

	result := query.Offset(pagination.Offset()).Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&tools)

	if result.Error != nil {
		slog.Error("failed to list farm tools", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to list tools")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, helpers.NewPaginatedResponse(tools, total, pagination))
}
```

---

## Step 27: Update `routes/routes.go` with all routes

Add all new handlers and routes. The full routes section becomes:

```go
	// --- Routes ---

	// Farm routes
	farmHandler := handlers.NewFarmHandler(db)
	r.Route("/farms", func(r chi.Router) {
		r.Get("/", farmHandler.List)
		r.Post("/", farmHandler.Create)
		r.Get("/{id}", farmHandler.Get)
		r.Put("/{id}", farmHandler.Update)
		r.Delete("/{id}", farmHandler.Delete)
		r.Get("/{id}/trees", farmHandler.ListTrees)
		r.Get("/{id}/workers", farmHandler.ListWorkers)
		r.Get("/{id}/tools", farmHandler.ListTools)
		// Stats endpoint added in Phase 4
		// r.Get("/{id}/stats", farmHandler.Stats)
	})

	// Banana Tree routes
	treeHandler := handlers.NewBananaTreeHandler(db)
	r.Route("/trees", func(r chi.Router) {
		r.Get("/", treeHandler.List)
		r.Post("/", treeHandler.Create)
		r.Get("/{id}", treeHandler.Get)
		r.Put("/{id}", treeHandler.Update)
		r.Delete("/{id}", treeHandler.Delete)
		r.Get("/{id}/bunches", treeHandler.ListBunches)
	})

	// Bunch routes
	bunchHandler := handlers.NewBunchHandler(db)
	r.Route("/bunches", func(r chi.Router) {
		r.Get("/", bunchHandler.List)
		r.Post("/", bunchHandler.Create)
		r.Get("/{id}", bunchHandler.Get)
		r.Put("/{id}", bunchHandler.Update)
		r.Delete("/{id}", bunchHandler.Delete)
		r.Get("/{id}/bananas", bunchHandler.ListBananas)
	})

	// Banana routes
	bananaHandler := handlers.NewBananaHandler(db)
	r.Route("/bananas", func(r chi.Router) {
		r.Get("/", bananaHandler.List)
		r.Post("/", bananaHandler.Create)
		r.Get("/{id}", bananaHandler.Get)
		r.Put("/{id}", bananaHandler.Update)
		r.Delete("/{id}", bananaHandler.Delete)
	})

	// Tool routes
	toolHandler := handlers.NewToolHandler(db)
	r.Route("/tools", func(r chi.Router) {
		r.Get("/", toolHandler.List)
		r.Post("/", toolHandler.Create)
		r.Get("/{id}", toolHandler.Get)
		r.Put("/{id}", toolHandler.Update)
		r.Delete("/{id}", toolHandler.Delete)
	})

	// Worker routes
	workerHandler := handlers.NewWorkerHandler(db)
	r.Route("/workers", func(r chi.Router) {
		r.Get("/", workerHandler.List)
		r.Post("/", workerHandler.Create)
		r.Get("/{id}", workerHandler.Get)
		r.Put("/{id}", workerHandler.Update)
		r.Delete("/{id}", workerHandler.Delete)
	})
```

---

## Step 28: Update `database/database.go` migrations

Make sure all models are in AutoMigrate:

```go
err := db.AutoMigrate(
    &models.Farm{},
    &models.BananaTree{},
    &models.Bunch{},
    &models.Banana{},
    &models.Tool{},
    &models.Worker{},
)
```

**Delete `banana_farm.db`** before running to get a fresh schema, then restart:

```bash
rm banana_farm.db
go run .
```

---

## Verify Phase 3 is complete

### Test the full relationship chain:

```bash
# 1. Create a farm
curl -X POST http://localhost:8080/farms \
  -H "Content-Type: application/json" \
  -d '{"name": "Cordova Plantation", "location": "Hawaii", "size_acres": 50, "established": "2020-01-01"}'

# 2. Create a tree on the farm
curl -X POST http://localhost:8080/trees \
  -H "Content-Type: application/json" \
  -d '{"farm_id": 1, "variety": "cavendish", "planted_at": "2024-03-01"}'

# 3. Update tree status through its lifecycle
curl -X PUT http://localhost:8080/trees/1 \
  -H "Content-Type: application/json" \
  -d '{"status": "flowering"}'

# 4. Create a bunch on the tree
curl -X POST http://localhost:8080/bunches \
  -H "Content-Type: application/json" \
  -d '{"banana_tree_id": 1, "weight_kg": 25.5}'

# 5. Create bananas on the bunch
curl -X POST http://localhost:8080/bananas \
  -H "Content-Type: application/json" \
  -d '{"bunch_id": 1, "hand_number": 1, "size": "large", "weight_grams": 120}'

# 6. Navigate relationships
curl http://localhost:8080/farms/1/trees
curl http://localhost:8080/trees/1/bunches
curl http://localhost:8080/bunches/1/bananas

# 7. Filter
curl "http://localhost:8080/trees?status=flowering"
curl "http://localhost:8080/bananas?ripeness=green"
curl "http://localhost:8080/tools?type=machete"

# 8. Create workers and tools
curl -X POST http://localhost:8080/workers \
  -H "Content-Type: application/json" \
  -d '{"farm_id": 1, "name": "Juan", "role": "harvester"}'

curl -X POST http://localhost:8080/tools \
  -H "Content-Type: application/json" \
  -d '{"farm_id": 1, "name": "Main Machete", "type": "machete"}'
```

---

## Phase 3 Checklist

- [ ] `models/stubs.go` deleted
- [ ] `models/banana_tree.go` with lifecycle status constants and validation
- [ ] `models/bunch.go` with nullable harvested_at
- [ ] `models/banana.go` with hand_number and ripeness
- [ ] `models/tool.go` with type and condition enums
- [ ] `models/worker.go` with role enum
- [ ] `handlers/banana_tree.go` with CRUD + ListBunches
- [ ] `handlers/bunch.go` with CRUD + ListBananas
- [ ] `handlers/banana.go` with full CRUD
- [ ] `handlers/tool.go` with full CRUD
- [ ] `handlers/worker.go` with full CRUD
- [ ] Farm handler has ListTrees, ListWorkers, ListTools methods
- [ ] `routes/routes.go` updated with all routes
- [ ] `database/database.go` has all models in AutoMigrate
- [ ] Full relationship chain works: Farm → Tree → Bunch → Banana
- [ ] Nested routes work: `/farms/1/trees`, `/trees/1/bunches`, `/bunches/1/bananas`
- [ ] Filtering works on all list endpoints
- [ ] Foreign key validation (can't create tree for nonexistent farm)
