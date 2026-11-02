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

// getUser returns a public profile by username.
func (a *API) getUser(c *gin.Context) {
	viewer := auth.Current(c)
	var u models.User
	if err := a.db.Preload("Repos").Preload("Badges").
		Where("username = ?", strings.ToLower(c.Param("username"))).First(&u).Error; err != nil {
		notFound(c, "No such user. They may have been optimised away.")
		return
	}

	out := toUserResponse(&u)

	var followers int64
	a.db.Model(&models.Follow{}).Where("following_id = ?", u.ID).Count(&followers)
	out.FollowerCount = int(followers)

	if viewer != nil && viewer.ID != u.ID {
		var n int64
		a.db.Model(&models.Follow{}).
			Where("follower_id = ? AND following_id = ?", viewer.ID, u.ID).Count(&n)
		following := n > 0
		out.IsFollowing = &following
	}

	var threads []models.Thread
	a.db.Preload("Author").Preload("Category").Preload("Tags").
		Where("author_id = ?", u.ID).Order("created_at DESC").Limit(20).Find(&threads)
	threadDTOs := make([]threadResponse, 0, len(threads))
	for i := range threads {
		threadDTOs = append(threadDTOs, *a.toThreadResponse(&threads[i], viewer))
	}

	c.JSON(http.StatusOK, gin.H{"user": out, "threads": threadDTOs})
}

// listUsers is the member directory, filterable by nationality and seniority.
func (a *API) listUsers(c *gin.Context) {
	limit, offset := page(c)

	q := a.db.Model(&models.User{})
	if v := c.Query("country"); v != "" {
		q = q.Where("country_code = ?", strings.ToUpper(v))
	}
	if v := c.Query("seniority"); v != "" {
		q = q.Where("seniority = ?", v)
	}
	if v := c.Query("cause"); v != "" {
		q = q.Where("cause_of_death = ?", v)
	}
	if v := c.Query("employed"); v == "false" {
		q = q.Where("is_employed = ?", false)
	} else if v == "true" {
		q = q.Where("is_employed = ?", true)
	}
	if v := c.Query("language"); v != "" {
		q = q.Where("spoken_languages LIKE ?", "%"+strings.ToLower(v)+"%")
	}
	if v := strings.TrimSpace(c.Query("q")); v != "" {
		like := "%" + strings.ToLower(v) + "%"
		q = q.Where("LOWER(username) LIKE ? OR LOWER(display_name) LIKE ?", like, like)
	}

	var total int64
	q.Count(&total)

	order := "reputation DESC, created_at DESC"
	switch c.Query("sort") {
	case "new":
		order = "created_at DESC"
	case "fired":
		order = "fired_at DESC"
	case "loud":
		order = "post_count DESC"
	}

	var users []models.User
	if err := q.Preload("Badges").Order(order).Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		serverError(c, err)
		return
	}
	items := make([]userResponse, 0, len(users))
	for i := range users {
		items = append(items, *toUserResponse(&users[i]))
	}
	c.JSON(http.StatusOK, paginated[userResponse]{
		Items: items, Page: offset/limit + 1, Limit: limit, Total: total,
	})
}

type updateProfileRequest struct {
	DisplayName     *string  `json:"display_name"`
	Bio             *string  `json:"bio"`
	CountryCode     *string  `json:"country_code"`
	Timezone        *string  `json:"timezone"`
	Seniority       *string  `json:"seniority"`
	YearsExp        *int     `json:"years_exp"`
	CauseOfDeath    *string  `json:"cause_of_death"`
	Epitaph         *string  `json:"epitaph"`
	CurrentRole     *string  `json:"current_role"`
	IsEmployed      *bool    `json:"is_employed"`
	FiredAt         *string  `json:"fired_at"`
	WebsiteURL      *string  `json:"website_url"`
	LinkedInURL     *string  `json:"linkedin_url"`
	UILanguage      *string  `json:"ui_language"`
	SpokenLanguages []string `json:"spoken_languages"`
	CopiumMode      *bool    `json:"copium_mode"`
}

// updateProfile is a partial update: only the fields present in the body change.
func (a *API) updateProfile(c *gin.Context) {
	u := auth.Current(c)
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "Invalid payload.")
		return
	}

	if req.DisplayName != nil {
		u.DisplayName = truncate(strings.TrimSpace(*req.DisplayName), 80)
	}
	if req.Bio != nil {
		u.Bio = truncate(*req.Bio, 500)
	}
	if req.CountryCode != nil {
		code := strings.ToUpper(strings.TrimSpace(*req.CountryCode))
		if code != "" && !isValidOption(countryOptions, code) {
			badRequest(c, "Unknown country code: "+code)
			return
		}
		u.CountryCode = code
	}
	if req.Timezone != nil {
		u.Timezone = truncate(*req.Timezone, 64)
	}
	if req.Seniority != nil {
		if !isValidOption(seniorityOptions, *req.Seniority) {
			badRequest(c, "Unknown seniority: "+*req.Seniority)
			return
		}
		u.Seniority = models.Seniority(*req.Seniority)
	}
	if req.YearsExp != nil {
		u.YearsExp = clamp(*req.YearsExp, 0, 60)
	}
	if req.CauseOfDeath != nil {
		if !isValidOption(causeOptions, *req.CauseOfDeath) {
			badRequest(c, "Unknown cause of death: "+*req.CauseOfDeath)
			return
		}
		u.CauseOfDeath = models.CauseOfDeath(*req.CauseOfDeath)
	}
	if req.Epitaph != nil {
		u.Epitaph = truncate(*req.Epitaph, 180)
	}
	if req.CurrentRole != nil {
		u.CurrentRole = truncate(*req.CurrentRole, 120)
	}
	if req.IsEmployed != nil {
		u.IsEmployed = *req.IsEmployed
	}
	if req.FiredAt != nil {
		if *req.FiredAt == "" {
			u.FiredAt = nil
		} else {
			t, err := time.Parse("2006-01-02", *req.FiredAt)
			if err != nil {
				badRequest(c, "fired_at must be YYYY-MM-DD.")
				return
			}
			u.FiredAt = &t
		}
	}
	if req.WebsiteURL != nil {
		if *req.WebsiteURL != "" && !isHTTPURL(*req.WebsiteURL) {
			badRequest(c, "website_url must be a valid http(s) link.")
			return
		}
		u.WebsiteURL = truncate(*req.WebsiteURL, 255)
	}
	if req.LinkedInURL != nil {
		if *req.LinkedInURL != "" && !isHTTPURL(*req.LinkedInURL) {
			badRequest(c, "linkedin_url must be a valid http(s) link.")
			return
		}
		u.LinkedInURL = truncate(*req.LinkedInURL, 255)
	}
	if req.UILanguage != nil {
		u.UILanguage = normalizeLanguage(*req.UILanguage, u.UILanguage)
	}
	if req.SpokenLanguages != nil {
		valid := make([]string, 0, len(req.SpokenLanguages))
		for _, l := range req.SpokenLanguages {
			l = strings.ToLower(strings.TrimSpace(l))
			if isValidOption(languageOptions, l) {
				valid = append(valid, l)
			}
		}
		u.SpokenLanguages = strings.Join(valid, ",")
	}
	if req.CopiumMode != nil {
		u.CopiumMode = *req.CopiumMode
	}

	if err := a.db.Save(u).Error; err != nil {
		serverError(c, err)
		return
	}
	a.awardBadges(u)
	c.JSON(http.StatusOK, gin.H{"user": toUserResponse(u)})
}

type repoRequest struct {
	Name        string `json:"name" binding:"required,max=140"`
	URL         string `json:"url" binding:"required,url,max=512"`
	Description string `json:"description" binding:"max=300"`
	Language    string `json:"language" binding:"max=40"`
	WrittenByAI bool   `json:"written_by_ai"`
}

func (a *API) addRepo(c *gin.Context) {
	u := auth.Current(c)
	var req repoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "A repo needs a name and a valid URL. That's the whole form.")
		return
	}
	if !isHTTPURL(req.URL) {
		badRequest(c, "A repo URL needs to be http(s). No javascript: tricks.")
		return
	}
	r := models.Repo{
		PublicID: store.NewPublicID("rep"), UserID: u.ID,
		Name: req.Name, URL: req.URL, Description: req.Description,
		Language: req.Language, WrittenByAI: req.WrittenByAI,
	}
	if err := a.db.Create(&r).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"repo": repoDTO{
		ID: r.PublicID, Name: r.Name, URL: r.URL, Description: r.Description,
		Language: r.Language, Stars: r.Stars, WrittenByAI: r.WrittenByAI,
	}})
}

func (a *API) deleteRepo(c *gin.Context) {
	u := auth.Current(c)
	res := a.db.Where("public_id = ? AND user_id = ?", c.Param("id"), u.ID).Delete(&models.Repo{})
	if res.Error != nil {
		serverError(c, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		notFound(c, "No such repo on your profile.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Repo unlinked."})
}

// toggleFollow follows or unfollows a user.
func (a *API) toggleFollow(c *gin.Context) {
	me := auth.Current(c)
	var target models.User
	if err := a.db.Select("id", "username").
		Where("username = ?", strings.ToLower(c.Param("username"))).First(&target).Error; err != nil {
		notFound(c, "No such user.")
		return
	}
	if target.ID == me.ID {
		badRequest(c, "You cannot follow yourself. Emotionally, you already do.")
		return
	}

	var f models.Follow
	err := a.db.Where("follower_id = ? AND following_id = ?", me.ID, target.ID).First(&f).Error
	switch {
	case err == nil:
		if err := a.db.Delete(&f).Error; err != nil {
			serverError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"following": false})
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := a.db.Create(&models.Follow{FollowerID: me.ID, FollowingID: target.ID}).Error; err != nil {
			serverError(c, err)
			return
		}
		a.notify(target.ID, "follow", "@"+me.Username+" is watching your decline", "", "/u/"+me.Username)
		c.JSON(http.StatusOK, gin.H{"following": true})
	default:
		serverError(c, err)
	}
}

// stats powers the "Wall of the Fallen" dashboard.
func (a *API) stats(c *gin.Context) {
	type countRow struct {
		Key   string `json:"key"`
		Count int    `json:"count"`
	}

	var totalUsers, totalThreads, totalPosts, totalFired, totalReactions int64
	a.db.Model(&models.User{}).Count(&totalUsers)
	a.db.Model(&models.Thread{}).Count(&totalThreads)
	a.db.Model(&models.Post{}).Count(&totalPosts)
	a.db.Model(&models.User{}).Where("is_employed = ?", false).Count(&totalFired)
	a.db.Model(&models.Reaction{}).Count(&totalReactions)

	var byCause, byCountry, bySeniority, byLanguage []countRow
	a.db.Model(&models.User{}).Select("cause_of_death as key, COUNT(*) as count").
		Group("cause_of_death").Order("count DESC").Scan(&byCause)
	a.db.Model(&models.User{}).Select("country_code as key, COUNT(*) as count").
		Where("country_code != ''").Group("country_code").Order("count DESC").Limit(20).Scan(&byCountry)
	a.db.Model(&models.User{}).Select("seniority as key, COUNT(*) as count").
		Group("seniority").Order("count DESC").Scan(&bySeniority)
	a.db.Model(&models.Thread{}).Select("language as key, COUNT(*) as count").
		Group("language").Order("count DESC").Scan(&byLanguage)

	// The headline number: how many distinct ways we have collectively been replaced.
	var distinctCauses int64
	a.db.Model(&models.User{}).Distinct("cause_of_death").Count(&distinctCauses)

	c.JSON(http.StatusOK, gin.H{
		"total_users":     totalUsers,
		"total_threads":   totalThreads,
		"total_posts":     totalPosts,
		"total_fired":     totalFired,
		"total_reactions": totalReactions,
		"ways_documented": totalThreads, // one thread, one way the machine won
		"distinct_causes": distinctCauses,
		"by_cause":        byCause,
		"by_country":      byCountry,
		"by_seniority":    bySeniority,
		"by_language":     byLanguage,
	})
}

// awardBadges grants badges based on the profile's current state. Idempotent.
func (a *API) awardBadges(u *models.User) {
	earned := []string{}
	if u.CauseOfDeath == models.CauseShadowExposed {
		earned = append(earned, "shadow-exposed")
	}
	if !u.IsEmployed {
		earned = append(earned, "certified-obsolete")
	}
	if u.YearsExp >= 10 {
		earned = append(earned, "legacy-system")
	}
	if u.CopiumMode {
		earned = append(earned, "copium-enjoyer")
	}
	if len(splitCSV(u.SpokenLanguages)) >= 3 {
		earned = append(earned, "polyglot-unemployed")
	}
	if u.Seniority == models.SeniorityCTO {
		earned = append(earned, "saw-the-demo")
	}
	if len(earned) == 0 {
		return
	}
	var badges []models.Badge
	a.db.Where("slug IN ?", earned).Find(&badges)
	for i := range badges {
		// Append is a no-op when the association already exists.
		_ = a.db.Model(u).Association("Badges").Append(&badges[i])
	}
}
