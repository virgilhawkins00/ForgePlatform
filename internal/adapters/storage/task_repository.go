package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// TaskRepository implements ports.TaskRepository using SQLite.
type TaskRepository struct {
	db *DB
}

// NewTaskRepository creates a new task repository.
func NewTaskRepository(db *DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create persists a new task.
func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	payloadJSON, err := json.Marshal(task.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	idBytes, _ := task.ID.MarshalBinary()

	query := `
		INSERT INTO tasks (id, type, payload, status, priority, max_retries, retry_count, 
		                   run_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.conn.ExecContext(ctx, query,
		idBytes,
		string(task.Type),
		payloadJSON,
		string(task.Status),
		task.Priority,
		task.MaxRetries,
		task.RetryCount,
		task.RunAt.UnixMilli(),
		task.CreatedAt.UnixMilli(),
		task.UpdatedAt.UnixMilli(),
	)

	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	return nil
}

// GetByID retrieves a task by its ID.
func (r *TaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	idBytes, _ := id.MarshalBinary()

	query := `
		SELECT id, type, payload, status, priority, max_retries, retry_count,
		       run_at, locked_until, created_at, updated_at, completed_at, error
		FROM tasks WHERE id = ?
	`

	row := r.db.conn.QueryRowContext(ctx, query, idBytes)
	return r.scanTask(row)
}

// Update updates an existing task.
func (r *TaskRepository) Update(ctx context.Context, task *domain.Task) error {
	payloadJSON, _ := json.Marshal(task.Payload)
	idBytes, _ := task.ID.MarshalBinary()

	var lockedUntil, completedAt *int64
	if task.LockedUntil != nil {
		v := task.LockedUntil.UnixMilli()
		lockedUntil = &v
	}
	if task.CompletedAt != nil {
		v := task.CompletedAt.UnixMilli()
		completedAt = &v
	}

	query := `
		UPDATE tasks SET
			type = ?, payload = ?, status = ?, priority = ?, max_retries = ?,
			retry_count = ?, run_at = ?, locked_until = ?, updated_at = ?,
			completed_at = ?, error = ?
		WHERE id = ?
	`

	_, err := r.db.conn.ExecContext(ctx, query,
		string(task.Type),
		payloadJSON,
		string(task.Status),
		task.Priority,
		task.MaxRetries,
		task.RetryCount,
		task.RunAt.UnixMilli(),
		lockedUntil,
		task.UpdatedAt.UnixMilli(),
		completedAt,
		task.Error,
		idBytes,
	)

	return err
}

// Delete removes a task.
func (r *TaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	idBytes, _ := id.MarshalBinary()
	_, err := r.db.conn.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", idBytes)
	return err
}

// List retrieves tasks with optional filtering.
func (r *TaskRepository) List(ctx context.Context, filter ports.TaskFilter) ([]*domain.Task, error) {
	query := "SELECT id, type, payload, status, priority, max_retries, retry_count, run_at, locked_until, created_at, updated_at, completed_at, error FROM tasks WHERE 1=1"
	var args []interface{}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, string(*filter.Status))
	}
	if filter.Type != nil {
		query += " AND type = ?"
		args = append(args, string(*filter.Type))
	}

	query += " ORDER BY priority DESC, run_at ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		task, err := r.scanTaskRows(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ClaimNext atomically claims the next pending task.
func (r *TaskRepository) ClaimNext(ctx context.Context, lockDuration time.Duration) (*domain.Task, error) {
	now := time.Now()
	lockedUntil := now.Add(lockDuration)

	// Use UPDATE...RETURNING for atomic claim (SQLite 3.35+)
	query := `
		UPDATE tasks SET status = 'RUNNING', locked_until = ?, updated_at = ?
		WHERE id = (
			SELECT id FROM tasks
			WHERE status = 'PENDING' AND run_at <= ?
			ORDER BY priority DESC, run_at ASC
			LIMIT 1
		)
		RETURNING id, type, payload, status, priority, max_retries, retry_count,
		          run_at, locked_until, created_at, updated_at, completed_at, error
	`

	row := r.db.conn.QueryRowContext(ctx, query,
		lockedUntil.UnixMilli(),
		now.UnixMilli(),
		now.UnixMilli(),
	)

	task, err := r.scanTask(row)
	if err == sql.ErrNoRows {
		return nil, nil // No tasks available
	}
	return task, err
}

// ReleaseExpired releases tasks with expired locks.
func (r *TaskRepository) ReleaseExpired(ctx context.Context) (int64, error) {
	now := time.Now()
	result, err := r.db.conn.ExecContext(ctx,
		"UPDATE tasks SET status = 'PENDING', locked_until = NULL WHERE status = 'RUNNING' AND locked_until < ?",
		now.UnixMilli(),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *TaskRepository) scanTask(row *sql.Row) (*domain.Task, error) {
	var task domain.Task
	var idBytes, payloadJSON []byte
	var taskType, status string
	var runAt, createdAt, updatedAt int64
	var lockedUntil, completedAt sql.NullInt64
	var errorStr sql.NullString

	err := row.Scan(&idBytes, &taskType, &payloadJSON, &status, &task.Priority,
		&task.MaxRetries, &task.RetryCount, &runAt, &lockedUntil, &createdAt,
		&updatedAt, &completedAt, &errorStr)
	if err != nil {
		return nil, err
	}

	task.ID = uuidFromBytes(idBytes)
	task.Type = domain.TaskType(taskType)
	task.Status = domain.TaskStatus(status)
	_ = json.Unmarshal(payloadJSON, &task.Payload)
	task.RunAt = time.UnixMilli(runAt)
	task.CreatedAt = time.UnixMilli(createdAt)
	task.UpdatedAt = time.UnixMilli(updatedAt)

	if lockedUntil.Valid {
		t := time.UnixMilli(lockedUntil.Int64)
		task.LockedUntil = &t
	}
	if completedAt.Valid {
		t := time.UnixMilli(completedAt.Int64)
		task.CompletedAt = &t
	}
	if errorStr.Valid {
		task.Error = errorStr.String
	}

	return &task, nil
}

func (r *TaskRepository) scanTaskRows(rows *sql.Rows) (*domain.Task, error) {
	var task domain.Task
	var idBytes, payloadJSON []byte
	var taskType, status string
	var runAt, createdAt, updatedAt int64
	var lockedUntil, completedAt sql.NullInt64
	var errorStr sql.NullString

	err := rows.Scan(&idBytes, &taskType, &payloadJSON, &status, &task.Priority,
		&task.MaxRetries, &task.RetryCount, &runAt, &lockedUntil, &createdAt,
		&updatedAt, &completedAt, &errorStr)
	if err != nil {
		return nil, err
	}

	task.ID = uuidFromBytes(idBytes)
	task.Type = domain.TaskType(taskType)
	task.Status = domain.TaskStatus(status)
	_ = json.Unmarshal(payloadJSON, &task.Payload)
	task.RunAt = time.UnixMilli(runAt)
	task.CreatedAt = time.UnixMilli(createdAt)
	task.UpdatedAt = time.UnixMilli(updatedAt)

	if lockedUntil.Valid {
		t := time.UnixMilli(lockedUntil.Int64)
		task.LockedUntil = &t
	}
	if completedAt.Valid {
		t := time.UnixMilli(completedAt.Int64)
		task.CompletedAt = &t
	}
	if errorStr.Valid {
		task.Error = errorStr.String
	}

	return &task, nil
}

var _ ports.TaskRepository = (*TaskRepository)(nil)

