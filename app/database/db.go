package database

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite connection.
type DB struct {
	conn *sql.DB
}

// Plan represents a high-level plan that groups related tasks.
type Plan struct {
	ID             int64
	Title          string
	Description    string
	CreatedAt      string
	Archived       int
	TaskCount      int
	CompletedCount int
}

// Task represents a single actionable item belonging to a plan.
type Task struct {
	ID        int64
	PlanID    int64
	Title     string
	Notes     string
	Category  string
	Status    string
	DueDate   string
	CreatedAt string
	Archived  int
	PlanTitle string
}

// TaskStats holds aggregate counts for the dashboard.
type TaskStats struct {
	Total      int
	Completed  int
	InProgress int
	NotStarted int
	Overdue    int
}

// TaskFilters holds optional filters for GetAllTasks.
type TaskFilters struct {
	PlanID   string
	Status   string
	Category string
}

// New opens (or creates) the SQLite database at dbPath and runs migrations.
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	stmts := []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS plans (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			title       TEXT    NOT NULL,
			description TEXT,
			created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
			archived    INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			plan_id    INTEGER NOT NULL,
			title      TEXT    NOT NULL,
			notes      TEXT,
			category   TEXT,
			status     TEXT    NOT NULL DEFAULT 'not_started',
			due_date   TEXT,
			created_at TEXT    NOT NULL DEFAULT (datetime('now')),
			archived   INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (plan_id) REFERENCES plans(id)
		)`,
	}
	for _, s := range stmts {
		if _, err := db.conn.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

// SeedDevData inserts sample plans and tasks if the database is empty.
func (db *DB) SeedDevData() {
	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM plans").Scan(&count)
	if count > 0 {
		return
	}

	r1, _ := db.conn.Exec("INSERT INTO plans (title, description) VALUES (?, ?)",
		"CS 408 Project", "Full-stack web application project for Spring 2026")
	plan1, _ := r1.LastInsertId()

	r2, _ := db.conn.Exec("INSERT INTO plans (title, description) VALUES (?, ?)",
		"Personal Goals", "Things to accomplish this semester")
	plan2, _ := r2.LastInsertId()

	seed := []struct {
		planID   int64
		title    string
		notes    string
		category string
		status   string
		dueDate  string
	}{
		{plan1, "Set up database schema", "Create SQLite tables for plans and tasks", "School", "completed", "2026-03-15"},
		{plan1, "Implement CRUD routes", "Backend routes for all CRUD operations", "School", "in_progress", "2026-03-22"},
		{plan1, "Build frontend templates", "Go HTML templates with Bootstrap styling", "School", "not_started", "2026-03-29"},
		{plan2, "Exercise 3x per week", "Hit the gym consistently", "Personal", "in_progress", "2026-05-01"},
		{plan2, "Read two books", "For personal growth", "Personal", "not_started", "2026-05-15"},
	}
	for _, t := range seed {
		db.conn.Exec(
			"INSERT INTO tasks (plan_id, title, notes, category, status, due_date) VALUES (?, ?, ?, ?, ?, ?)",
			t.planID, t.title, t.notes, t.category, t.status, t.dueDate,
		)
	}
	log.Println("Dev seed data inserted")
}

// SeedTestData clears the database and inserts known test fixtures.
func (db *DB) SeedTestData() {
	db.ClearDatabase()

	r1, _ := db.conn.Exec("INSERT INTO plans (title, description) VALUES (?, ?)",
		"CS 408 Project", "Full-stack web application project")
	plan1, _ := r1.LastInsertId()

	r2, _ := db.conn.Exec("INSERT INTO plans (title, description) VALUES (?, ?)",
		"Study Schedule", "Weekly study plan for finals")
	plan2, _ := r2.LastInsertId()

	db.conn.Exec("INSERT INTO tasks (plan_id, title, notes, category, status, due_date) VALUES (?, ?, ?, ?, ?, ?)",
		plan1, "Set up database schema", "Create tables", "School", "completed", "2026-03-15")
	db.conn.Exec("INSERT INTO tasks (plan_id, title, notes, category, status, due_date) VALUES (?, ?, ?, ?, ?, ?)",
		plan1, "Implement CRUD routes", "Backend API routes", "School", "in_progress", "2026-03-22")
	db.conn.Exec("INSERT INTO tasks (plan_id, title, notes, category, status, due_date) VALUES (?, ?, ?, ?, ?, ?)",
		plan1, "Build frontend templates", "Go HTML templates", "School", "not_started", "2026-03-29")
	db.conn.Exec("INSERT INTO tasks (plan_id, title, notes, category, status, due_date) VALUES (?, ?, ?, ?, ?, ?)",
		plan2, "Review lecture notes", "Go over chapters 5-8", "School", "not_started", "2026-03-28")
	db.conn.Exec("INSERT INTO tasks (plan_id, title, notes, category, status, due_date) VALUES (?, ?, ?, ?, ?, ?)",
		plan2, "Study group meeting", "Meet with team at library", "School", "in_progress", "2026-03-26")

	log.Println("Test seed data inserted")
}

// ClearDatabase removes all rows from tasks and plans.
func (db *DB) ClearDatabase() {
	db.conn.Exec("DELETE FROM tasks")
	db.conn.Exec("DELETE FROM plans")
}

// ─── Plans ────────────────────────────────────────────────────────────────────

// GetAllPlans returns all non-archived plans with their task and completion counts.
func (db *DB) GetAllPlans() ([]Plan, error) {
	rows, err := db.conn.Query(`
		SELECT p.id, p.title, COALESCE(p.description,''), p.created_at, p.archived,
		       COUNT(t.id) AS task_count,
		       SUM(CASE WHEN t.status = 'completed' THEN 1 ELSE 0 END) AS completed_count
		FROM plans p
		LEFT JOIN tasks t ON t.plan_id = p.id AND t.archived = 0
		WHERE p.archived = 0
		GROUP BY p.id
		ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []Plan
	for rows.Next() {
		var p Plan
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.CreatedAt, &p.Archived,
			&p.TaskCount, &p.CompletedCount); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, nil
}

// GetPlanByID returns a single plan by ID, or nil if not found.
func (db *DB) GetPlanByID(id int64) (*Plan, error) {
	var p Plan
	err := db.conn.QueryRow(
		`SELECT id, title, COALESCE(description,''), created_at, archived
		 FROM plans WHERE id = ? AND archived = 0`, id,
	).Scan(&p.ID, &p.Title, &p.Description, &p.CreatedAt, &p.Archived)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// CreatePlan inserts a new plan and returns its ID.
func (db *DB) CreatePlan(title, description string) (int64, error) {
	r, err := db.conn.Exec("INSERT INTO plans (title, description) VALUES (?, ?)", title, description)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

// DeletePlan removes a plan and all its tasks.
func (db *DB) DeletePlan(id int64) error {
	db.conn.Exec("DELETE FROM tasks WHERE plan_id = ?", id)
	_, err := db.conn.Exec("DELETE FROM plans WHERE id = ?", id)
	return err
}

// ─── Tasks ────────────────────────────────────────────────────────────────────

// GetAllTasks returns all non-archived tasks, optionally filtered.
func (db *DB) GetAllTasks(f TaskFilters) ([]Task, error) {
	query := `
		SELECT t.id, t.plan_id, t.title, COALESCE(t.notes,''), COALESCE(t.category,''),
		       t.status, COALESCE(t.due_date,''), t.created_at, t.archived, COALESCE(p.title,'')
		FROM tasks t
		LEFT JOIN plans p ON t.plan_id = p.id
		WHERE t.archived = 0`
	args := []any{}

	if f.PlanID != "" {
		query += " AND t.plan_id = ?"
		args = append(args, f.PlanID)
	}
	if f.Status != "" {
		query += " AND t.status = ?"
		args = append(args, f.Status)
	}
	if f.Category != "" {
		query += " AND t.category = ?"
		args = append(args, f.Category)
	}
	query += " ORDER BY CASE WHEN t.due_date = '' THEN 1 ELSE 0 END, t.due_date ASC, t.created_at DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.PlanID, &t.Title, &t.Notes, &t.Category,
			&t.Status, &t.DueDate, &t.CreatedAt, &t.Archived, &t.PlanTitle); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// GetTaskByID returns a single task by ID (with plan title), or nil if not found.
func (db *DB) GetTaskByID(id int64) (*Task, error) {
	var t Task
	err := db.conn.QueryRow(`
		SELECT t.id, t.plan_id, t.title, COALESCE(t.notes,''), COALESCE(t.category,''),
		       t.status, COALESCE(t.due_date,''), t.created_at, t.archived, COALESCE(p.title,'')
		FROM tasks t
		LEFT JOIN plans p ON t.plan_id = p.id
		WHERE t.id = ?`, id,
	).Scan(&t.ID, &t.PlanID, &t.Title, &t.Notes, &t.Category,
		&t.Status, &t.DueDate, &t.CreatedAt, &t.Archived, &t.PlanTitle)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetTasksByPlanID returns all non-archived tasks for a given plan.
func (db *DB) GetTasksByPlanID(planID int64) ([]Task, error) {
	rows, err := db.conn.Query(`
		SELECT id, plan_id, title, COALESCE(notes,''), COALESCE(category,''),
		       status, COALESCE(due_date,''), created_at, archived, ''
		FROM tasks
		WHERE plan_id = ? AND archived = 0
		ORDER BY CASE WHEN due_date = '' THEN 1 ELSE 0 END, due_date ASC`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.PlanID, &t.Title, &t.Notes, &t.Category,
			&t.Status, &t.DueDate, &t.CreatedAt, &t.Archived, &t.PlanTitle); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// CreateTask inserts a new task and returns its ID.
func (db *DB) CreateTask(planID int64, title, notes, category, status, dueDate string) (int64, error) {
	if status == "" {
		status = "not_started"
	}
	r, err := db.conn.Exec(
		`INSERT INTO tasks (plan_id, title, notes, category, status, due_date)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		planID, title, notes, category, status, dueDate,
	)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

// UpdateTaskStatus sets the status field of a task.
func (db *DB) UpdateTaskStatus(id int64, status string) error {
	_, err := db.conn.Exec("UPDATE tasks SET status = ? WHERE id = ?", status, id)
	return err
}

// DeleteTask removes a task by ID.
func (db *DB) DeleteTask(id int64) error {
	_, err := db.conn.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}

// GetCategories returns the distinct non-empty categories in use.
func (db *DB) GetCategories() ([]string, error) {
	rows, err := db.conn.Query(
		"SELECT DISTINCT category FROM tasks WHERE category != '' AND archived = 0 ORDER BY category")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var c string
		rows.Scan(&c)
		cats = append(cats, c)
	}
	return cats, nil
}

// GetTaskStats returns aggregate counts for the dashboard.
func (db *DB) GetTaskStats() (TaskStats, error) {
	var s TaskStats
	db.conn.QueryRow("SELECT COUNT(*) FROM tasks WHERE archived = 0").Scan(&s.Total)
	db.conn.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'completed' AND archived = 0").Scan(&s.Completed)
	db.conn.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'in_progress' AND archived = 0").Scan(&s.InProgress)
	db.conn.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'not_started' AND archived = 0").Scan(&s.NotStarted)
	db.conn.QueryRow("SELECT COUNT(*) FROM tasks WHERE due_date < date('now') AND status != 'completed' AND archived = 0").Scan(&s.Overdue)
	return s, nil
}

// GetOverdueTasks returns tasks whose due date has passed and are not completed.
func (db *DB) GetOverdueTasks() ([]Task, error) {
	rows, err := db.conn.Query(`
		SELECT t.id, t.plan_id, t.title, COALESCE(t.notes,''), COALESCE(t.category,''),
		       t.status, COALESCE(t.due_date,''), t.created_at, t.archived, COALESCE(p.title,'')
		FROM tasks t
		LEFT JOIN plans p ON t.plan_id = p.id
		WHERE t.due_date < date('now') AND t.status != 'completed' AND t.archived = 0
		ORDER BY t.due_date ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

// GetUpcomingTasks returns incomplete tasks due within the next 7 days.
func (db *DB) GetUpcomingTasks() ([]Task, error) {
	rows, err := db.conn.Query(`
		SELECT t.id, t.plan_id, t.title, COALESCE(t.notes,''), COALESCE(t.category,''),
		       t.status, COALESCE(t.due_date,''), t.created_at, t.archived, COALESCE(p.title,'')
		FROM tasks t
		LEFT JOIN plans p ON t.plan_id = p.id
		WHERE t.due_date >= date('now') AND t.due_date <= date('now', '+7 days')
		  AND t.status != 'completed' AND t.archived = 0
		ORDER BY t.due_date ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func scanTasks(rows *sql.Rows) ([]Task, error) {
	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.PlanID, &t.Title, &t.Notes, &t.Category,
			&t.Status, &t.DueDate, &t.CreatedAt, &t.Archived, &t.PlanTitle); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
