package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"video_solicitation_microservice/internal/core/domain/entity"
	"video_solicitation_microservice/internal/core/domain/port"
	"video_solicitation_microservice/internal/core/domain/value_object"
)

type videoRepositorySQLite struct {
	db *sql.DB
}

func NewVideoRepositorySQLite(db *sql.DB) port.VideoRepository {
	return &videoRepositorySQLite{db: db}
}

func (r *videoRepositorySQLite) Save(ctx context.Context, video *entity.Video) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO videos (id, user_id, user_name, user_email, file_name, duration_seconds, size_bytes, status, bucket_name, video_chunk_folder, image_folder, download_url, error_cause, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		video.ID,
		video.User.ID,
		video.User.Name,
		video.User.Email,
		video.Metadata.FileName,
		video.Metadata.DurationSeconds,
		video.Metadata.SizeBytes,
		string(video.Status),
		video.FileLocation.BucketName,
		video.FileLocation.VideoChunkFolder,
		video.FileLocation.ImageFolder,
		nullString(video.FileLocation.DownloadURL),
		nullString(video.ErrorCause),
		video.CreatedAt.Format(time.RFC3339),
		video.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert video: %w", err)
	}

	for _, chunk := range video.Chunks {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO chunks (video_id, part_number, start_time, end_time, frame_per_second, status, video_object_id)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			video.ID,
			chunk.PartNumber,
			chunk.StartTime,
			chunk.EndTime,
			chunk.FramePerSecond,
			string(chunk.Status),
			chunk.VideoObjectID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert chunk %d: %w", chunk.PartNumber, err)
		}
	}

	return tx.Commit()
}

func (r *videoRepositorySQLite) FindByID(ctx context.Context, id string) (*entity.Video, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, user_name, user_email, file_name, duration_seconds, size_bytes, status, bucket_name, video_chunk_folder, image_folder, download_url, error_cause, created_at, updated_at
		FROM videos WHERE id = ?`, id)

	video := &entity.Video{}
	var status string
	var downloadURL, errorCause sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(
		&video.ID,
		&video.User.ID,
		&video.User.Name,
		&video.User.Email,
		&video.Metadata.FileName,
		&video.Metadata.DurationSeconds,
		&video.Metadata.SizeBytes,
		&status,
		&video.FileLocation.BucketName,
		&video.FileLocation.VideoChunkFolder,
		&video.FileLocation.ImageFolder,
		&downloadURL,
		&errorCause,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan video: %w", err)
	}

	video.Status = value_object.VideoStatus(status)
	video.FileLocation.DownloadURL = downloadURL.String
	video.ErrorCause = errorCause.String

	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		video.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		video.UpdatedAt = t
	}

	// Load chunks
	rows, err := r.db.QueryContext(ctx, `
		SELECT part_number, start_time, end_time, frame_per_second, status, video_object_id
		FROM chunks WHERE video_id = ? ORDER BY part_number`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var chunk entity.Chunk
		var chunkStatus string
		if err := rows.Scan(
			&chunk.PartNumber,
			&chunk.StartTime,
			&chunk.EndTime,
			&chunk.FramePerSecond,
			&chunkStatus,
			&chunk.VideoObjectID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunk.Status = value_object.ChunkStatus(chunkStatus)
		video.Chunks = append(video.Chunks, chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating chunks: %w", err)
	}

	return video, nil
}

func (r *videoRepositorySQLite) Update(ctx context.Context, video *entity.Video) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE videos SET status = ?, download_url = ?, error_cause = ?, updated_at = ?
		WHERE id = ?`,
		string(video.Status),
		nullString(video.FileLocation.DownloadURL),
		nullString(video.ErrorCause),
		video.UpdatedAt.Format(time.RFC3339),
		video.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update video: %w", err)
	}

	for _, chunk := range video.Chunks {
		_, err = tx.ExecContext(ctx, `
			UPDATE chunks SET status = ? WHERE video_id = ? AND part_number = ?`,
			string(chunk.Status),
			video.ID,
			chunk.PartNumber,
		)
		if err != nil {
			return fmt.Errorf("failed to update chunk %d: %w", chunk.PartNumber, err)
		}
	}

	return tx.Commit()
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
