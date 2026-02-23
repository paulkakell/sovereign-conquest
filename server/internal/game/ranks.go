package game

import "strings"

// RankNames are the canonical rank titles for each player level.
// Level 1 corresponds to RankNames[0].
var RankNames = []string{
	"Recruit",
	"Space Recruit",
	"Orbital Initiate",
	"Stellar Initiate",
	"Cadet",
	"Junior Cadet",
	"Senior Cadet",
	"Crewman Apprentice",
	"Crewman",
	"Senior Crewman",
	"Specialist Fourth Class",
	"Specialist Third Class",
	"Specialist Second Class",
	"Specialist First Class",
	"Technical Specialist",
	"Advanced Technical Specialist",
	"Flight Corporal",
	"Corporal",
	"Senior Corporal",
	"Tactical Corporal",
	"Sergeant",
	"Tactical Sergeant",
	"Orbital Sergeant",
	"Stellar Sergeant",
	"Gunnery Sergeant",
	"Master Sergeant",
	"First Sergeant",
	"Senior First Sergeant",
	"Command Sergeant",
	"Command Sergeant Major",
	"Fleet Sergeant Major",
	"Sector Sergeant Major",
	"Stellar Sergeant Major",
	"High Command Sergeant Major",
	"Enlisted Command Chief",
	"Enlisted High Chief",
	"Enlisted Fleet Chief",
	"Enlisted Stellar Chief",
	"Senior Enlisted Advisor",
	"Supreme Enlisted Advisor",
	"Warrant Officer Candidate",
	"Warrant Officer One",
	"Warrant Officer Two",
	"Warrant Officer Three",
	"Warrant Officer Four",
	"Warrant Officer Five",
	"Senior Warrant Officer",
	"Chief Warrant Officer",
	"Master Warrant Officer",
	"Fleet Warrant Officer",
	"Sector Warrant Officer",
	"Stellar Warrant Officer",
	"High Warrant Officer",
	"Grand Warrant Officer",
	"Supreme Warrant Officer",
	"Ensign",
	"Sub-Lieutenant",
	"Lieutenant Junior Grade",
	"Lieutenant",
	"Senior Lieutenant",
	"Flight Lieutenant",
	"Orbital Lieutenant",
	"Stellar Lieutenant",
	"Captain",
	"Senior Captain",
	"Fleet Captain",
	"Wing Captain",
	"Group Captain",
	"Squadron Captain",
	"Major",
	"Wing Major",
	"Group Major",
	"Sector Major",
	"Fleet Major",
	"Lieutenant Colonel",
	"Senior Lieutenant Colonel",
	"Fleet Lieutenant Colonel",
	"Sector Lieutenant Colonel",
	"Stellar Lieutenant Colonel",
	"Colonel",
	"Senior Colonel",
	"Fleet Colonel",
	"Sector Colonel",
	"Stellar Colonel",
	"Command Colonel",
	"High Colonel",
	"Grand Colonel",
	"Supreme Colonel",
	"Brigadier",
	"Brigadier General",
	"Senior Brigadier General",
	"Fleet Brigadier General",
	"Sector Brigadier General",
	"Major General",
	"Senior Major General",
	"Fleet Major General",
	"Sector Major General",
	"Stellar Major General",
	"Lieutenant General",
	"Senior Lieutenant General",
	"Fleet Lieutenant General",
	"Sector Lieutenant General",
	"Stellar Lieutenant General",
	"General",
	"Senior General",
	"Fleet General",
	"Sector General",
	"Stellar General",
	"High General",
	"Grand General",
	"Supreme General",
	"Commodore",
	"Senior Commodore",
	"Fleet Commodore",
	"Sector Commodore",
	"Stellar Commodore",
	"Rear Admiral",
	"Upper Rear Admiral",
	"Fleet Rear Admiral",
	"Sector Rear Admiral",
	"Vice Admiral",
	"Senior Vice Admiral",
	"Fleet Vice Admiral",
	"Sector Vice Admiral",
	"Stellar Vice Admiral",
	"Admiral",
	"Senior Admiral",
	"Fleet Admiral",
	"Sector Admiral",
	"Stellar Admiral",
	"High Admiral",
	"Grand Admiral",
	"Supreme Admiral",
	"Marshal",
	"Fleet Marshal",
	"Sector Marshal",
	"Stellar Marshal",
	"High Marshal",
	"Grand Marshal",
	"Supreme Marshal",
	"Star Marshal",
	"Grand Star Marshal",
	"Supreme Star Marshal",
	"High Commander",
	"Fleet Commander",
	"Sector Commander",
	"Stellar Commander",
	"Grand Commander",
	"Supreme Commander",
	"High Strategic Commander",
	"Fleet Strategic Commander",
	"Sector Strategic Commander",
	"Stellar Strategic Commander",
	"Grand Strategic Commander",
	"Supreme Strategic Commander",
	"Overmarshal",
	"Fleet Overmarshal",
	"Sector Overmarshal",
	"Stellar Overmarshal",
	"High Overmarshal",
	"Grand Overmarshal",
	"Supreme Overmarshal",
	"Lord Commander",
	"High Lord Commander",
	"Grand Lord Commander",
	"Stellar Lord Commander",
	"Sector Lord Commander",
	"Fleet Lord Commander",
	"Supreme Lord Commander",
	"Star Lord",
	"High Star Lord",
	"Grand Star Lord",
	"Stellar Star Lord",
	"Sector Star Lord",
	"Fleet Star Lord",
	"Supreme Star Lord",
	"Arch Commander",
	"High Arch Commander",
	"Grand Arch Commander",
	"Supreme Arch Commander",
	"Arch Marshal",
	"High Arch Marshal",
	"Grand Arch Marshal",
	"Supreme Arch Marshal",
	"Prime Commander",
	"High Prime Commander",
	"Grand Prime Commander",
	"Supreme Prime Commander",
	"Prime Marshal",
	"High Prime Marshal",
	"Grand Prime Marshal",
	"Supreme Prime Marshal",
	"Celestial Commander",
	"High Celestial Commander",
	"Grand Celestial Commander",
	"Supreme Celestial Commander",
	"Celestial Marshal",
	"High Celestial Marshal",
	"Grand Celestial Marshal",
	"Supreme Celestial Marshal",
}

const (
	// MaxPlayerLevel is the maximum attainable level/rank.
	MaxPlayerLevel = 200

	// xpBasePerLevel controls the overall slope of level progression.
	// XP required to reach level L is: base * (L-1) * L / 2
	xpBasePerLevel int64 = 50
)

func RankNameForLevel(level int) string {
	if level < 1 {
		level = 1
	}
	if level > len(RankNames) {
		return RankNames[len(RankNames)-1]
	}
	return RankNames[level-1]
}

// XPForLevel returns the total XP required to reach the given level.
// Level 1 requires 0 XP.
func XPForLevel(level int) int64 {
	if level <= 1 {
		return 0
	}
	if level > MaxPlayerLevel {
		level = MaxPlayerLevel
	}
	// base*(L-1)*L/2
	l := int64(level)
	return xpBasePerLevel * (l - 1) * l / 2
}

// LevelForXP converts total XP into the corresponding level (clamped to MaxPlayerLevel).
func LevelForXP(xp int64) int {
	if xp < 0 {
		xp = 0
	}
	lo, hi := 1, MaxPlayerLevel
	best := 1
	for lo <= hi {
		mid := (lo + hi) / 2
		if XPForLevel(mid) <= xp {
			best = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	if best < 1 {
		best = 1
	}
	return best
}

// XPGainForCommand determines how much XP a successful command awards.
// This is intentionally simple and can be tuned later.
func XPGainForCommand(cmd CommandRequest, cost int) int64 {
	t := strings.ToUpper(strings.TrimSpace(cmd.Type))
	a := strings.ToUpper(strings.TrimSpace(cmd.Action))

	switch t {
	case "SCAN":
		return 10
	case "MOVE":
		return 10
	case "TRADE":
		return 15
	case "PLANET":
		switch a {
		case "COLONIZE":
			return 120
		case "UPGRADE_CITADEL":
			return 60
		case "LOAD", "UNLOAD":
			return 12
		default:
			return 4
		}
	case "CORP":
		switch a {
		case "CREATE":
			return 50
		case "JOIN":
			return 25
		case "LEAVE":
			return 10
		case "SAY":
			return 2
		case "DEPOSIT", "WITHDRAW":
			return 5
		default:
			return 2
		}
	case "MINE":
		switch a {
		case "DEPLOY":
			return 25
		case "SWEEP":
			return 20
		default:
			return 2
		}
	case "MARKET", "ROUTE", "EVENTS":
		return 3
	case "RANKINGS", "SEASON":
		return 1
	case "HELP":
		return 1
	case "SHIPYARD":
		switch a {
		case "BUY":
			return 30
		case "SELL":
			return 10
		case "UPGRADE":
			return 12
		default:
			return 2
		}
	default:
		// Fall back to a small award so new command types still contribute.
		if cost < 1 {
			return 1
		}
		return int64(cost) * 5
	}
}

// AwardXP adds XP to the player and updates their level.
func AwardXP(p *Player, delta int64) (leveledUp bool, oldLevel, newLevel int) {
	if p == nil {
		return false, 0, 0
	}
	if delta <= 0 {
		return false, p.Level, p.Level
	}

	oldLevel = LevelForXP(p.XP)
	p.XP += delta
	if p.XP < 0 {
		p.XP = 0
	}
	newLevel = LevelForXP(p.XP)
	p.Level = newLevel
	return newLevel > oldLevel, oldLevel, newLevel
}
