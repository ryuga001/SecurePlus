package datadiscovery

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	dto "dpdp-backend/internal/admin/dto/datadiscovery"
	repo "dpdp-backend/internal/admin/repositories/datadiscovery"
	service "dpdp-backend/internal/admin/services/datadiscovery"
	"dpdp-backend/internal/admin/utils"
	"dpdp-backend/internal/db"
)

const (
	PrivilegeScanView   = "admin.discovery.scan.view"
	PrivilegeScanCreate = "admin.discovery.scan.create"
)

var scanStatuses = []string{
	db.ScanStatusPending,
	db.ScanStatusRunning,
	db.ScanStatusPartial,
	db.ScanStatusCompleted,
	db.ScanStatusFailed,
}

var fileStatuses = []string{db.FileStatusSucceeded, db.FileStatusFailed}

type ScanHandler struct {
	svc *service.ScanService
}

func NewScanHandler(svc *service.ScanService) *ScanHandler {
	return &ScanHandler{svc: svc}
}

func (h *ScanHandler) RegisterRoutes(
	protected *gin.RouterGroup,
	guard func(privilege string) gin.HandlerFunc,
) {
	group := protected.Group("/admin/data-discovery/scans")

	group.GET("", guard(PrivilegeScanView), h.list)
	group.GET("/:id", guard(PrivilegeScanView), h.get)
	group.GET("/:id/files", guard(PrivilegeScanView), h.files)
	group.POST("", guard(PrivilegeScanCreate), h.create)
}

func (h *ScanHandler) list(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var query dto.ScanListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c)
		return
	}

	listing, err := h.svc.List(c.Request.Context(), actor.CustomerID, repo.ScanListParams{
		PolicyIDs: utils.ParseIDList(query.PolicyIDs),
		Statuses:  utils.ParseEnumList(query.Statuses, scanStatuses...),
		Page:      query.Page,
		PageSize:  query.PageSize,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	items := make([]dto.ScanListItem, 0, len(listing.Items))
	for _, item := range listing.Items {
		items = append(items, toScanListItem(item.Scan, item.PolicyName))
	}

	c.JSON(http.StatusOK, utils.ListResponse[dto.ScanListItem]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *ScanHandler) get(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	id, ok := scanID(c)
	if !ok {
		return
	}

	detail, err := h.svc.Get(c.Request.Context(), actor.CustomerID, id)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toScanResponse(detail))
}

func (h *ScanHandler) files(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	id, ok := scanID(c)
	if !ok {
		return
	}

	var query dto.FileResultQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c)
		return
	}

	listing, err := h.svc.Files(c.Request.Context(), actor.CustomerID, id, repo.FileResultListParams{
		Statuses:     utils.ParseEnumList(query.Statuses, fileStatuses...),
		WithFindings: query.WithFindings,
		Page:         query.Page,
		PageSize:     query.PageSize,
	})
	if err != nil {
		utils.Respond(c, err)
		return
	}

	items := make([]dto.FileResultItem, 0, len(listing.Items))
	for _, row := range listing.Items {
		findings := []db.FileFinding(row.Findings)
		if findings == nil {
			findings = []db.FileFinding{}
		}

		items = append(items, dto.FileResultItem{
			ID:             row.ID,
			TargetPosition: row.TargetPosition,
			FileKey:        row.FileKey,
			FileName:       row.FileName,
			Extension:      row.Extension,
			MIMEType:       row.MIMEType,
			SizeBytes:      row.SizeBytes,
			ModifiedAt:     row.ModifiedAt,
			Status:         row.Status,
			ErrorCode:      text(row.ErrorCode),
			FindingsTotal:  row.FindingsTotal,
			Findings:       findings,
			BytesProcessed: row.BytesProcessed,
			DurationMS:     row.DurationMS,
			ProcessedAt:    row.ProcessedAt,
		})
	}

	c.JSON(http.StatusOK, utils.ListResponse[dto.FileResultItem]{
		Items:    items,
		Page:     listing.Page,
		PageSize: listing.PageSize,
		Total:    listing.Total,
	})
}

func (h *ScanHandler) create(c *gin.Context) {
	actor, ok := utils.Actor(c)
	if !ok {
		return
	}

	var req dto.ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c)
		return
	}

	detail, err := h.svc.Create(c.Request.Context(), actor.CustomerID, actor.UserID, req.PolicyID)
	if err != nil {
		utils.Respond(c, err)
		return
	}

	c.JSON(http.StatusAccepted, toScanResponse(detail))
}

func scanID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		utils.Fail(c, http.StatusBadRequest, utils.CodeValidation, utils.MsgInvalidIdentifier)
		return 0, false
	}

	return id, true
}

func toScanListItem(scan db.DataDiscoveryScan, policyName string) dto.ScanListItem {
	return dto.ScanListItem{
		ID:         scan.ID,
		PolicyID:   scan.PolicyID,
		PolicyName: policyName,
		Status:     scan.Status,
		ErrorCode:  text(scan.ErrorCode),
		Counters:   scan.ScanCounters,
		StartedAt:  scan.StartedAt,
		FinishedAt: scan.FinishedAt,
		CreatedAt:  scan.CreatedAt,
		UpdatedAt:  scan.UpdatedAt,
	}
}

func toScanResponse(detail service.ScanDetail) dto.ScanResponse {
	targets := make([]dto.ScanTargetItem, 0, len(detail.Targets))
	for _, target := range detail.Targets {
		targets = append(targets, dto.ScanTargetItem{
			Position:        target.Position,
			Target:          target.Target,
			Status:          target.Status,
			ErrorCode:       text(target.ErrorCode),
			FilesDiscovered: target.FilesDiscovered,
			FilesSkipped:    target.FilesSkipped,
			FilesSucceeded:  target.FilesSucceeded,
			FilesFailed:     target.FilesFailed,
			FindingsTotal:   target.FindingsTotal,
			StartedAt:       target.StartedAt,
			FinishedAt:      target.FinishedAt,
		})
	}

	return dto.ScanResponse{
		ScanListItem: toScanListItem(detail.Scan, detail.PolicyName),
		Targets:      targets,
	}
}
