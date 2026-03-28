package database

import (
	"os"
	"testing"
)

// newTestDB creates a temporary in-memory (file-based) SQLite database for testing.
func newTestDB(t *testing.T) *DB {
	t.Helper()
	f, err := os.CreateTemp("", "planit-test-*.sqlite")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	path := f.Name()
	f.Close()

	db, err := New(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(path)
	})
	return db
}

// ─── Plan tests ───────────────────────────────────────────────────────────────

func TestCreateAndGetPlan(t *testing.T) {
	db := newTestDB(t)

	id, err := db.CreatePlan("Test Plan", "A description")
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero plan ID")
	}

	plan, err := db.GetPlanByID(id)
	if err != nil {
		t.Fatalf("GetPlanByID: %v", err)
	}
	if plan == nil {
		t.Fatal("expected plan, got nil")
	}
	if plan.Title != "Test Plan" {
		t.Errorf("title = %q, want %q", plan.Title, "Test Plan")
	}
	if plan.Description != "A description" {
		t.Errorf("description = %q, want %q", plan.Description, "A description")
	}
}

func TestGetPlanByID_NotFound(t *testing.T) {
	db := newTestDB(t)

	plan, err := db.GetPlanByID(9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan != nil {
		t.Errorf("expected nil plan for missing ID, got %+v", plan)
	}
}

func TestGetAllPlans(t *testing.T) {
	db := newTestDB(t)

	db.CreatePlan("Plan A", "")
	db.CreatePlan("Plan B", "desc")

	plans, err := db.GetAllPlans()
	if err != nil {
		t.Fatalf("GetAllPlans: %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("got %d plans, want 2", len(plans))
	}
}

func TestDeletePlan(t *testing.T) {
	db := newTestDB(t)

	id, _ := db.CreatePlan("To Delete", "")
	if err := db.DeletePlan(id); err != nil {
		t.Fatalf("DeletePlan: %v", err)
	}

	plan, _ := db.GetPlanByID(id)
	if plan != nil {
		t.Error("expected plan to be deleted, but it still exists")
	}
}

func TestDeletePlanAlsoDeletesTasks(t *testing.T) {
	db := newTestDB(t)

	planID, _ := db.CreatePlan("Plan with tasks", "")
	taskID, _ := db.CreateTask(planID, "Orphan task", "", "", "not_started", "")

	db.DeletePlan(planID)

	task, _ := db.GetTaskByID(taskID)
	if task != nil {
		t.Error("expected task to be deleted when plan was deleted")
	}
}

// ─── Task tests ───────────────────────────────────────────────────────────────

func TestCreateAndGetTask(t *testing.T) {
	db := newTestDB(t)

	planID, _ := db.CreatePlan("My Plan", "")
	taskID, err := db.CreateTask(planID, "My Task", "Some notes", "School", "in_progress", "2026-04-01")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	task, err := db.GetTaskByID(taskID)
	if err != nil {
		t.Fatalf("GetTaskByID: %v", err)
	}
	if task == nil {
		t.Fatal("expected task, got nil")
	}
	if task.Title != "My Task" {
		t.Errorf("title = %q, want %q", task.Title, "My Task")
	}
	if task.Status != "in_progress" {
		t.Errorf("status = %q, want %q", task.Status, "in_progress")
	}
	if task.Category != "School" {
		t.Errorf("category = %q, want %q", task.Category, "School")
	}
	if task.DueDate != "2026-04-01" {
		t.Errorf("due_date = %q, want %q", task.DueDate, "2026-04-01")
	}
}

func TestGetTaskByID_NotFound(t *testing.T) {
	db := newTestDB(t)

	task, err := db.GetTaskByID(9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task != nil {
		t.Errorf("expected nil task for missing ID, got %+v", task)
	}
}

func TestGetAllTasks_NoFilter(t *testing.T) {
	db := newTestDB(t)

	planID, _ := db.CreatePlan("Plan", "")
	db.CreateTask(planID, "Task 1", "", "School", "not_started", "")
	db.CreateTask(planID, "Task 2", "", "Work", "completed", "")

	tasks, err := db.GetAllTasks(TaskFilters{})
	if err != nil {
		t.Fatalf("GetAllTasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2", len(tasks))
	}
}

func TestGetAllTasks_FilterByStatus(t *testing.T) {
	db := newTestDB(t)

	planID, _ := db.CreatePlan("Plan", "")
	db.CreateTask(planID, "Task 1", "", "", "not_started", "")
	db.CreateTask(planID, "Task 2", "", "", "completed", "")
	db.CreateTask(planID, "Task 3", "", "", "completed", "")

	tasks, err := db.GetAllTasks(TaskFilters{Status: "completed"})
	if err != nil {
		t.Fatalf("GetAllTasks with status filter: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("got %d tasks with status=completed, want 2", len(tasks))
	}
}

func TestGetAllTasks_FilterByCategory(t *testing.T) {
	db := newTestDB(t)

	planID, _ := db.CreatePlan("Plan", "")
	db.CreateTask(planID, "Task 1", "", "School", "not_started", "")
	db.CreateTask(planID, "Task 2", "", "Work", "not_started", "")
	db.CreateTask(planID, "Task 3", "", "School", "not_started", "")

	tasks, err := db.GetAllTasks(TaskFilters{Category: "School"})
	if err != nil {
		t.Fatalf("GetAllTasks with category filter: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("got %d tasks with category=School, want 2", len(tasks))
	}
}

func TestGetTasksByPlanID(t *testing.T) {
	db := newTestDB(t)

	planID1, _ := db.CreatePlan("Plan 1", "")
	planID2, _ := db.CreatePlan("Plan 2", "")
	db.CreateTask(planID1, "Task A", "", "", "not_started", "")
	db.CreateTask(planID1, "Task B", "", "", "not_started", "")
	db.CreateTask(planID2, "Task C", "", "", "not_started", "")

	tasks, err := db.GetTasksByPlanID(planID1)
	if err != nil {
		t.Fatalf("GetTasksByPlanID: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("got %d tasks for plan 1, want 2", len(tasks))
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	db := newTestDB(t)

	planID, _ := db.CreatePlan("Plan", "")
	taskID, _ := db.CreateTask(planID, "Task", "", "", "not_started", "")

	if err := db.UpdateTaskStatus(taskID, "completed"); err != nil {
		t.Fatalf("UpdateTaskStatus: %v", err)
	}

	task, _ := db.GetTaskByID(taskID)
	if task.Status != "completed" {
		t.Errorf("status = %q, want %q", task.Status, "completed")
	}
}

func TestDeleteTask(t *testing.T) {
	db := newTestDB(t)

	planID, _ := db.CreatePlan("Plan", "")
	taskID, _ := db.CreateTask(planID, "Task to delete", "", "", "not_started", "")

	if err := db.DeleteTask(taskID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	task, _ := db.GetTaskByID(taskID)
	if task != nil {
		t.Error("expected task to be deleted, but it still exists")
	}
}

func TestGetCategories(t *testing.T) {
	db := newTestDB(t)

	planID, _ := db.CreatePlan("Plan", "")
	db.CreateTask(planID, "Task 1", "", "School", "not_started", "")
	db.CreateTask(planID, "Task 2", "", "Work", "not_started", "")
	db.CreateTask(planID, "Task 3", "", "School", "not_started", "")

	cats, err := db.GetCategories()
	if err != nil {
		t.Fatalf("GetCategories: %v", err)
	}
	if len(cats) != 2 {
		t.Errorf("got %d categories, want 2", len(cats))
	}
}

func TestGetTaskStats(t *testing.T) {
	db := newTestDB(t)

	planID, _ := db.CreatePlan("Plan", "")
	db.CreateTask(planID, "T1", "", "", "not_started", "")
	db.CreateTask(planID, "T2", "", "", "in_progress", "")
	db.CreateTask(planID, "T3", "", "", "completed", "")

	stats, err := db.GetTaskStats()
	if err != nil {
		t.Fatalf("GetTaskStats: %v", err)
	}
	if stats.Total != 3 {
		t.Errorf("Total = %d, want 3", stats.Total)
	}
	if stats.Completed != 1 {
		t.Errorf("Completed = %d, want 1", stats.Completed)
	}
	if stats.InProgress != 1 {
		t.Errorf("InProgress = %d, want 1", stats.InProgress)
	}
	if stats.NotStarted != 1 {
		t.Errorf("NotStarted = %d, want 1", stats.NotStarted)
	}
}

func TestSeedTestData(t *testing.T) {
	db := newTestDB(t)
	db.SeedTestData()

	plans, err := db.GetAllPlans()
	if err != nil {
		t.Fatalf("GetAllPlans after seed: %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("got %d plans after seed, want 2", len(plans))
	}

	tasks, err := db.GetAllTasks(TaskFilters{})
	if err != nil {
		t.Fatalf("GetAllTasks after seed: %v", err)
	}
	if len(tasks) != 5 {
		t.Errorf("got %d tasks after seed, want 5", len(tasks))
	}
}

func TestClearDatabase(t *testing.T) {
	db := newTestDB(t)
	db.SeedTestData()
	db.ClearDatabase()

	plans, _ := db.GetAllPlans()
	if len(plans) != 0 {
		t.Errorf("expected 0 plans after clear, got %d", len(plans))
	}
	tasks, _ := db.GetAllTasks(TaskFilters{})
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks after clear, got %d", len(tasks))
	}
}
