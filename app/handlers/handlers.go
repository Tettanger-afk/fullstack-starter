package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"planit/database"
)

// TemplateData is the generic data bag passed to every template.
type TemplateData struct {
	Title         string
	CurrentPath   string
	Plans         []database.Plan
	Plan          *database.Plan
	Tasks         []database.Task
	Task          *database.Task
	Categories    []string
	Filters       database.TaskFilters
	Stats         database.TaskStats
	OverdueTasks  []database.Task
	UpcomingTasks []database.Task
	Error         string
}

// Handler holds the database connection and template helper functions.
type Handler struct {
	db      *database.DB
	funcMap template.FuncMap
}

// NewMux wires up all routes and returns the http.Handler for the application.
func NewMux(db *database.DB) http.Handler {
	h := &Handler{
		db: db,
		funcMap: template.FuncMap{
			"formatDate": func(dateStr string) string {
				if dateStr == "" {
					return ""
				}
				s := dateStr
				if len(s) > 10 {
					s = s[:10]
				}
				t, err := time.Parse("2006-01-02", s)
				if err != nil {
					return dateStr
				}
				return t.Format("Jan 2, 2006")
			},
			"isOverdue": func(dateStr, status string) bool {
				if dateStr == "" || status == "completed" {
					return false
				}
				today := time.Now().Format("2006-01-02")
				return dateStr < today
			},
			"isDueSoon": func(dateStr, status string) bool {
				if dateStr == "" || status == "completed" {
					return false
				}
				today := time.Now().Format("2006-01-02")
				sevenDays := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
				return dateStr >= today && dateStr <= sevenDays
			},
			"statusLabel": func(status string) string {
				switch status {
				case "not_started":
					return "Not Started"
				case "in_progress":
					return "In Progress"
				case "completed":
					return "Completed"
				default:
					return status
				}
			},
			"statusBadge": func(status string) string {
				switch status {
				case "not_started":
					return "secondary"
				case "in_progress":
					return "warning"
				case "completed":
					return "success"
				default:
					return "secondary"
				}
			},
			"statusIcon": func(status string) string {
				switch status {
				case "not_started":
					return "bi-circle"
				case "in_progress":
					return "bi-arrow-repeat"
				case "completed":
					return "bi-check-circle-fill"
				default:
					return "bi-circle"
				}
			},
			"completionPct": func(completed, total int) int {
				if total == 0 {
					return 0
				}
				return completed * 100 / total
			},
			"hasPrefix": strings.HasPrefix,
			"eq":        func(a, b string) bool { return a == b },
		},
	}

	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Landing page — exact "/" only
	mux.HandleFunc("GET /{$}", h.index)

	// Dashboard
	mux.HandleFunc("GET /dashboard", h.dashboard)

	// Plans
	mux.HandleFunc("GET /plans", h.plansIndex)
	mux.HandleFunc("GET /plans/new", h.plansNew)
	mux.HandleFunc("POST /plans", h.plansCreate)
	mux.HandleFunc("GET /plans/{id}", h.planShow)
	mux.HandleFunc("POST /plans/{id}/delete", h.planDelete)

	// Tasks — /tasks/new must be registered before /tasks/{id}
	mux.HandleFunc("GET /tasks", h.tasksIndex)
	mux.HandleFunc("GET /tasks/new", h.tasksNew)
	mux.HandleFunc("POST /tasks", h.tasksCreate)
	mux.HandleFunc("GET /tasks/{id}", h.taskShow)
	mux.HandleFunc("POST /tasks/{id}/status", h.taskUpdateStatus)
	mux.HandleFunc("POST /tasks/{id}/delete", h.taskDelete)

	// Catch-all 404
	mux.HandleFunc("/", h.notFound)

	return mux
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (h *Handler) render(w http.ResponseWriter, r *http.Request, page string, data TemplateData) {
	data.CurrentPath = r.URL.Path
	t, err := template.New("").Funcs(h.funcMap).ParseFiles(
		"templates/layout.html",
		"templates/"+page+".html",
	)
	if err != nil {
		log.Printf("template parse error (%s): %v", page, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("template execute error (%s): %v", page, err)
	}
}

func (h *Handler) notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	h.render(w, r, "404", TemplateData{Title: "404 – Page Not Found"})
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// ─── Landing ──────────────────────────────────────────────────────────────────

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "index", TemplateData{Title: "Home"})
}

// ─── Dashboard ────────────────────────────────────────────────────────────────

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	stats, _ := h.db.GetTaskStats()
	overdue, _ := h.db.GetOverdueTasks()
	upcoming, _ := h.db.GetUpcomingTasks()
	h.render(w, r, "dashboard", TemplateData{
		Title:         "Dashboard",
		Stats:         stats,
		OverdueTasks:  overdue,
		UpcomingTasks: upcoming,
	})
}

// ─── Plans ────────────────────────────────────────────────────────────────────

func (h *Handler) plansIndex(w http.ResponseWriter, r *http.Request) {
	plans, _ := h.db.GetAllPlans()
	h.render(w, r, "plans_index", TemplateData{Title: "My Plans", Plans: plans})
}

func (h *Handler) plansNew(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "plans_new", TemplateData{Title: "Create Plan"})
}

func (h *Handler) plansCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))

	if title == "" {
		h.render(w, r, "plans_new", TemplateData{Title: "Create Plan", Error: "Title is required."})
		return
	}
	if _, err := h.db.CreatePlan(title, description); err != nil {
		h.render(w, r, "plans_new", TemplateData{Title: "Create Plan", Error: "Failed to create plan."})
		return
	}
	http.Redirect(w, r, "/plans", http.StatusSeeOther)
}

func (h *Handler) planShow(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.notFound(w, r)
		return
	}
	plan, err := h.db.GetPlanByID(id)
	if err != nil || plan == nil {
		h.notFound(w, r)
		return
	}
	tasks, _ := h.db.GetTasksByPlanID(id)
	h.render(w, r, "plans_show", TemplateData{Title: plan.Title, Plan: plan, Tasks: tasks})
}

func (h *Handler) planDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.notFound(w, r)
		return
	}
	h.db.DeletePlan(id)
	http.Redirect(w, r, "/plans", http.StatusSeeOther)
}

// ─── Tasks ────────────────────────────────────────────────────────────────────

func (h *Handler) tasksIndex(w http.ResponseWriter, r *http.Request) {
	filters := database.TaskFilters{
		PlanID:   r.URL.Query().Get("plan_id"),
		Status:   r.URL.Query().Get("status"),
		Category: r.URL.Query().Get("category"),
	}
	tasks, _ := h.db.GetAllTasks(filters)
	plans, _ := h.db.GetAllPlans()
	categories, _ := h.db.GetCategories()
	h.render(w, r, "tasks_index", TemplateData{
		Title:      "My Tasks",
		Tasks:      tasks,
		Plans:      plans,
		Categories: categories,
		Filters:    filters,
	})
}

func (h *Handler) tasksNew(w http.ResponseWriter, r *http.Request) {
	plans, _ := h.db.GetAllPlans()
	h.render(w, r, "tasks_new", TemplateData{Title: "Create Task", Plans: plans})
}

func (h *Handler) tasksCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	planIDStr := r.FormValue("plan_id")
	title := strings.TrimSpace(r.FormValue("title"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	category := r.FormValue("category")
	status := r.FormValue("status")
	dueDate := r.FormValue("due_date")

	planID, err := strconv.ParseInt(planIDStr, 10, 64)
	if title == "" || err != nil {
		plans, _ := h.db.GetAllPlans()
		h.render(w, r, "tasks_new", TemplateData{
			Title: "Create Task", Plans: plans, Error: "Title and plan are required.",
		})
		return
	}
	if _, err := h.db.CreateTask(planID, title, notes, category, status, dueDate); err != nil {
		plans, _ := h.db.GetAllPlans()
		h.render(w, r, "tasks_new", TemplateData{
			Title: "Create Task", Plans: plans, Error: "Failed to create task.",
		})
		return
	}
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

func (h *Handler) taskShow(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.notFound(w, r)
		return
	}
	task, err := h.db.GetTaskByID(id)
	if err != nil || task == nil {
		h.notFound(w, r)
		return
	}
	h.render(w, r, "tasks_show", TemplateData{Title: task.Title, Task: task})
}

func (h *Handler) taskUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.notFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	h.db.UpdateTaskStatus(id, r.FormValue("status"))
	http.Redirect(w, r, "/tasks/"+r.PathValue("id"), http.StatusSeeOther)
}

func (h *Handler) taskDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.notFound(w, r)
		return
	}
	h.db.DeleteTask(id)
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}
