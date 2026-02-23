package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	mineSweepCapacity   = 10
	mineTriggerCap      = 5
	mineDamagePerMine   = int64(50)
	mineMaxDeployPerCmd = 1000
)

func executeMineCommand(ctx context.Context, tx pgx.Tx, p *Player, cmd CommandRequest) (phase2Result, error) {
	action := cmd.Action
	if action == "" {
		action = "INFO"
	}

	switch action {
	case "DEPLOY":
		return mineDeploy(ctx, tx, p, cmd.Quantity)
	case "SWEEP":
		return mineSweep(ctx, tx, p)
	case "INFO":
		msg := "MINE DEPLOY {qty} (uses equipment cargo) | MINE SWEEP"
		return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "SYSTEM", msg: msg}}}, nil
	default:
		return phase2Result{OK: false, Message: "Unknown MINE subcommand.", ErrorCode: "UNKNOWN_SUBCOMMAND"}, nil
	}
}

func mineDeploy(ctx context.Context, tx pgx.Tx, p *Player, qty int) (phase2Result, error) {
	if qty < 1 {
		return phase2Result{OK: false, Message: "Deploy quantity must be at least 1.", ErrorCode: "INVALID_QTY"}, nil
	}
	if qty > mineMaxDeployPerCmd {
		return phase2Result{OK: false, Message: fmt.Sprintf("Deploy quantity too large (max %d per command).", mineMaxDeployPerCmd), ErrorCode: "INVALID_QTY"}, nil
	}
	isProt, err := IsProtectorateSector(ctx, tx, p.SectorID)
	if err != nil {
		return phase2Result{}, err
	}
	if isProt {
		return phase2Result{OK: false, Message: "The Galactic Protectorate forbids mine deployment in Protectorate sectors.", ErrorCode: "PROTECTORATE_PEACE"}, nil
	}

	if p.CargoEquipment < qty {
		return phase2Result{OK: false, Message: "Not enough equipment cargo to deploy mines.", ErrorCode: "INSUFFICIENT_EQUIPMENT"}, nil
	}

	p.CargoEquipment -= qty

	var ownerCorp any = nil
	if p.CorpID != "" {
		ownerCorp = p.CorpID
	}

	var newQty int
	err := tx.QueryRow(ctx, `
		INSERT INTO mines(sector_id, owner_player_id, owner_corp_id, qty)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (sector_id, owner_player_id)
		DO UPDATE SET
			qty = mines.qty + EXCLUDED.qty,
			owner_corp_id = EXCLUDED.owner_corp_id
		RETURNING qty
	`, p.SectorID, p.ID, ownerCorp, qty).Scan(&newQty)
	if err != nil {
		return phase2Result{}, err
	}

	msg := fmt.Sprintf("Deployed %d mines in sector %d. Your minefield here is now %d.", qty, p.SectorID, newQty)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func mineSweep(ctx context.Context, tx pgx.Tx, p *Player) (phase2Result, error) {
	corpID := p.CorpID

	rows, err := tx.Query(ctx, `
		SELECT owner_player_id, qty
		FROM mines
		WHERE sector_id=$1
			AND owner_player_id <> $2
			AND (owner_corp_id IS NULL OR owner_corp_id <> $3 OR $3 = '')
		ORDER BY qty DESC
		FOR UPDATE
	`, p.SectorID, p.ID, corpID)
	if err != nil {
		return phase2Result{}, err
	}
	defer rows.Close()

	type mineRow struct {
		OwnerPlayerID string
		Qty           int
	}
	list := make([]mineRow, 0, 8)
	for rows.Next() {
		var mr mineRow
		if err := rows.Scan(&mr.OwnerPlayerID, &mr.Qty); err != nil {
			return phase2Result{}, err
		}
		if mr.Qty > 0 {
			list = append(list, mr)
		}
	}
	if err := rows.Err(); err != nil {
		return phase2Result{}, err
	}

	remaining := mineSweepCapacity
	removed := 0
	for _, mr := range list {
		if remaining <= 0 {
			break
		}
		take := mr.Qty
		if take > remaining {
			take = remaining
		}
		remaining -= take
		removed += take
		newQty := mr.Qty - take
		if newQty <= 0 {
			_, _ = tx.Exec(ctx, "DELETE FROM mines WHERE sector_id=$1 AND owner_player_id=$2", p.SectorID, mr.OwnerPlayerID)
		} else {
			_, _ = tx.Exec(ctx, "UPDATE mines SET qty=$3 WHERE sector_id=$1 AND owner_player_id=$2", p.SectorID, mr.OwnerPlayerID, newQty)
		}
	}

	if removed == 0 {
		msg := "No hostile mines detected."
		return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "SYSTEM", msg: msg}}}, nil
	}
	msg := fmt.Sprintf("Swept %d hostile mines in sector %d.", removed, p.SectorID)
	return phase2Result{OK: true, Message: msg, Logs: []logToInsert{{kind: "ACTION", msg: msg}}}, nil
}

func applyMineStrike(ctx context.Context, tx pgx.Tx, p *Player) (respMsg string, logMsg string, err error) {
	corpID := p.CorpID

	rows, err := tx.Query(ctx, `
		SELECT owner_player_id, qty
		FROM mines
		WHERE sector_id=$1
			AND owner_player_id <> $2
			AND (owner_corp_id IS NULL OR owner_corp_id <> $3 OR $3 = '')
		ORDER BY created_at ASC
		FOR UPDATE
	`, p.SectorID, p.ID, corpID)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()

	type mineRow struct {
		OwnerPlayerID string
		Qty           int
	}
	list := make([]mineRow, 0, 8)
	total := 0
	for rows.Next() {
		var mr mineRow
		if err := rows.Scan(&mr.OwnerPlayerID, &mr.Qty); err != nil {
			return "", "", err
		}
		if mr.Qty > 0 {
			list = append(list, mr)
			total += mr.Qty
		}
	}
	if err := rows.Err(); err != nil {
		return "", "", err
	}
	if total <= 0 {
		return "", "", nil
	}

	triggered := mineTriggerCount(total)
	remaining := triggered
	for _, mr := range list {
		if remaining <= 0 {
			break
		}
		take := mr.Qty
		if take > remaining {
			take = remaining
		}
		remaining -= take
		newQty := mr.Qty - take
		if newQty <= 0 {
			_, _ = tx.Exec(ctx, "DELETE FROM mines WHERE sector_id=$1 AND owner_player_id=$2", p.SectorID, mr.OwnerPlayerID)
		} else {
			_, _ = tx.Exec(ctx, "UPDATE mines SET qty=$3 WHERE sector_id=$1 AND owner_player_id=$2", p.SectorID, mr.OwnerPlayerID, newQty)
		}
	}

	damage := mineDamageCredits(triggered)
	if damage > p.Credits {
		damage = p.Credits
	}
	p.Credits -= damage

	respMsg = fmt.Sprintf("Mine strike! %d mines detonated. Repairs cost %d credits.", triggered, damage)
	logMsg = strings.TrimSpace(respMsg)
	return respMsg, logMsg, nil
}

func mineTriggerCount(totalHostile int) int {
	if totalHostile < 1 {
		return 0
	}
	if totalHostile > mineTriggerCap {
		return mineTriggerCap
	}
	return totalHostile
}

func mineDamageCredits(triggered int) int64 {
	if triggered < 1 {
		return 0
	}
	return int64(triggered) * mineDamagePerMine
}
