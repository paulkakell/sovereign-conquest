package game

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommandError struct {
	Msg string
}

func (e CommandError) Error() string { return e.Msg }

func RegenTurns(p *Player, regenSeconds int, now time.Time) int {
	if regenSeconds < 10 {
		regenSeconds = 10
	}
	if p.LastTurnRegen.IsZero() {
		p.LastTurnRegen = now
		return 0
	}
	delta := now.Sub(p.LastTurnRegen)
	if delta <= 0 {
		return 0
	}
	added := int(delta.Seconds()) / regenSeconds
	if added <= 0 {
		return 0
	}
	p.Turns += added
	if p.Turns > p.TurnsMax {
		p.Turns = p.TurnsMax
	}
	p.LastTurnRegen = p.LastTurnRegen.Add(time.Duration(added*regenSeconds) * time.Second)
	return added
}

type logToInsert struct {
	kind string
	msg  string
}

func ExecuteCommand(ctx context.Context, pool *pgxpool.Pool, playerID string, cmd CommandRequest, regenSeconds int) (CommandResponse, error) {
	cmd.Type = strings.ToUpper(strings.TrimSpace(cmd.Type))
	cmd.Action = strings.ToUpper(strings.TrimSpace(cmd.Action))
	cmd.Commodity = strings.ToUpper(strings.TrimSpace(cmd.Commodity))
	cmd.Name = strings.TrimSpace(cmd.Name)
	cmd.Text = strings.TrimSpace(cmd.Text)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return CommandResponse{OK: false, Error: "db error"}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	p, err := LoadPlayerForUpdate(ctx, tx, playerID)
	if err != nil {
		return CommandResponse{OK: false, Error: "player not found"}, err
	}

	RegenTurns(&p, regenSeconds, time.Now().UTC())

	// Force a password change before any gameplay actions when required.
	// This is primarily used for the seeded initial admin account.
	if p.MustChangePass {
		return failWithState(ctx, pool, tx, p, "Password change required. Use the Change Password form.", "PASSWORD_CHANGE_REQUIRED")
	}

	cost := effectiveCommandCost(p, cmd)
	if cost > 0 && p.Turns < cost {
		// persist regen changes
		_ = SavePlayer(ctx, tx, p)
		sector, _ := LoadSectorView(ctx, tx, p.SectorID)
		_ = tx.Commit(ctx)
		logs, _ := LoadRecentLogs(ctx, pool, p.ID, 20)
		return CommandResponse{
			OK:      false,
			Message: "Not enough turns.",
			Error:   "NOT_ENOUGH_TURNS",
			State:   p.ToState(),
			Sector:  sector,
			Logs:    logs,
		}, CommandError{Msg: "not enough turns"}
	}

	success := false
	message := ""
	errCode := ""
	logsToInsert := make([]logToInsert, 0, 2)

	switch cmd.Type {
	case "SCAN":
		p.Turns -= cost
		success = true
		message = fmt.Sprintf("Scan complete for sector %d.", p.SectorID)
		logsToInsert = append(logsToInsert, logToInsert{kind: "ACTION", msg: message})
		_ = MarkDiscovered(ctx, tx, p.ID, p.SectorID)
		if err := CaptureScanIntel(ctx, tx, p.ID, p.SectorID); err != nil {
			return CommandResponse{OK: false, Error: "db error"}, err
		}

	case "MOVE":
		if cmd.To < 1 {
			return failWithState(ctx, pool, tx, p, "Invalid destination sector.", "INVALID_MOVE")
		}
		if !p.IsAdmin {
			var ok int
			err := tx.QueryRow(ctx, "SELECT 1 FROM warps WHERE from_sector=$1 AND to_sector=$2", p.SectorID, cmd.To).Scan(&ok)
			if errors.Is(err, pgx.ErrNoRows) {
				return failWithState(ctx, pool, tx, p, "No warp to that sector.", "INVALID_MOVE")
			}
			if err != nil {
				return CommandResponse{OK: false, Error: "db error"}, err
			}
		} else {
			// God mode: allow moving to any existing sector (teleport).
			var ok int
			err := tx.QueryRow(ctx, "SELECT 1 FROM sectors WHERE id=$1", cmd.To).Scan(&ok)
			if errors.Is(err, pgx.ErrNoRows) {
				return failWithState(ctx, pool, tx, p, "Invalid destination sector.", "INVALID_MOVE")
			}
			if err != nil {
				return CommandResponse{OK: false, Error: "db error"}, err
			}
		}
		p.Turns -= cost
		p.SectorID = cmd.To
		success = true
		moveMsg := fmt.Sprintf("Moved to sector %d.", p.SectorID)
		message = moveMsg
		logsToInsert = append(logsToInsert, logToInsert{kind: "ACTION", msg: moveMsg})
		_ = MarkDiscovered(ctx, tx, p.ID, p.SectorID)

		evtMsg, evtKind, evtLog, evtErr := applySectorEventOnEntry(ctx, tx, &p)
		if evtErr != nil {
			return CommandResponse{OK: false, Error: "db error"}, evtErr
		}
		if evtMsg != "" {
			message = message + "\n" + evtMsg
			if evtLog != "" {
				logsToInsert = append(logsToInsert, logToInsert{kind: evtKind, msg: evtLog})
			}
		}

		strikeMsg, strikeLog, strikeErr := applyMineStrike(ctx, tx, &p)
		if strikeErr != nil {
			return CommandResponse{OK: false, Error: "db error"}, strikeErr
		}
		if strikeMsg != "" {
			message = message + "\n" + strikeMsg
			if strikeLog != "" {
				logsToInsert = append(logsToInsert, logToInsert{kind: "COMBAT", msg: strikeLog})
			}
		}

	case "MARKET":
		out, execErr := executeMarketCommand(ctx, tx, p, cmd)
		if execErr != nil {
			return CommandResponse{OK: false, Error: "db error"}, execErr
		}
		success = true
		message = out
		logsToInsert = append(logsToInsert, logToInsert{kind: "SYSTEM", msg: out})

	case "ROUTE":
		out, execErr := executeRouteCommand(ctx, tx, p, cmd)
		if execErr != nil {
			return CommandResponse{OK: false, Error: "db error"}, execErr
		}
		success = true
		message = out
		logsToInsert = append(logsToInsert, logToInsert{kind: "SYSTEM", msg: out})

	case "EVENTS":
		out, execErr := executeEventsCommand(ctx, tx, p)
		if execErr != nil {
			return CommandResponse{OK: false, Error: "db error"}, execErr
		}
		success = true
		message = out
		logsToInsert = append(logsToInsert, logToInsert{kind: "SYSTEM", msg: out})

	case "TRADE":
		tradeMsg, tradeOk, tradeErr := executeTrade(ctx, tx, &p, cmd)
		if tradeErr != nil {
			// Database/transaction error: rollback to avoid partially applied state.
			return CommandResponse{OK: false, Error: "db error"}, tradeErr
		}
		if !tradeOk {
			return failWithState(ctx, pool, tx, p, tradeMsg, "TRADE_ERROR")
		}
		p.Turns -= cost
		success = true
		message = tradeMsg
		logsToInsert = append(logsToInsert, logToInsert{kind: "ACTION", msg: tradeMsg})

	case "PLANET":
		out, execErr := executePlanetCommand(ctx, tx, &p, cmd)
		if execErr != nil {
			return CommandResponse{OK: false, Error: "db error"}, execErr
		}
		if !out.OK {
			return failWithState(ctx, pool, tx, p, out.Message, out.ErrorCode)
		}
		p.Turns -= cost
		success = true
		message = out.Message
		for _, l := range out.Logs {
			logsToInsert = append(logsToInsert, logToInsert{kind: l.kind, msg: l.msg})
		}

	case "CORP":
		out, execErr := executeCorpCommand(ctx, tx, &p, cmd)
		if execErr != nil {
			return CommandResponse{OK: false, Error: "db error"}, execErr
		}
		if !out.OK {
			return failWithState(ctx, pool, tx, p, out.Message, out.ErrorCode)
		}
		p.Turns -= cost
		success = true
		message = out.Message
		for _, l := range out.Logs {
			logsToInsert = append(logsToInsert, logToInsert{kind: l.kind, msg: l.msg})
		}

	case "MINE":
		out, execErr := executeMineCommand(ctx, tx, &p, cmd)
		if execErr != nil {
			return CommandResponse{OK: false, Error: "db error"}, execErr
		}
		if !out.OK {
			return failWithState(ctx, pool, tx, p, out.Message, out.ErrorCode)
		}
		p.Turns -= cost
		success = true
		message = out.Message
		for _, l := range out.Logs {
			logsToInsert = append(logsToInsert, logToInsert{kind: l.kind, msg: l.msg})
		}

	case "SHIPYARD":
		out, execErr := executeShipyardCommand(ctx, tx, &p, cmd)
		if execErr != nil {
			return CommandResponse{OK: false, Error: "db error"}, execErr
		}
		if !out.OK {
			return failWithState(ctx, pool, tx, p, out.Message, out.ErrorCode)
		}
		success = true
		message = out.Message
		for _, l := range out.Logs {
			logsToInsert = append(logsToInsert, logToInsert{kind: l.kind, msg: l.msg})
		}

	case "RANKINGS":
		out, execErr := executeRankingsCommand(ctx, tx, p)
		if execErr != nil {
			return CommandResponse{OK: false, Error: "db error"}, execErr
		}
		success = true
		message = out
		logsToInsert = append(logsToInsert, logToInsert{kind: "SYSTEM", msg: out})

	case "SEASON":
		out, execErr := executeSeasonCommand(ctx, tx, p)
		if execErr != nil {
			return CommandResponse{OK: false, Error: "db error"}, execErr
		}
		success = true
		message = out
		logsToInsert = append(logsToInsert, logToInsert{kind: "SYSTEM", msg: out})

	case "HELP":
		success = true
		message = helpText()
		logsToInsert = append(logsToInsert, logToInsert{kind: "SYSTEM", msg: message})

	default:
		return failWithState(ctx, pool, tx, p, "Unknown command.", "UNKNOWN_COMMAND")
	}

	// Award XP for successful actions.
	if success {
		xpGain := XPGainForCommand(cmd, cost)
		if xpGain > 0 {
			leveled, _, newLevel := AwardXP(&p, xpGain)
			if leveled {
				rankMsg := fmt.Sprintf("Rank up! Level %d (%s).", newLevel, RankNameForLevel(newLevel))
				logsToInsert = append(logsToInsert, logToInsert{kind: "SYSTEM", msg: rankMsg})
				message = message + "\n" + rankMsg
			}
		}
	}

	if err := SavePlayer(ctx, tx, p); err != nil {
		return CommandResponse{OK: false, Error: "db error"}, err
	}

	for _, l := range logsToInsert {
		if l.msg == "" {
			continue
		}
		_ = InsertLog(ctx, tx, p.ID, l.kind, l.msg)
	}

	sector, err := LoadSectorView(ctx, tx, p.SectorID)
	if err != nil {
		return CommandResponse{OK: false, Error: "db error"}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return CommandResponse{OK: false, Error: "db error"}, err
	}

	logs, _ := LoadRecentLogs(ctx, pool, p.ID, 20)

	return CommandResponse{
		OK:      success,
		Message: message,
		Error:   errCode,
		State:   p.ToState(),
		Sector:  sector,
		Logs:    logs,
	}, nil
}

func failWithState(ctx context.Context, pool *pgxpool.Pool, tx pgx.Tx, p Player, msg, code string) (CommandResponse, error) {
	// persist regen changes only
	_ = SavePlayer(ctx, tx, p)
	sector, _ := LoadSectorView(ctx, tx, p.SectorID)
	_ = tx.Commit(ctx)
	logs, _ := LoadRecentLogs(ctx, pool, p.ID, 20)

	return CommandResponse{
		OK:      false,
		Message: msg,
		Error:   code,
		State:   p.ToState(),
		Sector:  sector,
		Logs:    logs,
	}, CommandError{Msg: msg}
}

func helpText() string {
	return strings.Join([]string{
		"Core: SCAN | MOVE {to} | TRADE {BUY|SELL} {ORE|ORGANICS|EQUIPMENT} {qty}",
		"Phase2: PLANET INFO | PLANET COLONIZE [name] | PLANET LOAD {commodity} {qty} | PLANET UNLOAD {commodity} {qty} | PLANET UPGRADE CITADEL",
		"Phase2: CORP INFO | CORP CREATE {name} | CORP JOIN {name} | CORP LEAVE | CORP SAY {message} | CORP DEPOSIT {credits} | CORP WITHDRAW {credits}",
		"Phase2: MINE DEPLOY {qty} | MINE SWEEP",
		"Phase2: SHIPYARD | SHIPYARD BUY {SCOUT|TRADER|FREIGHTER|INTERCEPTOR} | SHIPYARD SELL | SHIPYARD UPGRADE {CARGO|TURNS}",
		"Phase2: RANKINGS | SEASON",
		"Phase3: MARKET [ORE|ORGANICS|EQUIPMENT] | ROUTE [ORE|ORGANICS|EQUIPMENT] | EVENTS",
	}, "\n")
}

func effectiveCommandCost(p Player, cmd CommandRequest) int {
	if p.IsAdmin {
		return 0
	}
	return commandCost(cmd)
}

func commandCost(cmd CommandRequest) int {
	switch cmd.Type {
	case "SCAN":
		return 1
	case "MOVE":
		return 1
	case "TRADE":
		return 1
	case "PLANET":
		switch cmd.Action {
		case "COLONIZE":
			return 5
		case "LOAD", "UNLOAD":
			return 1
		case "UPGRADE_CITADEL":
			return 2
		default:
			return 0
		}
	case "MINE":
		switch cmd.Action {
		case "DEPLOY", "SWEEP":
			return 1
		default:
			return 0
		}
	case "HELP", "CORP", "RANKINGS", "SEASON", "MARKET", "ROUTE", "EVENTS", "SHIPYARD":
		return 0
	default:
		return 0
	}
}
