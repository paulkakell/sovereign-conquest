package game

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"

	"sovereignconquest/internal/util"
)

const (
	corpNameMinLen = 3
	corpNameMaxLen = 32
	corpSayMaxLen  = 200
)

var corpNameRe = regexp.MustCompile(`^[A-Za-z0-9 _\-]+$`)

func executeCorpCommand(ctx context.Context, tx pgx.Tx, p *Player, cmd CommandRequest) (phase2Result, error) {
	action := cmd.Action
	if action == "" {
		action = "INFO"
	}

	switch action {
	case "INFO":
		return corpInfo(ctx, tx, *p)
	case "CREATE":
		return corpCreate(ctx, tx, p, cmd.Name)
	case "JOIN":
		return corpJoin(ctx, tx, p, cmd.Name)
	case "LEAVE":
		return corpLeave(ctx, tx, p)
	case "SAY":
		return corpSay(ctx, tx, *p, cmd.Text)
	case "DEPOSIT":
		return corpDeposit(ctx, tx, p, cmd.Quantity)
	case "WITHDRAW":
		return corpWithdraw(ctx, tx, p, cmd.Quantity)
	default:
		return phase2Result{OK: false, Message: "Unknown CORP subcommand.", ErrorCode: "UNKNOWN_SUBCOMMAND"}, nil
	}
}

func corpInfo(ctx context.Context, tx pgx.Tx, p Player) (phase2Result, error) {
	if p.CorpID == "" {
		msg := "You are not in a corporation. Use CORP CREATE {name} or CORP JOIN {name}."
		return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "SYSTEM", msg: msg}}}, nil
	}

	var name string
	var credits int64
	err := tx.QueryRow(ctx, "SELECT name, credits FROM corporations WHERE id=$1", p.CorpID).Scan(&name, &credits)
	if err != nil {
		return phase2Result{}, err
	}

	var members int
	_ = tx.QueryRow(ctx, "SELECT COUNT(1) FROM corp_members WHERE corp_id=$1", p.CorpID).Scan(&members)

	var planets int
	_ = tx.QueryRow(ctx, "SELECT COUNT(1) FROM planets WHERE owner_corp_id=$1", p.CorpID).Scan(&planets)

	msg := strings.Join([]string{
		fmt.Sprintf("Corporation: %s", name),
		fmt.Sprintf("Role: %s", p.CorpRole),
		fmt.Sprintf("Members: %d", members),
		fmt.Sprintf("Bank credits: %d", credits),
		fmt.Sprintf("Planets controlled: %d", planets),
	}, "\n")

	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "SYSTEM", msg: msg}}}, nil
}

func corpCreate(ctx context.Context, tx pgx.Tx, p *Player, name string) (phase2Result, error) {
	if p.CorpID != "" {
		return phase2Result{OK: false, Message: "You are already in a corporation.", ErrorCode: "ALREADY_IN_CORP"}, nil
	}

	name, err := validateCorpName(name)
	if err != nil {
		return phase2Result{OK: false, Message: err.Error(), ErrorCode: "INVALID_NAME"}, nil
	}

	corpID, err := util.NewID()
	if err != nil {
		return phase2Result{}, err
	}

	_, err = tx.Exec(ctx, "INSERT INTO corporations(id, name, credits) VALUES ($1,$2,0)", corpID, name)
	if err != nil {
		return phase2Result{OK: false, Message: "Corporation name unavailable.", ErrorCode: "NAME_UNAVAILABLE"}, nil
	}

	_, err = tx.Exec(ctx, "INSERT INTO corp_members(corp_id, player_id, role) VALUES ($1,$2,'LEADER')", corpID, p.ID)
	if err != nil {
		return phase2Result{}, err
	}

	p.CorpID = corpID
	p.CorpName = name
	p.CorpRole = "LEADER"
	p.CorpCredits = 0

	msg := fmt.Sprintf("Created corporation '%s'.", name)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func corpJoin(ctx context.Context, tx pgx.Tx, p *Player, name string) (phase2Result, error) {
	if p.CorpID != "" {
		return phase2Result{OK: false, Message: "You are already in a corporation.", ErrorCode: "ALREADY_IN_CORP"}, nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return phase2Result{OK: false, Message: "Usage: CORP JOIN {name}", ErrorCode: "INVALID_ARGS"}, nil
	}

	var corpID string
	var corpName string
	var corpCredits int64
	err := tx.QueryRow(ctx, "SELECT id, name, credits FROM corporations WHERE lower(name)=lower($1)", name).Scan(&corpID, &corpName, &corpCredits)
	if err == pgx.ErrNoRows {
		return phase2Result{OK: false, Message: "Corporation not found.", ErrorCode: "NOT_FOUND"}, nil
	}
	if err != nil {
		return phase2Result{}, err
	}

	_, err = tx.Exec(ctx, "INSERT INTO corp_members(corp_id, player_id, role) VALUES ($1,$2,'MEMBER')", corpID, p.ID)
	if err != nil {
		return phase2Result{OK: false, Message: "Unable to join corporation.", ErrorCode: "JOIN_FAILED"}, nil
	}

	p.CorpID = corpID
	p.CorpName = corpName
	p.CorpRole = "MEMBER"
	p.CorpCredits = corpCredits

	msg := fmt.Sprintf("Joined corporation '%s'.", corpName)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func corpLeave(ctx context.Context, tx pgx.Tx, p *Player) (phase2Result, error) {
	if p.CorpID == "" {
		return phase2Result{OK: false, Message: "You are not in a corporation.", ErrorCode: "NOT_IN_CORP"}, nil
	}
	corpID := p.CorpID
	corpName := p.CorpName

	var members int
	if err := tx.QueryRow(ctx, "SELECT COUNT(1) FROM corp_members WHERE corp_id=$1", corpID).Scan(&members); err != nil {
		return phase2Result{}, err
	}

	if strings.ToUpper(p.CorpRole) == "LEADER" && members > 1 {
		var newLeader string
		err := tx.QueryRow(ctx, `
			SELECT player_id
			FROM corp_members
			WHERE corp_id=$1 AND player_id <> $2
			ORDER BY joined_at ASC
			LIMIT 1
		`, corpID, p.ID).Scan(&newLeader)
		if err != nil {
			return phase2Result{}, err
		}
		_, _ = tx.Exec(ctx, "UPDATE corp_members SET role='LEADER' WHERE corp_id=$1 AND player_id=$2", corpID, newLeader)
	}

	_, err := tx.Exec(ctx, "DELETE FROM corp_members WHERE player_id=$1", p.ID)
	if err != nil {
		return phase2Result{}, err
	}

	if members <= 1 {
		// No members remain; remove corporation.
		_, _ = tx.Exec(ctx, "DELETE FROM corporations WHERE id=$1", corpID)
	}

	p.CorpID = ""
	p.CorpName = ""
	p.CorpRole = ""
	p.CorpCredits = 0

	msg := fmt.Sprintf("Left corporation '%s'.", corpName)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func corpSay(ctx context.Context, tx pgx.Tx, p Player, text string) (phase2Result, error) {
	if p.CorpID == "" {
		return phase2Result{OK: false, Message: "You are not in a corporation.", ErrorCode: "NOT_IN_CORP"}, nil
	}
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.TrimSpace(text)
	if text == "" {
		return phase2Result{OK: false, Message: "Message cannot be empty.", ErrorCode: "INVALID_ARGS"}, nil
	}
	if len(text) > corpSayMaxLen {
		return phase2Result{OK: false, Message: fmt.Sprintf("Message too long (max %d).", corpSayMaxLen), ErrorCode: "INVALID_ARGS"}, nil
	}

	_, err := tx.Exec(ctx, "INSERT INTO corp_messages(corp_id, player_id, message) VALUES ($1,$2,$3)", p.CorpID, p.ID, text)
	if err != nil {
		return phase2Result{}, err
	}

	payload := fmt.Sprintf("[%s] %s: %s", p.CorpName, p.Username, text)

	memberRows, err := tx.Query(ctx, "SELECT player_id FROM corp_members WHERE corp_id=$1", p.CorpID)
	if err != nil {
		return phase2Result{}, err
	}
	defer memberRows.Close()
	for memberRows.Next() {
		var pid string
		if err := memberRows.Scan(&pid); err != nil {
			return phase2Result{}, err
		}
		_, _ = tx.Exec(ctx, "INSERT INTO logs(player_id, kind, message) VALUES ($1,'CORP',$2)", pid, payload)
	}
	if err := memberRows.Err(); err != nil {
		return phase2Result{}, err
	}

	msg := "Corp message sent."
	return phase2Result{OK: true, Message: msg}, nil
}

func corpDeposit(ctx context.Context, tx pgx.Tx, p *Player, amount int) (phase2Result, error) {
	if p.CorpID == "" {
		return phase2Result{OK: false, Message: "You are not in a corporation.", ErrorCode: "NOT_IN_CORP"}, nil
	}
	if amount < 1 {
		return phase2Result{OK: false, Message: "Deposit amount must be at least 1.", ErrorCode: "INVALID_ARGS"}, nil
	}
	amt := int64(amount)
	if p.Credits < amt {
		return phase2Result{OK: false, Message: "Not enough credits.", ErrorCode: "INSUFFICIENT_CREDITS"}, nil
	}

	var newCredits int64
	err := tx.QueryRow(ctx, "UPDATE corporations SET credits = credits + $2 WHERE id=$1 RETURNING credits", p.CorpID, amt).Scan(&newCredits)
	if err != nil {
		return phase2Result{}, err
	}

	p.Credits -= amt
	p.CorpCredits = newCredits

	msg := fmt.Sprintf("Deposited %d credits to corp bank. New bank balance: %d.", amt, newCredits)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func corpWithdraw(ctx context.Context, tx pgx.Tx, p *Player, amount int) (phase2Result, error) {
	if p.CorpID == "" {
		return phase2Result{OK: false, Message: "You are not in a corporation.", ErrorCode: "NOT_IN_CORP"}, nil
	}
	role := strings.ToUpper(strings.TrimSpace(p.CorpRole))
	if role != "LEADER" && role != "OFFICER" {
		return phase2Result{OK: false, Message: "Only corp leadership can withdraw.", ErrorCode: "FORBIDDEN"}, nil
	}
	if amount < 1 {
		return phase2Result{OK: false, Message: "Withdraw amount must be at least 1.", ErrorCode: "INVALID_ARGS"}, nil
	}
	amt := int64(amount)

	var newCredits int64
	err := tx.QueryRow(ctx, "UPDATE corporations SET credits = credits - $2 WHERE id=$1 AND credits >= $2 RETURNING credits", p.CorpID, amt).Scan(&newCredits)
	if err == pgx.ErrNoRows {
		return phase2Result{OK: false, Message: "Corp bank does not have enough credits.", ErrorCode: "INSUFFICIENT_FUNDS"}, nil
	}
	if err != nil {
		return phase2Result{}, err
	}

	p.Credits += amt
	p.CorpCredits = newCredits

	msg := fmt.Sprintf("Withdrew %d credits from corp bank. New bank balance: %d.", amt, newCredits)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func validateCorpName(name string) (string, error) {
	name = sanitizeCorpName(name)
	if len(name) < corpNameMinLen {
		return "", fmt.Errorf("Corporation name too short (min %d).", corpNameMinLen)
	}
	if len(name) > corpNameMaxLen {
		return "", fmt.Errorf("Corporation name too long (max %d).", corpNameMaxLen)
	}
	if !corpNameRe.MatchString(name) {
		return "", fmt.Errorf("Corporation name contains invalid characters.")
	}
	return name, nil
}

func sanitizeCorpName(name string) string {
	name = strings.ReplaceAll(name, "\n", " ")
	name = strings.ReplaceAll(name, "\r", " ")
	name = strings.TrimSpace(name)
	fields := strings.Fields(name)
	return strings.Join(fields, " ")
}
