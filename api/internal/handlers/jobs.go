package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/auth"
	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

// --- Job board: "Hire the Fallen" ---
// Companies pay to post here. IsSponsored pins to the top — that's the premium tier.
// In MVP, sponsorship is toggled manually after payment. Stripe webhook comes later.

type jobRequest struct {
	Company     string `json:"company" binding:"required,min=2,max=120"`
	Title       string `json:"title" binding:"required,min=2,max=200"`
	Body        string `json:"body" binding:"required,min=10,max=8000"`
	ApplyURL    string `json:"apply_url" binding:"required,url"`
	Location    string `json:"location" binding:"max=120"`
	Remote      bool   `json:"remote"`
	CountryCode string `json:"country_code" binding:"max=2"`
	SalaryMin   *int   `json:"salary_min"`
	SalaryMax   *int   `json:"salary_max"`
	Currency    string `json:"currency" binding:"max=3"`
	Seniority   string `json:"seniority"`
	Language    string `json:"language"`
}

// listJobs returns job postings. Sponsored ones float to the top, then newest.
// Public — you don't need an account to see who's hiring the survivors.
func (a *API) listJobs(c *gin.Context) {
	limit, offset := page(c)

	q := a.db.Model(&models.JobPosting{}).Where("is_filled = ?", false)

	if seniority := c.Query("seniority"); seniority != "" {
		q = q.Where("seniority = ?", seniority)
	}
	if remote := c.Query("remote"); remote == "true" {
		q = q.Where("remote = ?", true)
	}
	if cc := c.Query("country"); cc != "" {
		q = q.Where("country_code = ?", strings.ToUpper(cc))
	}
	if lang := c.Query("language"); lang != "" {
		q = q.Where("language = ?", normalizeLanguage(lang, "en"))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		serverError(c, err)
		return
	}

	var jobs []models.JobPosting
	if err := q.Preload("PostedBy").
		Order("is_sponsored DESC, created_at DESC").
		Limit(limit).Offset(offset).
		Find(&jobs).Error; err != nil {
		serverError(c, err)
		return
	}

	items := make([]jobPostingResponse, 0, len(jobs))
	for i := range jobs {
		items = append(items, *toJobPostingResponse(&jobs[i]))
	}
	c.JSON(http.StatusOK, paginated[jobPostingResponse]{
		Items: items, Total: total, Page: offset/limit + 1,
	})
}

// getJob returns one job posting and bumps the view counter.
func (a *API) getJob(c *gin.Context) {
	j, err := a.findJob(c.Param("id"))
	if err != nil {
		notFound(c, "That job was filled, deleted, or never existed. Like your last role.")
		return
	}
	a.db.Model(&models.JobPosting{}).Where("id = ?", j.ID).UpdateColumn("view_count", j.ViewCount+1)

	c.JSON(http.StatusOK, gin.H{"job": toJobPostingResponse(j)})
}

// createJob creates a job posting. Any logged-in user can post — the payment
// and sponsorship toggle happens out-of-band in MVP.
func (a *API) createJob(c *gin.Context) {
	u := auth.Current(c)
	var req jobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid job posting: "+err.Error())
		return
	}
	if !isHTTPURL(req.ApplyURL) {
		badRequest(c, "apply_url must be a valid http(s) link. No javascript: tricks.")
		return
	}
	j := models.JobPosting{
		PublicID:    store.NewPublicID("job"),
		PostedByID:  u.ID,
		Company:     strings.TrimSpace(req.Company),
		Title:       strings.TrimSpace(req.Title),
		Body:        strings.TrimSpace(req.Body),
		ApplyURL:    strings.TrimSpace(req.ApplyURL),
		Location:    strings.TrimSpace(req.Location),
		Remote:      req.Remote,
		CountryCode: strings.ToUpper(strings.TrimSpace(req.CountryCode)),
		SalaryMin:   req.SalaryMin,
		SalaryMax:   req.SalaryMax,
		Currency:    firstNonEmpty(strings.ToUpper(req.Currency), "USD"),
		Seniority:   models.Seniority(firstNonEmpty(req.Seniority, string(models.SeniorityMid))),
		Language:    normalizeLanguage(req.Language, "en"),
	}

	if err := a.db.Create(&j).Error; err != nil {
		serverError(c, err)
		return
	}

	j.PostedBy = u
	c.JSON(http.StatusCreated, gin.H{"job": toJobPostingResponse(&j)})
}

// toggleSponsored pins or unpins a job. Admin only in MVP — this is the money lever.
func (a *API) toggleSponsored(c *gin.Context) {
	u := auth.Current(c)
	if u.Role != "admin" {
		fail(c, http.StatusForbidden, "forbidden",
			"Only admins can sponsor jobs. Pay us and we'll flip the switch.")
		return
	}

	j, err := a.findJob(c.Param("id"))
	if err != nil {
		notFound(c, "Job not found. It may have been filled.")
		return
	}

	j.IsSponsored = !j.IsSponsored
	if err := a.db.Save(&j).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"job": toJobPostingResponse(j), "sponsored": j.IsSponsored})
}

// markJobFilled takes a job off the market.
func (a *API) markJobFilled(c *gin.Context) {
	u := auth.Current(c)
	j, err := a.findJob(c.Param("id"))
	if err != nil {
		notFound(c, "Job not found.")
		return
	}
	if j.PostedByID != u.ID && u.Role != "admin" {
		fail(c, http.StatusForbidden, "forbidden", "That's not your job to fill.")
		return
	}
	j.IsFilled = true
	if err := a.db.Save(&j).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"job": toJobPostingResponse(j)})
}

// deleteJob soft-deletes a job posting. Owner or admin.
func (a *API) deleteJob(c *gin.Context) {
	u := auth.Current(c)
	j, err := a.findJob(c.Param("id"))
	if err != nil {
		notFound(c, "Job not found.")
		return
	}
	if j.PostedByID != u.ID && u.Role != "admin" {
		fail(c, http.StatusForbidden, "forbidden", "You can't delete someone else's posting.")
		return
	}
	if err := a.db.Delete(&j).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Job removed. The position has been deprecated."})
}

func (a *API) findJob(publicID string) (*models.JobPosting, error) {
	var j models.JobPosting
	err := a.db.Preload("PostedBy").Where("public_id = ?", publicID).First(&j).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// jobStats powers the "Hire the Fallen" sidebar: how many open, how many sponsored.
func (a *API) jobStats(c *gin.Context) {
	var openCount, sponsoredCount int64
	a.db.Model(&models.JobPosting{}).Where("is_filled = ?", false).Count(&openCount)
	a.db.Model(&models.JobPosting{}).Where("is_filled = ? AND is_sponsored = ?", false, true).Count(&sponsoredCount)

	// Count by seniority for the filter sidebar.
	type seniorityCount struct {
		Seniority string `json:"seniority"`
		Count     int    `json:"count"`
	}
	var bySeniority []seniorityCount
	a.db.Model(&models.JobPosting{}).
		Select("seniority, count(*) as count").
		Where("is_filled = ?", false).
		Group("seniority").
		Order("count DESC").
		Scan(&bySeniority)

	c.JSON(http.StatusOK, gin.H{
		"open":         openCount,
		"sponsored":    sponsoredCount,
		"by_seniority": bySeniority,
		"posted_ago":   time.Now().Format(time.RFC3339),
	})
}
