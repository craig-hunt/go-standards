package postgres

const (
	migrationsPattern = "migrations/*.sql"
	errorFormat       = "%s: %w"
)

const (
	opOpen            = "open database pool"
	opPing            = "reach database"
	opPrepareMigrate  = "prepare migration tracking"
	opListMigrations  = "list migrations"
	opApplyMigration  = "apply migration"
	opSeed            = "seed demo data"
	opListTasks       = "list tasks"
	opCreateTask      = "create task"
	opSetCompleted    = "set task completion"
	opDeleteTask      = "delete task"
	opDeleteCompleted = "delete completed tasks"
	opSaveSignup      = "save signup"
	opListItems       = "list inventory items"
)

const (
	sqlCreateMigrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
    name text PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
)`
	sqlMigrationApplied = `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)`
	sqlRecordMigration  = `INSERT INTO schema_migrations (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`
)

const (
	sqlListTasks       = `SELECT id, title, completed FROM tasks ORDER BY id`
	sqlCreateTask      = `INSERT INTO tasks (title) VALUES ($1) RETURNING id, title, completed`
	sqlSetCompleted    = `UPDATE tasks SET completed = $2 WHERE id = $1 RETURNING id, title, completed`
	sqlDeleteTask      = `DELETE FROM tasks WHERE id = $1`
	sqlDeleteCompleted = `DELETE FROM tasks WHERE completed`
	sqlSeedTasks       = `INSERT INTO tasks (title)
SELECT seed.title
FROM unnest($1::text[]) WITH ORDINALITY AS seed (title, position)
WHERE NOT EXISTS (SELECT 1 FROM tasks)
ORDER BY seed.position`
)

const (
	sqlSaveSignup = `INSERT INTO signups (full_name, email, plan, seats, notes)
VALUES ($1, $2, $3, $4, $5)
RETURNING id`
)

const (
	sqlListItems  = `SELECT name, quantity, status FROM inventory_items ORDER BY name`
	sqlUpsertItem = `INSERT INTO inventory_items (name, quantity, status)
VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET quantity = EXCLUDED.quantity, status = EXCLUDED.status`
)
