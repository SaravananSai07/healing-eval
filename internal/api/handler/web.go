package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/saisaravanan/healing-eval/internal/domain"
	"github.com/saisaravanan/healing-eval/internal/storage"
)

type WebHandler struct {
	templates map[string]*template.Template
	convRepo  *storage.ConversationRepo
	evalRepo  *storage.EvaluationRepo
	suggRepo  *storage.SuggestionRepo
}

func NewWebHandler(convRepo *storage.ConversationRepo, evalRepo *storage.EvaluationRepo, suggRepo *storage.SuggestionRepo) *WebHandler {
	funcMap := template.FuncMap{
		"scoreClass": func(score float64) string {
			if score >= 0.8 {
				return "score-high"
			}
			if score >= 0.6 {
				return "score-medium"
			}
			return "score-low"
		},
		"statusClass": func(status interface{}) string {
			s := fmt.Sprintf("%v", status)
			switch s {
			case "pending":
				return "pending"
			case "approved", "applied", "success":
				return "success"
			case "rejected", "error", "failed":
				return "error"
			default:
				return "pending"
			}
		},
		"formatTime": func(t time.Time) string {
			return t.Format("Jan 02 15:04")
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"toJSON": func(v interface{}) string {
			if v == nil {
				return "{}"
			}
			switch val := v.(type) {
			case []byte:
				return string(val)
			case json.RawMessage:
				return string(val)
			case string:
				return val
			default:
				b, err := json.Marshal(v)
				if err != nil {
					return fmt.Sprintf("%v", v)
				}
				return string(b)
			}
		},
		"formatEvaluatorType": func(t interface{}) string {
			s := fmt.Sprintf("%v", t)
			switch s {
			case "llm_judge":
				return "LLM Judge"
			case "tool_call":
				return "Tool Call"
			case "coherence":
				return "Coherence"
			case "heuristic":
				return "Heuristic"
			default:
				return strings.Title(strings.ReplaceAll(s, "_", " "))
			}
		},
		"formatIssueType": func(t string) string {
			switch t {
			case "context_loss":
				return "Context Loss"
			case "reference_handling":
				return "Reference Handling"
			case "hallucination":
				return "Hallucination"
			case "tool_error":
				return "Tool Error"
			case "format_error":
				return "Format Error"
			case "latency":
				return "High Latency"
			case "warning":
				return "Warning"
			case "minor":
				return "Minor Issue"
			default:
				return strings.Title(strings.ReplaceAll(t, "_", " "))
			}
		},
		"formatSeverity": func(s string) string {
			switch s {
			case "info":
				return "Info"
			case "warning":
				return "Warning"
			case "error":
				return "Error"
			case "critical":
				return "Critical"
			default:
				return strings.Title(s)
			}
		},
		"formatStatus": func(s interface{}) string {
			str := fmt.Sprintf("%v", s)
			switch str {
			case "pending":
				return "Pending"
			case "approved":
				return "Approved"
			case "rejected":
				return "Rejected"
			case "applied":
				return "Applied"
			case "success":
				return "Success"
			case "error", "failed":
				return "Failed"
			default:
				return strings.Title(str)
			}
		},
		"formatSuggestionType": func(t interface{}) string {
			s := fmt.Sprintf("%v", t)
			switch s {
			case "prompt":
				return "Prompt"
			case "tool":
				return "Tool"
			case "threshold":
				return "Threshold"
			default:
				return strings.Title(s)
			}
		},
	}

	templates := make(map[string]*template.Template)
	pages := []string{"dashboard", "suggestions", "metrics", "conversations", "reviews"}

	for _, page := range pages {
		t := template.Must(template.New("").Funcs(funcMap).ParseFiles(
			"web/templates/layout.html",
			"web/templates/"+page+".html",
		))
		templates[page] = t
	}

	partials := template.Must(template.New("").Funcs(funcMap).ParseFiles("web/templates/partials.html"))
	templates["partials"] = partials

	return &WebHandler{
		templates: templates,
		convRepo:  convRepo,
		evalRepo:  evalRepo,
		suggRepo:  suggRepo,
	}
}

type PageData struct {
	Title string
	Page  string
	Data  interface{}
}

func (h *WebHandler) Dashboard(c *gin.Context) {
	h.renderPage(c, "dashboard", PageData{
		Title: "Dashboard",
		Page:  "dashboard",
	})
}

func (h *WebHandler) Conversations(c *gin.Context) {
	h.renderPage(c, "conversations", PageData{
		Title: "Conversations",
		Page:  "conversations",
	})
}

func (h *WebHandler) Suggestions(c *gin.Context) {
	h.renderPage(c, "suggestions", PageData{
		Title: "Suggestions",
		Page:  "suggestions",
	})
}

func (h *WebHandler) Metrics(c *gin.Context) {
	h.renderPage(c, "metrics", PageData{
		Title: "Metrics",
		Page:  "metrics",
	})
}

func (h *WebHandler) Reviews(c *gin.Context) {
	h.renderPage(c, "reviews", PageData{
		Title: "Human Review Queue",
		Page:  "reviews",
	})
}

func (h *WebHandler) renderPage(c *gin.Context, name string, data PageData) {
	tmpl, ok := h.templates[name]
	if !ok {
		c.String(http.StatusInternalServerError, "Template not found: %s", name)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(c.Writer, "layout.html", data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

func (h *WebHandler) RecentEvaluations(c *gin.Context) {
	ctx := c.Request.Context()

	resp, err := h.evalRepo.Query(ctx, &domain.EvaluationsQueryRequest{
		Limit:     10,
		SortBy:    "created_at",
		SortOrder: "desc",
	})

	var evals []domain.Evaluation
	if err == nil && resp != nil {
		evals = resp.Evaluations
	}

	data := struct {
		Evaluations []domain.Evaluation
	}{
		Evaluations: evals,
	}

	h.renderPartial(c, "recent-evaluations", data)
}

func (h *WebHandler) ActiveIssues(c *gin.Context) {
	ctx := c.Request.Context()

	resp, err := h.evalRepo.Query(ctx, &domain.EvaluationsQueryRequest{
		HasIssues: boolPtr(true),
		Limit:     100,
	})

	issueMap := make(map[string]map[string]int)
	if err == nil && resp != nil {
		for _, eval := range resp.Evaluations {
			for _, issue := range eval.Issues {
				if issueMap[issue.Type] == nil {
					issueMap[issue.Type] = make(map[string]int)
				}
				issueMap[issue.Type][issue.Severity]++
			}
		}
	}

	type IssueCount struct {
		Type     string
		Severity string
		Count    int
	}

	var issues []IssueCount
	for issueType, severities := range issueMap {
		for severity, count := range severities {
			issues = append(issues, IssueCount{
				Type:     issueType,
				Severity: severity,
				Count:    count,
			})
		}
	}

	data := struct {
		Issues []IssueCount
	}{
		Issues: issues,
	}

	h.renderPartial(c, "active-issues", data)
}

func (h *WebHandler) EvaluatorPerformance(c *gin.Context) {
	ctx := c.Request.Context()

	evalTypes := []domain.EvaluatorType{
		domain.EvaluatorTypeLLMJudge,
		domain.EvaluatorTypeToolCall,
		domain.EvaluatorTypeCoherence,
		domain.EvaluatorTypeHeuristic,
	}

	type EvalStats struct {
		EvaluatorType string
		Count         int
		AvgScore      float64
	}

	var stats []EvalStats
	for _, et := range evalTypes {
		resp, err := h.evalRepo.Query(ctx, &domain.EvaluationsQueryRequest{
			EvaluatorTypes: []domain.EvaluatorType{et},
			Limit:          500,
		})

		if err != nil || resp == nil || len(resp.Evaluations) == 0 {
			continue
		}

		var totalScore float64
		for _, eval := range resp.Evaluations {
			totalScore += eval.Scores.Overall
		}

		stats = append(stats, EvalStats{
			EvaluatorType: string(et),
			Count:         resp.Total,
			AvgScore:      totalScore / float64(len(resp.Evaluations)),
		})
	}

	data := struct {
		Evaluators []EvalStats
	}{
		Evaluators: stats,
	}

	h.renderPartial(c, "evaluator-performance", data)
}

func (h *WebHandler) SuggestionsTable(c *gin.Context) {
	ctx := c.Request.Context()
	statusParam := c.Query("status")

	var status *domain.SuggestionStatus
	if statusParam != "" {
		s := domain.SuggestionStatus(statusParam)
		status = &s
	}

	suggestions, err := h.suggRepo.List(ctx, status, 50, 0)
	if err != nil {
		suggestions = []*domain.Suggestion{}
	}

	data := struct {
		Suggestions []*domain.Suggestion
	}{
		Suggestions: suggestions,
	}

	h.renderPartial(c, "suggestions-table", data)
}

func (h *WebHandler) EvaluatorAccuracy(c *gin.Context) {
	ctx := c.Request.Context()

	evalTypes := []domain.EvaluatorType{
		domain.EvaluatorTypeLLMJudge,
		domain.EvaluatorTypeToolCall,
		domain.EvaluatorTypeCoherence,
		domain.EvaluatorTypeHeuristic,
	}

	type AccuracyStats struct {
		EvaluatorType string
		Count         int
		AvgConfidence float64
		IssuesFound   int
	}

	var stats []AccuracyStats
	for _, et := range evalTypes {
		resp, err := h.evalRepo.Query(ctx, &domain.EvaluationsQueryRequest{
			EvaluatorTypes: []domain.EvaluatorType{et},
			Limit:          500,
		})

		if err != nil || resp == nil || len(resp.Evaluations) == 0 {
			continue
		}

		var totalConf float64
		var totalIssues int
		for _, eval := range resp.Evaluations {
			totalConf += eval.Confidence
			totalIssues += len(eval.Issues)
		}

		stats = append(stats, AccuracyStats{
			EvaluatorType: string(et),
			Count:         resp.Total,
			AvgConfidence: totalConf / float64(len(resp.Evaluations)),
			IssuesFound:   totalIssues,
		})
	}

	data := struct {
		Evaluators []AccuracyStats
	}{
		Evaluators: stats,
	}

	h.renderPartial(c, "evaluator-accuracy", data)
}

func (h *WebHandler) Calibration(c *gin.Context) {
	data := struct {
		Message string
	}{
		Message: "Calibration requires human labels for comparison",
	}

	h.renderPartial(c, "calibration", data)
}

func (h *WebHandler) BlindSpots(c *gin.Context) {
	data := struct {
		BlindSpots []struct{}
	}{
		BlindSpots: []struct{}{},
	}

	h.renderPartial(c, "blind-spots", data)
}

func (h *WebHandler) renderPartial(c *gin.Context, name string, data interface{}) {
	tmpl := h.templates["partials"]
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

func (h *WebHandler) StatConversations(c *gin.Context) {
	ctx := c.Request.Context()
	convs, _ := h.convRepo.GetUnprocessed(ctx, 0)

	count := len(convs)
	resp, err := h.evalRepo.Query(ctx, &domain.EvaluationsQueryRequest{Limit: 1})
	if err == nil && resp != nil {
		distinctConvs := make(map[string]bool)
		allResp, _ := h.evalRepo.Query(ctx, &domain.EvaluationsQueryRequest{Limit: 10000})
		if allResp != nil {
			for _, e := range allResp.Evaluations {
				distinctConvs[e.ConversationID] = true
			}
		}
		count = len(distinctConvs)
	}

	c.String(http.StatusOK, "%d", count)
}

func (h *WebHandler) StatEvaluations(c *gin.Context) {
	ctx := c.Request.Context()
	resp, err := h.evalRepo.Query(ctx, &domain.EvaluationsQueryRequest{Limit: 1})
	if err != nil || resp == nil {
		c.String(http.StatusOK, "0")
		return
	}
	c.String(http.StatusOK, "%d", resp.Total)
}

func (h *WebHandler) StatSuggestions(c *gin.Context) {
	ctx := c.Request.Context()
	suggestions, err := h.suggRepo.List(ctx, nil, 1000, 0)
	if err != nil {
		c.String(http.StatusOK, "0")
		return
	}
	c.String(http.StatusOK, "%d", len(suggestions))
}

func (h *WebHandler) StatAvgScore(c *gin.Context) {
	ctx := c.Request.Context()
	resp, err := h.evalRepo.Query(ctx, &domain.EvaluationsQueryRequest{Limit: 500})
	if err != nil || resp == nil || len(resp.Evaluations) == 0 {
		c.String(http.StatusOK, "-")
		return
	}

	var total float64
	for _, eval := range resp.Evaluations {
		total += eval.Scores.Overall
	}
	avg := total / float64(len(resp.Evaluations))
	c.String(http.StatusOK, "%.2f", avg)
}

func (h *WebHandler) ConversationsList(c *gin.Context) {
	ctx := c.Request.Context()
	pageStr := c.DefaultQuery("page", "1")
	issueFilter := c.Query("issue")
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	perPage := 10

	resp, err := h.evalRepo.Query(ctx, &domain.EvaluationsQueryRequest{
		Limit:     1000,
		SortBy:    "created_at",
		SortOrder: "desc",
	})

	convMap := make(map[string]*ConversationSummary)
	convHasIssue := make(map[string]bool)

	if err == nil && resp != nil {
		for _, eval := range resp.Evaluations {
			if _, exists := convMap[eval.ConversationID]; !exists {
				convMap[eval.ConversationID] = &ConversationSummary{
					ID:        eval.ConversationID,
					CreatedAt: eval.CreatedAt,
				}
			}
			summary := convMap[eval.ConversationID]
			summary.EvalCount++
			summary.TotalScore += eval.Scores.Overall
			summary.IssueCount += len(eval.Issues)

			if issueFilter != "" {
				for _, issue := range eval.Issues {
					if issue.Type == issueFilter {
						convHasIssue[eval.ConversationID] = true
					}
				}
			}
		}
	}

	var conversations []ConversationSummary
	for _, s := range convMap {
		if issueFilter != "" && !convHasIssue[s.ID] {
			continue
		}
		s.AvgScore = s.TotalScore / float64(s.EvalCount)
		conversations = append(conversations, *s)
	}

	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].CreatedAt.After(conversations[j].CreatedAt)
	})

	totalConvs := len(conversations)
	totalPages := (totalConvs + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	start := (page - 1) * perPage
	end := start + perPage
	if start > totalConvs {
		start = totalConvs
	}
	if end > totalConvs {
		end = totalConvs
	}

	paginatedConvs := []ConversationSummary{}
	if start < totalConvs {
		paginatedConvs = conversations[start:end]
	}

	data := struct {
		Conversations []ConversationSummary
		CurrentPage   int
		TotalPages    int
		Total         int
		IssueFilter   string
	}{
		Conversations: paginatedConvs,
		CurrentPage:   page,
		TotalPages:    totalPages,
		Total:         totalConvs,
		IssueFilter:   issueFilter,
	}

	h.renderPartial(c, "conversations-list", data)
}

type ConversationSummary struct {
	ID         string
	EvalCount  int
	AvgScore   float64
	TotalScore float64
	IssueCount int
	CreatedAt  time.Time
}

func (h *WebHandler) ConversationDetail(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	conv, err := h.convRepo.GetByID(ctx, id)
	if err != nil || conv == nil {
		c.String(http.StatusNotFound, "Conversation not found")
		return
	}

	evals, _ := h.evalRepo.GetByConversationID(ctx, id)

	data := struct {
		Conversation *domain.Conversation
		Evaluations  []*domain.Evaluation
	}{
		Conversation: conv,
		Evaluations:  evals,
	}

	h.renderPartial(c, "conversation-detail", data)
}

func boolPtr(b bool) *bool {
	return &b
}
