package game

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ActiveEvent struct {
	Kind         string
	SectorID     int
	Commodity    string
	PricePercent int
	Severity     int
	Title        string
	Description  string
	EndsAt       time.Time
}

func (e ActiveEvent) ToView() EventView {
	return EventView{
		Kind:         e.Kind,
		SectorID:     e.SectorID,
		Commodity:    e.Commodity,
		PricePercent: e.PricePercent,
		Severity:     e.Severity,
		Title:        e.Title,
		Description:  e.Description,
		EndsAt:       e.EndsAt,
	}
}

func LoadActiveEvent(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, sectorID int) (ActiveEvent, bool, error) {
	var e ActiveEvent
	err := q.QueryRow(ctx, `
		SELECT kind, sector_id, commodity, price_percent, severity, title, description, ends_at
		FROM events
		WHERE active=true AND ends_at > now() AND sector_id=$1
		LIMIT 1
	`, sectorID).Scan(&e.Kind, &e.SectorID, &e.Commodity, &e.PricePercent, &e.Severity, &e.Title, &e.Description, &e.EndsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ActiveEvent{}, false, nil
	}
	if err != nil {
		return ActiveEvent{}, false, err
	}
	return e, true, nil
}

func pricePercentForCommodity(e ActiveEvent, commodity string) int {
	if e.Kind != "ANOMALY" && e.Kind != "LIMITED" {
		return 100
	}
	p := e.PricePercent
	if p < 10 || p > 300 {
		return 100
	}
	c := strings.ToUpper(strings.TrimSpace(e.Commodity))
	if c == "" {
		c = "ALL"
	}
	if c == "ALL" || c == strings.ToUpper(commodity) {
		return p
	}
	return 100
}

func applySectorEventOnEntry(ctx context.Context, tx pgx.Tx, p *Player) (respMsg string, logKind string, logMsg string, err error) {
	e, ok, err := LoadActiveEvent(ctx, tx, p.SectorID)
	if err != nil || !ok {
		return "", "", "", err
	}

	remaining := time.Until(e.EndsAt)
	if remaining < 0 {
		remaining = 0
	}

	switch e.Kind {
	case "INVASION":
		// Credits penalty on entry (capped at available credits).
		penalty := int64(e.Severity) * 200
		if penalty < 0 {
			penalty = 0
		}
		if penalty > p.Credits {
			penalty = p.Credits
		}
		p.Credits -= penalty
		respMsg = fmt.Sprintf("Invasion alert: %s. Raiders seize %d credits.", e.Title, penalty)
		logKind = "COMBAT"
		logMsg = respMsg
		return respMsg, logKind, logMsg, nil
	case "ANOMALY", "LIMITED":
		respMsg = fmt.Sprintf("Event active: %s (ends in %s).", e.Title, formatDurationShort(remaining))
		logKind = "SYSTEM"
		logMsg = respMsg
		return respMsg, logKind, logMsg, nil
	default:
		respMsg = fmt.Sprintf("Event active: %s (ends in %s).", e.Title, formatDurationShort(remaining))
		logKind = "SYSTEM"
		logMsg = respMsg
		return respMsg, logKind, logMsg, nil
	}
}

func StartEventTicker(ctx context.Context, pool *pgxpool.Pool, tickSeconds int) {
	if tickSeconds <= 0 {
		return
	}
	if tickSeconds < 10 {
		tickSeconds = 10
	}
	// Deliberately not deterministic; events are meant to feel "alive".
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	ticker := time.NewTicker(time.Duration(tickSeconds) * time.Second)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Expire old events.
				_, _ = pool.Exec(ctx, `UPDATE events SET active=false WHERE active=true AND ends_at <= now()`)

				var activeCount int
				if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM events WHERE active=true`).Scan(&activeCount); err != nil {
					continue
				}
				if activeCount >= 5 {
					continue
				}

				// Probabilistic creation to avoid predictable spam.
				if rng.Float64() > 0.35 {
					continue
				}

				_ = createRandomEvent(ctx, pool, rng)
			}
		}
	}()
}

func createRandomEvent(ctx context.Context, pool *pgxpool.Pool, rng *rand.Rand) error {
	roll := rng.Intn(100)
	kind := "ANOMALY"
	if roll < 60 {
		kind = "ANOMALY"
	} else if roll < 85 {
		kind = "INVASION"
	} else {
		kind = "LIMITED"
	}

	var sectorID int
	if kind == "INVASION" {
		_ = pool.QueryRow(ctx, `SELECT id FROM sectors ORDER BY random() LIMIT 1`).Scan(&sectorID)
	} else {
		_ = pool.QueryRow(ctx, `SELECT sector_id FROM ports ORDER BY random() LIMIT 1`).Scan(&sectorID)
	}
	if sectorID < 1 {
		return nil
	}

	now := time.Now().UTC()

	commodity := "ALL"
	pricePercent := 100
	severity := 1
	title := ""
	desc := ""

	switch kind {
	case "ANOMALY":
		commodities := []string{"ORE", "ORGANICS", "EQUIPMENT"}
		commodity = commodities[rng.Intn(len(commodities))]
		if rng.Intn(2) == 0 {
			pricePercent = 80 + rng.Intn(11) // 80..90
			title = fmt.Sprintf("%s Glut", commodity)
			desc = fmt.Sprintf("A sudden surplus has depressed %s prices in this sector.", strings.ToLower(commodity))
		} else {
			pricePercent = 115 + rng.Intn(26) // 115..140
			title = fmt.Sprintf("%s Spike", commodity)
			desc = fmt.Sprintf("Demand is surging; %s prices are elevated in this sector.", strings.ToLower(commodity))
		}
		severity = 1 + rng.Intn(2)
	case "LIMITED":
		commodity = "ALL"
		pricePercent = 85 + rng.Intn(31) // 85..115
		severity = 1
		title = "Transient Market"
		desc = "A short-lived market distortion is affecting local prices."
	case "INVASION":
		commodity = "ALL"
		pricePercent = 100
		severity = 1 + rng.Intn(3) // 1..3
		title = "Raider Invasion"
		desc = "Hostile raiders are harassing traffic in this sector. Entry may cost credits."
	}

	durMin := 20 + rng.Intn(41) // 20..60
	if kind == "LIMITED" {
		durMin = 10 + rng.Intn(21) // 10..30
	} else if kind == "INVASION" {
		durMin = 15 + rng.Intn(46) // 15..60
	}
	endsAt := now.Add(time.Duration(durMin) * time.Minute)

	_, err := pool.Exec(ctx, `
		INSERT INTO events(kind, sector_id, commodity, price_percent, severity, title, description, started_at, ends_at, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,true)
	`, kind, sectorID, commodity, pricePercent, severity, title, desc, now, endsAt)
	if err != nil {
		// Ignore conflicts/errors; ticker will try again later.
		return nil
	}

	return nil
}
