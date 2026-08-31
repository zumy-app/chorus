package handlers

import (
	"net/http"

	"github.com/chorus/messenger/internal/models"
	"github.com/chorus/messenger/internal/services"
	"github.com/gin-gonic/gin"
)

// LearningHandler serves the pair-aware learning endpoints across the full
// engine: capabilities, profile, dashboard, roadmap path, placement, lessons,
// practice sessions, word-mining candidates, scenarios/roleplay, real-talk
// prompts, and streak recovery. All routes require auth.
type LearningHandler struct {
	capabilities *services.LearningCapabilityService
	profiles     *services.LearningProfileService
	dashboard    *services.LearningDashboardService
	curriculum   *services.CurriculumService
	placement    *services.PlacementService
	lessons      *services.LessonService
	sessions     *services.SessionComposerService
	mining       *services.WordMiningService
	scenario     *services.ScenarioService
	fluency      *services.FluencyScoreService
	vocab        *services.VocabularyService
}

func NewLearningHandler(
	capabilities *services.LearningCapabilityService,
	profiles *services.LearningProfileService,
	dashboard *services.LearningDashboardService,
	curriculum *services.CurriculumService,
	placement *services.PlacementService,
	lessons *services.LessonService,
	sessions *services.SessionComposerService,
	mining *services.WordMiningService,
	scenario *services.ScenarioService,
	fluency *services.FluencyScoreService,
	vocab *services.VocabularyService,
) *LearningHandler {
	return &LearningHandler{
		capabilities: capabilities,
		profiles:     profiles,
		dashboard:    dashboard,
		curriculum:   curriculum,
		placement:    placement,
		lessons:      lessons,
		sessions:     sessions,
		mining:       mining,
		scenario:     scenario,
		fluency:      fluency,
		vocab:        vocab,
	}
}

func (h *LearningHandler) GetCapabilities(c *gin.Context) {
	capability, err := h.capabilities.GetCapability(c.Request.Context(), c.Query("nativeLanguage"), c.Query("targetLanguage"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve learning capability"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": capability})
}

func (h *LearningHandler) GetProfile(c *gin.Context) {
	profile, err := h.profiles.GetProfile(c.Request.Context(), c.GetString("userID"), c.Query("targetLanguage"), c.Query("nativeLanguage"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load learning profile"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": profile})
}

func (h *LearningHandler) UpdateProfile(c *gin.Context) {
	var req models.LearningProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	profile, err := h.profiles.UpdateProfile(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update learning profile"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": profile})
}

func (h *LearningHandler) GetDashboard(c *gin.Context) {
	dash, err := h.dashboard.GetDashboard(c.Request.Context(), c.GetString("userID"), c.Query("targetLanguage"), c.Query("nativeLanguage"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load dashboard"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": dash})
}

func (h *LearningHandler) GetPath(c *gin.Context) {
	userID := c.GetString("userID")
	target := c.Query("targetLanguage")
	native := c.Query("nativeLanguage")
	capability, err := h.capabilities.GetCapability(c.Request.Context(), native, target)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve learning capability"})
		return
	}
	profile, err := h.profiles.GetProfile(c.Request.Context(), userID, target, native)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load learning profile"})
		return
	}
	path, err := h.curriculum.GetLearningPath(c.Request.Context(), userID, profile, capability)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load learning path"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": path})
}

// ---------------------------------------------------------------------------
// Placement
// ---------------------------------------------------------------------------

func (h *LearningHandler) StartPlacement(c *gin.Context) {
	userID := c.GetString("userID")
	resp, err := h.placement.StartPlacement(c.Request.Context(), userID, c.Query("targetLanguage"), c.Query("nativeLanguage"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start placement"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *LearningHandler) AnswerPlacement(c *gin.Context) {
	attemptID := c.Param("attemptId")
	var req models.PlacementAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	result, err := h.placement.AnswerPlacement(c.Request.Context(), c.GetString("userID"), attemptID, req.Answer)
	if err != nil {
		if err.Error() == "not complete" {
			// Return progress (next question) without error.
			resp, _ := h.placement.GetPlacement(c.Request.Context(), c.GetString("userID"), attemptID)
			c.JSON(http.StatusOK, gin.H{"data": resp})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to answer placement"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *LearningHandler) SkipPlacement(c *gin.Context) {
	result, err := h.placement.SkipPlacement(c.Request.Context(), c.GetString("userID"), c.Query("targetLanguage"), c.Query("nativeLanguage"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to skip placement"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *LearningHandler) GetPlacement(c *gin.Context) {
	resp, err := h.placement.GetPlacement(c.Request.Context(), c.GetString("userID"), c.Param("attemptId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load placement"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// ---------------------------------------------------------------------------
// Lessons
// ---------------------------------------------------------------------------

func (h *LearningHandler) GetUnit(c *gin.Context) {
	unitID := c.Param("unitId")
	userID := c.GetString("userID")
	unit, err := h.curriculum.GetUnitDetail(c.Request.Context(), userID, unitID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load unit"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": unit})
}

func (h *LearningHandler) StartLesson(c *gin.Context) {
	resp, err := h.lessons.StartLesson(c.Request.Context(), c.GetString("userID"), c.Param("lessonId"), c.Query("targetLanguage"), c.Query("nativeLanguage"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start lesson"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *LearningHandler) AnswerLessonStep(c *gin.Context) {
	var req models.AnswerLessonStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	result, err := h.lessons.AnswerStep(c.Request.Context(), c.GetString("userID"), c.Param("attemptId"), c.Param("stepId"), req.UserAnswer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to answer step"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *LearningHandler) CompleteLesson(c *gin.Context) {
	result, err := h.lessons.CompleteLesson(c.Request.Context(), c.GetString("userID"), c.Param("attemptId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete lesson"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *LearningHandler) GetLessonAttempt(c *gin.Context) {
	attempt, steps, err := h.lessons.GetAttempt(c.Request.Context(), c.GetString("userID"), c.Param("attemptId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load lesson attempt"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"attempt": attempt, "steps": steps}})
}

// ---------------------------------------------------------------------------
// Practice sessions
// ---------------------------------------------------------------------------

func (h *LearningHandler) StartSession(c *gin.Context) {
	var req models.StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	resp, err := h.sessions.StartSession(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start session: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *LearningHandler) GetSession(c *gin.Context) {
	sess, err := h.sessions.GetSession(c.Request.Context(), c.GetString("userID"), c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sess})
}

func (h *LearningHandler) AnswerSessionItem(c *gin.Context) {
	var req models.AnswerSessionItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	resp, err := h.sessions.AnswerItem(c.Request.Context(), c.GetString("userID"), c.Param("sessionId"), c.Param("itemId"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to answer item"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *LearningHandler) CompleteSession(c *gin.Context) {
	sess, err := h.sessions.CompleteSession(c.Request.Context(), c.GetString("userID"), c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sess})
}

// ---------------------------------------------------------------------------
// Mined vocabulary
// ---------------------------------------------------------------------------

func (h *LearningHandler) GetMinedItems(c *gin.Context) {
	items, err := h.mining.GetCandidateItems(c.Request.Context(), c.GetString("userID"), c.Query("targetLanguage"), c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load mined items"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *LearningHandler) AcceptMinedItem(c *gin.Context) {
	card, err := h.mining.AcceptMinedItem(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept mined item"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": card})
}

func (h *LearningHandler) IgnoreMinedItem(c *gin.Context) {
	if err := h.mining.IgnoreMinedItem(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ignore mined item"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

// ---------------------------------------------------------------------------
// Scenarios
// ---------------------------------------------------------------------------

func (h *LearningHandler) ListScenarios(c *gin.Context) {
	scenarios, err := h.scenario.ListScenarios(c.Request.Context(), c.GetString("userID"), c.Query("targetLanguage"), c.Query("nativeLanguage"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load scenarios"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": scenarios})
}

func (h *LearningHandler) GetScenario(c *gin.Context) {
	scenario, err := h.scenario.GetScenario(c.Request.Context(), c.GetString("userID"), c.Param("scenarioId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load scenario"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": scenario})
}

func (h *LearningHandler) StartScenario(c *gin.Context) {
	resp, err := h.scenario.StartScenario(c.Request.Context(), c.GetString("userID"), c.Param("scenarioId"), c.Query("targetLanguage"), c.Query("nativeLanguage"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start scenario"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *LearningHandler) GetScenarioRun(c *gin.Context) {
	run, err := h.scenario.GetRun(c.Request.Context(), c.GetString("userID"), c.Param("runId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load scenario run"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": run})
}

func (h *LearningHandler) SendScenarioMessage(c *gin.Context) {
	var req models.SendScenarioMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	reply, err := h.scenario.SendMessage(c.Request.Context(), c.GetString("userID"), c.Param("runId"), req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process scenario message"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reply})
}

func (h *LearningHandler) ScenarioHint(c *gin.Context) {
	hints, err := h.scenario.RequestHint(c.Request.Context(), c.GetString("userID"), c.Param("runId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load hint"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": hints})
}

func (h *LearningHandler) CompleteScenario(c *gin.Context) {
	reply, err := h.scenario.CompleteScenario(c.Request.Context(), c.GetString("userID"), c.Param("runId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete scenario"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reply})
}

// ---------------------------------------------------------------------------
// Real talk / streak
// ---------------------------------------------------------------------------

func (h *LearningHandler) RealTalkPrompts(c *gin.Context) {
	userID := c.GetString("userID")
	target := c.Query("targetLanguage")
	native := c.Query("nativeLanguage")
	profile, err := h.profiles.GetProfile(c.Request.Context(), userID, target, native)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load profile"})
		return
	}
	unitTitle := ""
	if profile.ActiveUnitID != "" {
		unitTitle = h.curriculum.GetUnitTitle(c.Request.Context(), profile.ActiveUnitID)
	}
	prompts := []models.RealTalkPrompt{
		{ID: "rt-1", Category: "Icebreakers", Text: "What is the first thing you usually do when you wake up?"},
		{ID: "rt-2", Category: "Deep Dives", Text: "Describe your favorite weekend activity and why it helps you relax."},
		{ID: "rt-3", Category: "Task-Based", Text: "Order a coffee and ask for a receipt, phrased naturally."},
	}
	if unitTitle != "" {
		prompts = append(prompts, models.RealTalkPrompt{ID: "rt-0", Category: "Unit goal", Text: "Practice something from your current unit: " + unitTitle, SourcePrompt: true})
	}
	c.JSON(http.StatusOK, gin.H{"data": prompts})
}

func (h *LearningHandler) MarkRealTalkUsed(c *gin.Context) {
	// Nudge usage is recorded client-side; the endpoint is a safe acceptance point.
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

func (h *LearningHandler) NudgeDismiss(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

func (h *LearningHandler) RecoverStreak(c *gin.Context) {
	target := c.Query("targetLanguage")
	native := c.Query("nativeLanguage")
	// Streak recovery books today's activity so the current-at-risk streak is
	// preserved for today.
	userID := c.GetString("userID")
	_, _ = h.sessions.BookRecovery(c.Request.Context(), userID, target, native)
	c.JSON(http.StatusOK, gin.H{"data": models.StreakRecoverResult{Recovered: true}})
}
