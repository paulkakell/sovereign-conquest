package game

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type sectorAdminInfo struct {
	ID             int
	IsProtectorate bool
	HasPort        bool
	PlanetName     string
	PlanetOwner    string // empty => unowned, "" with PlanetName empty => no planet
	PlayerNames    []string
}

// GenerateAdminAnsiMap returns a simple terminal-friendly (ASCII/ANSI-style) universe map.
// This is intentionally conservative: it is an approximate grid ordered by sector ID.
func GenerateAdminAnsiMap(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	// Sectors base.
	rows, err := pool.Query(ctx, `SELECT id, is_protectorate FROM sectors ORDER BY id`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	sectors := []sectorAdminInfo{}
	maxID := 0
	for rows.Next() {
		var id int
		var isProt bool
		if err := rows.Scan(&id, &isProt); err != nil {
			return "", err
		}
		sectors = append(sectors, sectorAdminInfo{ID: id, IsProtectorate: isProt})
		if id > maxID {
			maxID = id
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(sectors) == 0 {
		return "(no sectors)", nil
	}

	// Ports.
	portRows, err := pool.Query(ctx, `SELECT sector_id FROM ports`)
	if err != nil {
		return "", err
	}
	defer portRows.Close()
	ports := map[int]bool{}
	for portRows.Next() {
		var sid int
		if err := portRows.Scan(&sid); err != nil {
			return "", err
		}
		ports[sid] = true
	}
	if err := portRows.Err(); err != nil {
		return "", err
	}

	// Planets + owners.
	planetRows, err := pool.Query(ctx, `
		SELECT
			pl.sector_id,
			pl.name,
			COALESCE(c.name, u.username, '') AS owner
		FROM planets pl
		LEFT JOIN corporations c ON c.id = pl.owner_corp_id
		LEFT JOIN players p ON p.id = pl.owner_player_id
		LEFT JOIN users u ON u.id = p.user_id
	`)
	if err != nil {
		return "", err
	}
	defer planetRows.Close()

	planets := map[int]struct {
		name  string
		owner string
	}{}
	for planetRows.Next() {
		var sid int
		var name, owner string
		if err := planetRows.Scan(&sid, &name, &owner); err != nil {
			return "", err
		}
		planets[sid] = struct {
			name  string
			owner string
		}{name: name, owner: owner}
	}
	if err := planetRows.Err(); err != nil {
		return "", err
	}

	// Players by sector.
	playerRows, err := pool.Query(ctx, `
		SELECT p.sector_id, u.username
		FROM players p
		JOIN users u ON u.id = p.user_id
		ORDER BY p.sector_id, u.username
	`)
	if err != nil {
		return "", err
	}
	defer playerRows.Close()

	playersBySector := map[int][]string{}
	for playerRows.Next() {
		var sid int
		var username string
		if err := playerRows.Scan(&sid, &username); err != nil {
			return "", err
		}
		playersBySector[sid] = append(playersBySector[sid], username)
	}
	if err := playerRows.Err(); err != nil {
		return "", err
	}

	// Merge details.
	byID := map[int]*sectorAdminInfo{}
	for i := range sectors {
		s := &sectors[i]
		byID[s.ID] = s
		s.HasPort = ports[s.ID]
		if pl, ok := planets[s.ID]; ok {
			s.PlanetName = pl.name
			s.PlanetOwner = pl.owner
		}
		s.PlayerNames = playersBySector[s.ID]
	}

	// Layout.
	width := len(fmt.Sprintf("%d", maxID))
	cols := int(math.Ceil(math.Sqrt(float64(len(sectors)))))
	if cols < 10 {
		cols = 10
	}
	if cols > 25 {
		cols = 25
	}

	// Build map.
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Sovereign Conquest Universe Map (Admin)\n"))
	b.WriteString(fmt.Sprintf("Sectors: %d | Columns: %d\n\n", len(sectors), cols))
	b.WriteString("Cell format: <ID><G><S><P><N>\n")
	b.WriteString("  G = Protectorate sector (.)\n")
	b.WriteString("  S = Spaceport present (.)\n")
	b.WriteString("  P = Planet: . none | o unowned | A-Z owner initial\n")
	b.WriteString("  N = Players in sector: . none | 1-9 | + (10+)\n\n")

	for i, s := range sectors {
		gov := '.'
		if s.IsProtectorate {
			gov = 'G'
		}
		port := '.'
		if s.HasPort {
			port = 'S'
		}
		planetChar := '.'
		if s.PlanetName != "" {
			if s.PlanetOwner == "" {
				planetChar = 'o'
			} else {
				r := []rune(strings.TrimSpace(s.PlanetOwner))
				if len(r) > 0 {
					planetChar = rune(strings.ToUpper(string(r[0])))[0]
				}
			}
		}
		pcount := '.'
		if n := len(s.PlayerNames); n > 0 {
			if n >= 10 {
				pcount = '+'
			} else {
				pcount = rune('0' + n)
			}
		}

		cell := fmt.Sprintf("%0*d%c%c%c%c", width, s.ID, gov, port, planetChar, pcount)
		b.WriteString(cell)
		if (i+1)%cols == 0 {
			b.WriteString("\n")
		} else {
			b.WriteString(" ")
		}
	}
	if len(sectors)%cols != 0 {
		b.WriteString("\n")
	}

	// Details.
	b.WriteString("\nOwned planets:\n")
	owned := []struct {
		sid   int
		name  string
		owner string
	}{}
	for _, s := range sectors {
		if s.PlanetName == "" || s.PlanetOwner == "" {
			continue
		}
		owned = append(owned, struct {
			sid   int
			name  string
			owner string
		}{sid: s.ID, name: s.PlanetName, owner: s.PlanetOwner})
	}
	if len(owned) == 0 {
		b.WriteString("  (none)\n")
	} else {
		sort.Slice(owned, func(i, j int) bool { return owned[i].sid < owned[j].sid })
		for _, o := range owned {
			b.WriteString(fmt.Sprintf("  Sector %d: %s (Owner: %s)\n", o.sid, o.name, o.owner))
		}
	}

	b.WriteString("\nPlayers by sector:\n")
	sectorsWithPlayers := []int{}
	for sid, list := range playersBySector {
		if len(list) > 0 {
			sectorsWithPlayers = append(sectorsWithPlayers, sid)
		}
	}
	if len(sectorsWithPlayers) == 0 {
		b.WriteString("  (none)\n")
	} else {
		sort.Ints(sectorsWithPlayers)
		for _, sid := range sectorsWithPlayers {
			b.WriteString(fmt.Sprintf("  Sector %d: %s\n", sid, strings.Join(playersBySector[sid], ", ")))
		}
	}

	b.WriteString("\nProtectorate sectors:\n")
	protRows2, err := pool.Query(ctx, `SELECT id, protectorate_fighters FROM sectors WHERE is_protectorate=true ORDER BY id`)
	if err != nil {
		return "", err
	}
	defer protRows2.Close()
	protCount := 0
	for protRows2.Next() {
		var sid int
		var fighters int
		if err := protRows2.Scan(&sid, &fighters); err != nil {
			return "", err
		}
		protCount++
		b.WriteString(fmt.Sprintf("  Sector %d: %d fighters (Major port + shipyard)\n", sid, fighters))
	}
	if err := protRows2.Err(); err != nil {
		return "", err
	}
	if protCount == 0 {
		b.WriteString("  (none)\n")
	}

	return b.String(), nil
}
