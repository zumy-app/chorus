package handlers

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/chorus/messenger/internal/middleware"
	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

type TeacherHandler struct {
	svc    *services.TeacherService
	srsSvc *services.TeacherSrsService
}

func NewTeacherHandler(svc *services.TeacherService) *TeacherHandler {
	return &TeacherHandler{svc: svc}
}

func NewTeacherHandlerWithSRS(svc *services.TeacherService, srs *services.TeacherSrsService) *TeacherHandler {
	return &TeacherHandler{svc: svc, srsSvc: srs}
}

func (h *TeacherHandler) Apply(c *gin.Context) {
	userID := c.GetString("userID")
	var req models.TeacherApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid application. Check bio, languages, rate and video URL."))
		return
	}
	app, err := h.svc.Apply(userID, req)
	if err != nil {
		WriteError(c, middleware.ErrValidation(err.Error()))
		return
	}
	c.JSON(200, gin.H{"application": app})
}

func (h *TeacherHandler) GetMe(c *gin.Context) {
	userID := c.GetString("userID")
	app, err := h.svc.GetByUserID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(200, gin.H{"application": nil})
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to fetch application"))
		return
	}
	c.JSON(200, gin.H{"application": app})
}

func (h *TeacherHandler) Browse(c *gin.Context) {
	language := strings.ToLower(strings.TrimSpace(c.Query("language")))
	search := strings.TrimSpace(c.Query("search"))
	verifiedOnly := c.Query("verified") == "true" || c.Query("verified") == "1"
	sort := strings.TrimSpace(c.Query("sort"))
	minRating, _ := strconv.ParseFloat(c.Query("minRating"), 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	var maxRate *int
	if v := c.Query("maxRate"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxRate = &n
		}
	}
	var minRate *int
	if v := c.Query("minRate"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			minRate = &n
		}
	}
	tutors, total, err := h.svc.BrowseTutors(models.TutorBrowseFilter{
		Language:     language,
		Search:       search,
		VerifiedOnly: verifiedOnly,
		MinRating:    minRating,
		MaxRateCents: maxRate,
		MinRateCents: minRate,
		Sort:         sort,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to browse tutors"))
		return
	}
	hasMore := offset+len(tutors) < total
	c.JSON(200, gin.H{"tutors": tutors, "total": total, "hasMore": hasMore})
}

func (h *TeacherHandler) GetProfile(c *gin.Context) {
	id := c.Param("id")
	p, err := h.svc.GetTutorProfile(id)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Tutor not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to fetch tutor"))
		return
	}
	if p.Status != "approved" {
		WriteError(c, middleware.ErrNotFound("Tutor not found"))
		return
	}
	c.JSON(200, gin.H{"tutor": p})
}

func (h *TeacherHandler) GetTrialCredits(c *gin.Context) {
	userID := c.GetString("userID")
	tc, err := h.svc.GetTrialCredit(userID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch trial credits"))
		return
	}
	c.JSON(200, gin.H{"trialCredits": tc})
}

func (h *TeacherHandler) GetTrialCreditDashboard(c *gin.Context) {
	userID := c.GetString("userID")
	dash, err := h.svc.GetTrialCreditDashboard(userID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch trial dashboard"))
		return
	}
	c.JSON(200, gin.H{"dashboard": dash})
}

func (h *TeacherHandler) GetBooking(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")
	b, err := h.svc.GetBooking(userID, id)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Booking not found"))
			return
		}
		if strings.Contains(err.Error(), "not authorized") {
			WriteError(c, middleware.ErrForbidden("Not authorized"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to fetch booking"))
		return
	}
	c.JSON(200, gin.H{"booking": b})
}

func (h *TeacherHandler) CompleteBooking(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")
	b, err := h.svc.CompleteBooking(userID, id)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Booking not found"))
			return
		}
		if strings.Contains(err.Error(), "not authorized") {
			WriteError(c, middleware.ErrForbidden("Not authorized"))
			return
		}
		WriteError(c, middleware.ErrValidation(err.Error()))
		return
	}
	c.JSON(200, gin.H{"booking": b})
}

func (h *TeacherHandler) UpdateReviewNotes(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")
	var req models.UpdateReviewNotesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid notes"))
		return
	}
	b, err := h.svc.UpdateReviewNotes(userID, id, req.Notes)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Booking not found"))
			return
		}
		if strings.Contains(err.Error(), "only teacher") || strings.Contains(err.Error(), "not authorized") {
			WriteError(c, middleware.ErrForbidden(err.Error()))
			return
		}
		WriteError(c, middleware.ErrValidation(err.Error()))
		return
	}
	c.JSON(200, gin.H{"booking": b})
}

func (h *TeacherHandler) GetReviews(c *gin.Context) {
	id := c.Param("id")
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	reviews, total, err := h.svc.ListReviews(id, limit, offset)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch reviews"))
		return
	}
	hasMore := offset+len(reviews) < total
	c.JSON(200, gin.H{"reviews": reviews, "total": total, "hasMore": hasMore})
}

func (h *TeacherHandler) AddReview(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")
	var req models.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid review"))
		return
	}
	rev, err := h.svc.AddReview(userID, id, req.Rating, req.Comment)
	if err != nil {
		if strings.Contains(err.Error(), "tutor not") || strings.Contains(err.Error(), "not approved") {
			WriteError(c, middleware.ErrNotFound(err.Error()))
			return
		}
		if strings.Contains(err.Error(), "yourself") || strings.Contains(err.Error(), "rating") || strings.Contains(err.Error(), "comment") {
			WriteError(c, middleware.ErrValidation(err.Error()))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to add review"))
		return
	}
	c.JSON(201, gin.H{"review": rev})
}

func (h *TeacherHandler) GetDashboard(c *gin.Context) {
	userID := c.GetString("userID")
	dash, err := h.svc.GetDashboard(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Teacher application not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to fetch dashboard"))
		return
	}
	c.JSON(200, gin.H{"dashboard": dash})
}

func (h *TeacherHandler) GetAvailability(c *gin.Context) {
	id := c.Param("id")
	slots, err := h.svc.GetAvailability(id)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch availability"))
		return
	}
	c.JSON(200, gin.H{"availability": slots})
}

func (h *TeacherHandler) AddAvailability(c *gin.Context) {
	userID := c.GetString("userID")
	var req models.CreateAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid availability"))
		return
	}
	slot, err := h.svc.AddAvailability(userID, req.StartTime, req.EndTime)
	if err != nil {
		WriteError(c, middleware.ErrValidation(err.Error()))
		return
	}
	c.JSON(201, gin.H{"availability": slot})
}

func (h *TeacherHandler) RemoveAvailability(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")
	if err := h.svc.RemoveAvailability(userID, id); err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Availability not found"))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to remove availability"))
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

func (h *TeacherHandler) CreateBooking(c *gin.Context) {
	userID := c.GetString("userID")
	teacherID := c.Param("id")
	var req models.CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid booking"))
		return
	}
	b, err := h.svc.CreateBooking(userID, teacherID, req)
	if err != nil {
		if strings.Contains(err.Error(), "tutor not") || strings.Contains(err.Error(), "not approved") {
			WriteError(c, middleware.ErrNotFound(err.Error()))
			return
		}
		if strings.Contains(err.Error(), "no trial") || strings.Contains(err.Error(), "already booked") || strings.Contains(err.Error(), "yourself") || strings.Contains(err.Error(), "duration") || strings.Contains(err.Error(), "future") || strings.Contains(err.Error(), "end must") {
			WriteError(c, middleware.ErrValidation(err.Error()))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to create booking"))
		return
	}
	c.JSON(201, gin.H{"booking": b})
}

func (h *TeacherHandler) ListBookings(c *gin.Context) {
	userID := c.GetString("userID")
	role := strings.TrimSpace(c.Query("role"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	bookings, total, err := h.svc.ListBookings(userID, role, limit, offset)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch bookings"))
		return
	}
	hasMore := offset+len(bookings) < total
	c.JSON(200, gin.H{"bookings": bookings, "total": total, "hasMore": hasMore})
}

func (h *TeacherHandler) CancelBooking(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")
	b, err := h.svc.UpdateBookingStatus(userID, id, "cancelled")
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Booking not found"))
			return
		}
		if strings.Contains(err.Error(), "not authorized") {
			WriteError(c, middleware.ErrForbidden("Not authorized"))
			return
		}
		WriteError(c, middleware.ErrValidation(err.Error()))
		return
	}
	c.JSON(200, gin.H{"booking": b})
}

func (h *TeacherHandler) PushSRS(c *gin.Context) {
	if h.srsSvc == nil {
		WriteError(c, middleware.ErrInternal("SRS push not configured"))
		return
	}
	userID := c.GetString("userID")
	var req models.TeacherSrsPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteError(c, middleware.ErrValidation("Invalid push request: "+err.Error()))
		return
	}
	push, err := h.srsSvc.PushCards(userID, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "only approved") || strings.Contains(err.Error(), "no lesson") || strings.Contains(err.Error(), "booking") || strings.Contains(err.Error(), "cannot push") || strings.Contains(err.Error(), "cards") || strings.Contains(err.Error(), "language") || strings.Contains(err.Error(), "student") {
			WriteError(c, middleware.ErrValidation(err.Error()))
			return
		}
		if strings.Contains(err.Error(), "not authorized") {
			WriteError(c, middleware.ErrForbidden(err.Error()))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to push SRS cards"))
		return
	}
	c.JSON(201, gin.H{"push": push})
}

func (h *TeacherHandler) ListSrsPushes(c *gin.Context) {
	if h.srsSvc == nil {
		WriteError(c, middleware.ErrInternal("SRS push not configured"))
		return
	}
	userID := c.GetString("userID")
	role := strings.TrimSpace(c.Query("role"))
	peerID := strings.TrimSpace(c.Query("peerId"))
	if peerID == "" {
		peerID = strings.TrimSpace(c.Query("studentId"))
		if peerID == "" {
			peerID = strings.TrimSpace(c.Query("teacherId"))
		}
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	pushes, total, err := h.srsSvc.ListPushes(userID, role, peerID, limit, offset)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to list pushes"))
		return
	}
	hasMore := offset+len(pushes) < total
	c.JSON(200, gin.H{"pushes": pushes, "total": total, "hasMore": hasMore})
}

func (h *TeacherHandler) GetSrsPush(c *gin.Context) {
	if h.srsSvc == nil {
		WriteError(c, middleware.ErrInternal("SRS push not configured"))
		return
	}
	userID := c.GetString("userID")
	id := c.Param("id")
	p, err := h.srsSvc.GetPush(userID, id)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Push not found"))
			return
		}
		if strings.Contains(err.Error(), "not authorized") {
			WriteError(c, middleware.ErrForbidden(err.Error()))
			return
		}
		WriteError(c, middleware.ErrInternal("Failed to fetch push"))
		return
	}
	c.JSON(200, gin.H{"push": p})
}

func (h *TeacherHandler) GetSrsSandbox(c *gin.Context) {
	if h.srsSvc == nil {
		WriteError(c, middleware.ErrInternal("SRS push not configured"))
		return
	}
	userID := c.GetString("userID")
	peerID := c.Param("studentId")
	if peerID == "" {
		peerID = c.Param("id")
	}
	var teacherID, studentID string
	var status string
	if err := h.svc.DB().QueryRow(`SELECT status FROM teacher_applications WHERE user_id=$1`, userID).Scan(&status); err == nil && status == "approved" {
		teacherID = userID
		studentID = peerID
	} else {
		teacherID = peerID
		studentID = userID
	}
	pushes, err := h.srsSvc.ListPushesForSandbox(teacherID, studentID)
	if err != nil {
		WriteError(c, middleware.ErrInternal("Failed to fetch sandbox"))
		return
	}
	c.JSON(200, gin.H{"pushes": pushes, "teacherId": teacherID, "studentId": studentID})
}

func (h *TeacherHandler) ConfirmBooking(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")
	b, err := h.svc.UpdateBookingStatus(userID, id, "confirmed")
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(c, middleware.ErrNotFound("Booking not found"))
			return
		}
		if strings.Contains(err.Error(), "not authorized") || strings.Contains(err.Error(), "only teacher") {
			WriteError(c, middleware.ErrForbidden(err.Error()))
			return
		}
		WriteError(c, middleware.ErrValidation(err.Error()))
		return
	}
	c.JSON(200, gin.H{"booking": b})
}
