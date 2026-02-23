package game

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MessageKindUser       = "USER"
	MessageKindBugReport  = "BUG_REPORT"
	MessageKindSpamReport = "SPAM_REPORT"
)

type DirectMessageAttachmentView struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type DirectMessageView struct {
	ID               int64                         `json:"id"`
	Kind             string                        `json:"kind"`
	Subject          string                        `json:"subject"`
	Body             string                        `json:"body"`
	CreatedAt        time.Time                     `json:"created_at"`
	ReadAt           *time.Time                    `json:"read_at,omitempty"`
	RelatedMessageID *int64                        `json:"related_message_id,omitempty"`
	From             string                        `json:"from"`
	To               string                        `json:"to"`
	Attachments      []DirectMessageAttachmentView `json:"attachments,omitempty"`
}

type DirectMessageAttachmentData struct {
	Filename    string
	ContentType string
	Data        []byte
}

func LookupPlayerIDByUsername(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, username string) (string, error) {
	var pid string
	err := q.QueryRow(ctx, `
		SELECT p.id
		FROM players p
		JOIN users u ON u.id = p.user_id
		WHERE u.username = $1
	`, username).Scan(&pid)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return pid, err
}

func LookupUsernameByPlayerID(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, playerID string) (string, error) {
	var uname string
	err := q.QueryRow(ctx, `
		SELECT u.username
		FROM players p
		JOIN users u ON u.id = p.user_id
		WHERE p.id = $1
	`, playerID).Scan(&uname)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return uname, err
}

func LookupAdminPlayerID(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, preferredAdminUsername string) (string, error) {
	// Prefer the configured seeded admin username when available.
	if preferredAdminUsername != "" {
		var pid string
		err := q.QueryRow(ctx, `
			SELECT p.id
			FROM players p
			JOIN users u ON u.id = p.user_id
			WHERE u.username = $1 AND u.is_admin = true
		`, preferredAdminUsername).Scan(&pid)
		if err == nil {
			return pid, nil
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}

	// Fallback: any admin.
	var pid string
	err := q.QueryRow(ctx, `
		SELECT p.id
		FROM players p
		JOIN users u ON u.id = p.user_id
		WHERE u.is_admin = true
		ORDER BY u.created_at ASC
		LIMIT 1
	`).Scan(&pid)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return pid, err
}

func InsertDirectMessage(ctx context.Context, tx pgx.Tx, fromPlayerID, toPlayerID, kind, subject, body string, relatedMessageID *int64) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO direct_messages(from_player_id, to_player_id, kind, subject, body, related_message_id)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id
	`, fromPlayerID, toPlayerID, kind, subject, body, relatedMessageID).Scan(&id)
	return id, err
}

func InsertDirectMessageAttachment(ctx context.Context, tx pgx.Tx, messageID int64, filename, contentType string, data []byte) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO direct_message_attachments(message_id, filename, content_type, size_bytes, data)
		VALUES ($1,$2,$3,$4,$5)
	`, messageID, filename, contentType, int64(len(data)), data)
	return err
}

func LoadInboxDirectMessages(ctx context.Context, pool *pgxpool.Pool, playerID string, limit int) ([]DirectMessageView, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	rows, err := pool.Query(ctx, `
		SELECT
			m.id,
			m.kind,
			m.subject,
			m.body,
			m.created_at,
			m.read_at,
			m.related_message_id,
			uf.username AS from_username,
			ut.username AS to_username
		FROM direct_messages m
		JOIN players pf ON pf.id = m.from_player_id
		JOIN users uf ON uf.id = pf.user_id
		JOIN players pt ON pt.id = m.to_player_id
		JOIN users ut ON ut.id = pt.user_id
		WHERE m.to_player_id = $1 AND m.deleted_by_to=false
		ORDER BY m.created_at DESC
		LIMIT $2
	`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgs := make([]DirectMessageView, 0, limit)
	for rows.Next() {
		var m DirectMessageView
		if err := rows.Scan(&m.ID, &m.Kind, &m.Subject, &m.Body, &m.CreatedAt, &m.ReadAt, &m.RelatedMessageID, &m.From, &m.To); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attachments (small N; simple per-message lookup).
	for i := range msgs {
		atts, err := loadDirectMessageAttachments(ctx, pool, msgs[i].ID)
		if err != nil {
			return nil, err
		}
		msgs[i].Attachments = atts
	}

	return msgs, nil
}

func LoadSentDirectMessages(ctx context.Context, pool *pgxpool.Pool, playerID string, limit int) ([]DirectMessageView, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	rows, err := pool.Query(ctx, `
		SELECT
			m.id,
			m.kind,
			m.subject,
			m.body,
			m.created_at,
			m.read_at,
			m.related_message_id,
			uf.username AS from_username,
			ut.username AS to_username
		FROM direct_messages m
		JOIN players pf ON pf.id = m.from_player_id
		JOIN users uf ON uf.id = pf.user_id
		JOIN players pt ON pt.id = m.to_player_id
		JOIN users ut ON ut.id = pt.user_id
		WHERE m.from_player_id = $1 AND m.deleted_by_from=false
		ORDER BY m.created_at DESC
		LIMIT $2
	`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgs := make([]DirectMessageView, 0, limit)
	for rows.Next() {
		var m DirectMessageView
		if err := rows.Scan(&m.ID, &m.Kind, &m.Subject, &m.Body, &m.CreatedAt, &m.ReadAt, &m.RelatedMessageID, &m.From, &m.To); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range msgs {
		atts, err := loadDirectMessageAttachments(ctx, pool, msgs[i].ID)
		if err != nil {
			return nil, err
		}
		msgs[i].Attachments = atts
	}

	return msgs, nil
}

func loadDirectMessageAttachments(ctx context.Context, pool *pgxpool.Pool, messageID int64) ([]DirectMessageAttachmentView, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, filename, content_type, size_bytes
		FROM direct_message_attachments
		WHERE message_id = $1
		ORDER BY id ASC
	`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DirectMessageAttachmentView, 0, 4)
	for rows.Next() {
		var a DirectMessageAttachmentView
		if err := rows.Scan(&a.ID, &a.Filename, &a.ContentType, &a.SizeBytes); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// LoadDirectMessageForReport loads a message (and its attachments' raw bytes) for a spam/abuse report.
// The reporter must be the recipient of the message.
func LoadDirectMessageForReport(ctx context.Context, tx pgx.Tx, reporterPlayerID string, messageID int64) (DirectMessageView, []DirectMessageAttachmentData, error) {
	var m DirectMessageView
	err := tx.QueryRow(ctx, `
		SELECT
			m.id,
			m.kind,
			m.subject,
			m.body,
			m.created_at,
			uf.username AS from_username,
			ut.username AS to_username
		FROM direct_messages m
		JOIN players pf ON pf.id = m.from_player_id
		JOIN users uf ON uf.id = pf.user_id
		JOIN players pt ON pt.id = m.to_player_id
		JOIN users ut ON ut.id = pt.user_id
		WHERE m.id = $1 AND m.to_player_id = $2
	`, messageID, reporterPlayerID).Scan(&m.ID, &m.Kind, &m.Subject, &m.Body, &m.CreatedAt, &m.From, &m.To)
	if errors.Is(err, pgx.ErrNoRows) {
		return DirectMessageView{}, nil, ErrNotFound
	}
	if err != nil {
		return DirectMessageView{}, nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT filename, content_type, data
		FROM direct_message_attachments
		WHERE message_id = $1
		ORDER BY id ASC
	`, messageID)
	if err != nil {
		return DirectMessageView{}, nil, err
	}
	defer rows.Close()

	attachments := make([]DirectMessageAttachmentData, 0, 4)
	for rows.Next() {
		var a DirectMessageAttachmentData
		if err := rows.Scan(&a.Filename, &a.ContentType, &a.Data); err != nil {
			return DirectMessageView{}, nil, err
		}
		attachments = append(attachments, a)
	}
	if err := rows.Err(); err != nil {
		return DirectMessageView{}, nil, err
	}
	return m, attachments, nil
}

func LoadDirectMessageAttachmentForPlayer(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, attachmentID int64, playerID string) (string, string, []byte, error) {
	var filename, contentType string
	var data []byte
	err := q.QueryRow(ctx, `
		SELECT a.filename, a.content_type, a.data
		FROM direct_message_attachments a
		JOIN direct_messages m ON m.id = a.message_id
		WHERE a.id = $1
		AND ((m.from_player_id = $2 AND m.deleted_by_from=false) OR (m.to_player_id = $2 AND m.deleted_by_to=false))
	`, attachmentID, playerID).Scan(&filename, &contentType, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil, ErrNotFound
	}
	return filename, contentType, data, err
}

func LoadDirectMessageAttachmentForAdmin(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, attachmentID int64) (string, string, []byte, error) {
	var filename, contentType string
	var data []byte
	err := q.QueryRow(ctx, `
		SELECT filename, content_type, data
		FROM direct_message_attachments
		WHERE id = $1
	`, attachmentID).Scan(&filename, &contentType, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil, ErrNotFound
	}
	return filename, contentType, data, err
}

func FormatSpamReportBody(reporterUsername string, reported DirectMessageView) string {
	return fmt.Sprintf(
		"Spam/abuse report submitted by %s.\n\nReported message:\n- id: %d\n- from: %s\n- to: %s\n- sent_at: %s\n- subject: %s\n\nBody:\n%s\n",
		reporterUsername,
		reported.ID,
		reported.From,
		reported.To,
		reported.CreatedAt.UTC().Format(time.RFC3339),
		reported.Subject,
		reported.Body,
	)
}

// CountUnreadDirectMessages returns the number of unread inbox messages for a player.
func CountUnreadDirectMessages(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, playerID string) (int, error) {
	var c int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM direct_messages
		WHERE to_player_id=$1 AND read_at IS NULL AND deleted_by_to=false
	`, playerID).Scan(&c)
	return c, err
}

// MarkDirectMessagesRead marks the provided message IDs as read for the given player.
// Only messages where the player is the recipient are affected.
func MarkDirectMessagesRead(ctx context.Context, pool *pgxpool.Pool, playerID string, messageIDs []int64) (int64, error) {
	if len(messageIDs) == 0 {
		return 0, nil
	}
	ct, err := pool.Exec(ctx, `
		UPDATE direct_messages
		SET read_at = COALESCE(read_at, now())
		WHERE to_player_id=$1 AND id = ANY($2) AND deleted_by_to=false
	`, playerID, messageIDs)
	return ct.RowsAffected(), err
}

// DeleteDirectMessage hides a message from the requesting player.
// This is a per-user soft delete (sender and recipient can delete independently).
func DeleteDirectMessage(ctx context.Context, pool *pgxpool.Pool, playerID string, messageID int64) error {
	ct, err := pool.Exec(ctx, `
		UPDATE direct_messages
		SET deleted_by_to=true
		WHERE id=$1 AND to_player_id=$2 AND deleted_by_to=false
	`, messageID, playerID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() > 0 {
		return nil
	}

	ct, err = pool.Exec(ctx, `
		UPDATE direct_messages
		SET deleted_by_from=true
		WHERE id=$1 AND from_player_id=$2 AND deleted_by_from=false
	`, messageID, playerID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() > 0 {
		return nil
	}
	return ErrNotFound
}
