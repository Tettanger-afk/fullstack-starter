const Database = require('better-sqlite3');

const createPlansTableSQL = `
  CREATE TABLE IF NOT EXISTS plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    archived INTEGER DEFAULT 0
  )`;

const createTasksTableSQL = `
  CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    notes TEXT,
    category TEXT,
    status TEXT NOT NULL DEFAULT 'not_started',
    due_date TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    archived INTEGER DEFAULT 0,
    FOREIGN KEY (plan_id) REFERENCES plans(id)
  )`;

function createDatabaseManager(dbPath) {
  const database = new Database(dbPath);
  console.log('Database manager created for:', dbPath);
  database.pragma('foreign_keys = ON');
  database.exec(createPlansTableSQL);
  database.exec(createTasksTableSQL);

  function ensureConnected() {
    if (!database.open) {
      throw new Error('Database connection is not open');
    }
  }

  return {
    dbHelpers: {
      clearDatabase: () => {
        if (process.env.NODE_ENV === 'test') {
          ensureConnected();
          database.prepare('DELETE FROM tasks').run();
          database.prepare('DELETE FROM plans').run();
        } else {
          console.warn('clearDatabase called outside of test environment. FIXME!');
        }
      },

      seedTestData: () => {
        if (process.env.NODE_ENV === 'test') {
          ensureConnected();
          const insertPlan = database.prepare('INSERT INTO plans (title, description) VALUES (?, ?)');
          const plan1 = insertPlan.run('CS 408 Project', 'Full-stack web application project');
          const plan2 = insertPlan.run('Study Schedule', 'Weekly study plan for finals');

          const insertTask = database.prepare(
            'INSERT INTO tasks (plan_id, title, notes, category, status, due_date) VALUES (?, ?, ?, ?, ?, ?)'
          );
          insertTask.run(plan1.lastInsertRowid, 'Set up database schema', 'Create tables for plans and tasks', 'School', 'completed', '2026-03-15');
          insertTask.run(plan1.lastInsertRowid, 'Implement CRUD routes', 'Backend API routes for create, read, update, delete', 'School', 'in_progress', '2026-03-22');
          insertTask.run(plan1.lastInsertRowid, 'Build frontend templates', 'EJS templates with Bootstrap styling', 'School', 'not_started', '2026-03-29');
          insertTask.run(plan2.lastInsertRowid, 'Review lecture notes', 'Go over chapters 5-8', 'School', 'not_started', '2026-03-28');
          insertTask.run(plan2.lastInsertRowid, 'Study group meeting', 'Meet with team at library', 'School', 'in_progress', '2026-03-26');
          console.log('Seeding test data into database');
        } else {
          console.warn('seedTestData called outside of test environment. FIXME!');
        }
      },

      seedDevData: () => {
        ensureConnected();
        const count = database.prepare('SELECT COUNT(*) AS c FROM plans').get().c;
        if (count > 0) return;

        const insertPlan = database.prepare('INSERT INTO plans (title, description) VALUES (?, ?)');
        const plan1 = insertPlan.run('CS 408 Project', 'Full-stack web application project for Spring 2026');
        const plan2 = insertPlan.run('Personal Goals', 'Things to accomplish this semester');

        const insertTask = database.prepare(
          'INSERT INTO tasks (plan_id, title, notes, category, status, due_date) VALUES (?, ?, ?, ?, ?, ?)'
        );
        insertTask.run(plan1.lastInsertRowid, 'Set up database schema', 'Create SQLite tables for plans and tasks', 'School', 'completed', '2026-03-15');
        insertTask.run(plan1.lastInsertRowid, 'Implement CRUD routes', 'Backend routes for all CRUD operations', 'School', 'in_progress', '2026-03-22');
        insertTask.run(plan1.lastInsertRowid, 'Build frontend templates', 'EJS templates with Bootstrap', 'School', 'not_started', '2026-03-29');
        insertTask.run(plan2.lastInsertRowid, 'Exercise 3x per week', 'Hit the gym consistently', 'Personal', 'in_progress', '2026-05-01');
        insertTask.run(plan2.lastInsertRowid, 'Read two books', 'For personal growth', 'Personal', 'not_started', '2026-05-15');
        console.log('Dev seed data inserted');
      },

      // Plans
      getAllPlans: () => {
        return database.prepare(`
          SELECT p.*, COUNT(t.id) as task_count
          FROM plans p
          LEFT JOIN tasks t ON t.plan_id = p.id AND t.archived = 0
          WHERE p.archived = 0
          GROUP BY p.id
          ORDER BY p.created_at DESC
        `).all();
      },

      getPlanById: (id) => {
        return database.prepare('SELECT * FROM plans WHERE id = ? AND archived = 0').get(id);
      },

      createPlan: (title, description) => {
        const info = database.prepare('INSERT INTO plans (title, description) VALUES (?, ?)').run(title, description || null);
        return info.lastInsertRowid;
      },

      deletePlan: (id) => {
        database.prepare('DELETE FROM tasks WHERE plan_id = ?').run(id);
        const info = database.prepare('DELETE FROM plans WHERE id = ?').run(id);
        return info.changes;
      },

      // Tasks
      getAllTasks: (filters = {}) => {
        let query = 'SELECT t.*, p.title as plan_title FROM tasks t LEFT JOIN plans p ON t.plan_id = p.id WHERE t.archived = 0';
        const params = [];

        if (filters.plan_id) {
          query += ' AND t.plan_id = ?';
          params.push(filters.plan_id);
        }
        if (filters.status) {
          query += ' AND t.status = ?';
          params.push(filters.status);
        }
        if (filters.category) {
          query += ' AND t.category = ?';
          params.push(filters.category);
        }

        query += ' ORDER BY CASE WHEN t.due_date IS NULL THEN 1 ELSE 0 END, t.due_date ASC, t.created_at DESC';
        return database.prepare(query).all(...params);
      },

      getTaskById: (id) => {
        return database.prepare(`
          SELECT t.*, p.title as plan_title
          FROM tasks t
          LEFT JOIN plans p ON t.plan_id = p.id
          WHERE t.id = ?
        `).get(id);
      },

      getTasksByPlanId: (planId) => {
        return database.prepare(`
          SELECT * FROM tasks WHERE plan_id = ? AND archived = 0
          ORDER BY CASE WHEN due_date IS NULL THEN 1 ELSE 0 END, due_date ASC
        `).all(planId);
      },

      createTask: (plan_id, title, notes, category, status, due_date) => {
        const info = database.prepare(`
          INSERT INTO tasks (plan_id, title, notes, category, status, due_date)
          VALUES (?, ?, ?, ?, ?, ?)
        `).run(plan_id, title, notes || null, category || null, status || 'not_started', due_date || null);
        return info.lastInsertRowid;
      },

      updateTaskStatus: (id, status) => {
        const info = database.prepare('UPDATE tasks SET status = ? WHERE id = ?').run(status, id);
        return info.changes;
      },

      deleteTask: (id) => {
        const info = database.prepare('DELETE FROM tasks WHERE id = ?').run(id);
        return info.changes;
      },

      getCategories: () => {
        return database.prepare(
          'SELECT DISTINCT category FROM tasks WHERE category IS NOT NULL AND archived = 0 ORDER BY category'
        ).all().map(r => r.category);
      },

      getTaskStats: () => {
        return {
          total: database.prepare('SELECT COUNT(*) AS c FROM tasks WHERE archived = 0').get().c,
          completed: database.prepare("SELECT COUNT(*) AS c FROM tasks WHERE status = 'completed' AND archived = 0").get().c,
          in_progress: database.prepare("SELECT COUNT(*) AS c FROM tasks WHERE status = 'in_progress' AND archived = 0").get().c,
          not_started: database.prepare("SELECT COUNT(*) AS c FROM tasks WHERE status = 'not_started' AND archived = 0").get().c,
          overdue: database.prepare("SELECT COUNT(*) AS c FROM tasks WHERE due_date < date('now') AND status != 'completed' AND archived = 0").get().c,
        };
      },

      getOverdueTasks: () => {
        return database.prepare(`
          SELECT t.*, p.title as plan_title
          FROM tasks t LEFT JOIN plans p ON t.plan_id = p.id
          WHERE t.due_date < date('now') AND t.status != 'completed' AND t.archived = 0
          ORDER BY t.due_date ASC
        `).all();
      },

      getUpcomingTasks: () => {
        return database.prepare(`
          SELECT t.*, p.title as plan_title
          FROM tasks t LEFT JOIN plans p ON t.plan_id = p.id
          WHERE t.due_date >= date('now') AND t.due_date <= date('now', '+7 days')
          AND t.status != 'completed' AND t.archived = 0
          ORDER BY t.due_date ASC
        `).all();
      },
    }
  };
}

module.exports = {
  createDatabaseManager,
};
