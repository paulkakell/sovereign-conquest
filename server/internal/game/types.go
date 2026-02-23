package game

import "time"

type Player struct {
	ID             string
	UserID         string
	Username       string
	IsAdmin        bool
	MustChangePass bool
	Credits        int64
	Turns          int
	TurnsMax       int
	SectorID       int
	CargoMax       int
	CargoOre       int
	CargoOrganics  int
	CargoEquipment int
	LastTurnRegen  time.Time

	SeasonID   int
	SeasonName string

	CorpID      string
	CorpName    string
	CorpRole    string
	CorpCredits int64
}

type PlayerState struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	Username       string `json:"username"`
	IsAdmin        bool   `json:"is_admin"`
	MustChangePass bool   `json:"must_change_password"`
	Credits        int64  `json:"credits"`
	Turns          int    `json:"turns"`
	TurnsMax       int    `json:"turns_max"`
	SectorID       int    `json:"sector_id"`
	CargoMax       int    `json:"cargo_max"`
	CargoOre       int    `json:"cargo_ore"`
	CargoOrganics  int    `json:"cargo_organics"`
	CargoEquipment int    `json:"cargo_equipment"`

	SeasonID   int    `json:"season_id"`
	SeasonName string `json:"season_name"`

	CorpID      string `json:"corp_id,omitempty"`
	CorpName    string `json:"corp_name,omitempty"`
	CorpRole    string `json:"corp_role,omitempty"`
	CorpCredits int64  `json:"corp_credits,omitempty"`
}

func (p Player) ToState() PlayerState {
	return PlayerState{
		ID:             p.ID,
		UserID:         p.UserID,
		Username:       p.Username,
		IsAdmin:        p.IsAdmin,
		MustChangePass: p.MustChangePass,
		Credits:        p.Credits,
		Turns:          p.Turns,
		TurnsMax:       p.TurnsMax,
		SectorID:       p.SectorID,
		CargoMax:       p.CargoMax,
		CargoOre:       p.CargoOre,
		CargoOrganics:  p.CargoOrganics,
		CargoEquipment: p.CargoEquipment,
		SeasonID:       p.SeasonID,
		SeasonName:     p.SeasonName,
		CorpID:         p.CorpID,
		CorpName:       p.CorpName,
		CorpRole:       p.CorpRole,
		CorpCredits:    p.CorpCredits,
	}
}

type PortView struct {
	OreMode          string `json:"ore_mode"`
	OreQty           int    `json:"ore_qty"`
	OreBaseQty       int    `json:"ore_base_qty"`
	OrePrice         int    `json:"ore_price"`
	OrganicsMode     string `json:"organics_mode"`
	OrganicsQty      int    `json:"organics_qty"`
	OrganicsBaseQty  int    `json:"organics_base_qty"`
	OrganicsPrice    int    `json:"organics_price"`
	EquipmentMode    string `json:"equipment_mode"`
	EquipmentQty     int    `json:"equipment_qty"`
	EquipmentBaseQty int    `json:"equipment_base_qty"`
	EquipmentPrice   int    `json:"equipment_price"`
}

type PlanetView struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	OwnerType           string `json:"owner_type,omitempty"` // PLAYER | CORP
	Owner               string `json:"owner,omitempty"`
	ProductionOre       int    `json:"production_ore"`
	ProductionOrganics  int    `json:"production_organics"`
	ProductionEquipment int    `json:"production_equipment"`
	StorageOre          int    `json:"storage_ore"`
	StorageOrganics     int    `json:"storage_organics"`
	StorageEquipment    int    `json:"storage_equipment"`
	StorageMax          int    `json:"storage_max"`
	CitadelLevel        int    `json:"citadel_level"`
}

type EventView struct {
	Kind         string    `json:"kind"`
	SectorID     int       `json:"sector_id"`
	Commodity    string    `json:"commodity"`
	PricePercent int       `json:"price_percent"`
	Severity     int       `json:"severity"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	EndsAt       time.Time `json:"ends_at"`
}

type SectorView struct {
	ID     int         `json:"id"`
	Name   string      `json:"name"`
	Warps  []int       `json:"warps"`
	Port   *PortView   `json:"port,omitempty"`
	Planet *PlanetView `json:"planet,omitempty"`
	Event  *EventView  `json:"event,omitempty"`
	Mines  int         `json:"mines"`
}

type CommandRequest struct {
	Type      string `json:"type"`
	To        int    `json:"to,omitempty"`
	Action    string `json:"action,omitempty"`    // subcommand (PLANET/CORP/MINE) or BUY/SELL
	Commodity string `json:"commodity,omitempty"` // ORE, ORGANICS, EQUIPMENT
	Quantity  int    `json:"quantity,omitempty"`
	Name      string `json:"name,omitempty"`
	Text      string `json:"text,omitempty"`
}

type CommandResponse struct {
	OK      bool        `json:"ok"`
	Message string      `json:"message"`
	Error   string      `json:"error,omitempty"`
	State   PlayerState `json:"state"`
	Sector  SectorView  `json:"sector"`
	Logs    []LogEntry  `json:"logs,omitempty"`
}

type LogEntry struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	Message string    `json:"message"`
}
