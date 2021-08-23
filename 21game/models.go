package main

import (
	"errors"

	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	Password = "password"
	Size     = "size"
	IsOpen   = "isOpen"
	Title    = "title"
	MinBet   = "minBet"
	MaxBet   = "maxBet"
	Timeout  = "timeout"
	//Wallet
	Coins = "coins"
)

type GameState int

const (
	OpCodeTerminate int64 = iota
	OpCodeMatchMessageResponse
	OpCodeWaiting
	OpCodeBanker
	OpCodeBetting
	OpCodeBet
	OpCodeSit
	OpCodeSeat
)

const (
	Idle GameState = iota
	Terminated
	Playing
	Betting
	BankerTurn
	PlayerTurn
)

const (
	ResponseCodeSuccess = iota
	ResponseCodeTableFull
	ResponseCodeBadRequest
	ResponseCodeInternalError
)

type PlayerType string

const (
	Seated   PlayerType = "seated"
	Watching PlayerType = "watching"
	All      PlayerType = "all"
)

type MatchMessage struct {
	Id   *int64 `json:"id"`
	Data []byte `json:"data"`
}

type MatchMessageResponse struct {
	Id           int64 `json:"id"`
	ResponseCode int64 `json:"response_code"`
}

type MatchConfig interface {
	GetSize() int
	GetTitle() string
	GetMinBet() float64
	GetPassword() string
}

type matchConfig struct {
	Password string
	// Number of Players
	Size   int
	IsOpen bool
	Title  string
	MinBet float64
	MaxBet float64
	// Maximum time allowed to act
	Timeout int
}

func (c matchConfig) GetSize() int {
	return c.Size
}
func (c matchConfig) GetTitle() string {
	return c.Title
}
func (c matchConfig) GetMinBet() float64 {
	return c.MinBet
}
func (c matchConfig) GetPassword() string {
	return c.Password
}

func NewMatchConfig(params map[string]interface{}) (MatchConfig, error) {
	config := &matchConfig{}
	var err error = nil

	if s, ok := params[Size]; ok {
		if sv, ok := s.(int); ok {
			config.Size = sv
		}
	}

	if p, ok := params[Password]; ok {
		if pv, ok := p.(string); ok {
			config.Password = pv
		}
	}

	if o, ok := params[IsOpen]; ok {
		if ov, ok := o.(bool); ok {
			config.IsOpen = ov
		}
	}

	if config.IsOpen && config.Password == "" {
		err = errors.New("private matches need to have a pass set")
		return nil, err
	}

	if t, ok := params[Title]; ok {
		if tv, ok := t.(string); ok {
			config.Title = tv
		}
	}

	if config.Title == "" {
		err = errors.New("match title is required")
		return nil, err
	}

	if mb, ok := params[MinBet]; ok {
		if mbv, ok := mb.(float64); ok {
			config.MinBet = mbv
		}
	}

	if mb, ok := params[MaxBet]; ok {
		if mxbv, ok := mb.(float64); ok {
			config.MaxBet = mxbv
		}
	}

	if config.MinBet == 0 || config.MaxBet == 0 || config.MinBet > config.MaxBet {
		err = errors.New("invalid match bet values")
		return nil, err
	}

	if t, ok := params[Timeout]; ok {
		if tv, ok := t.(int); ok {
			config.Timeout = tv
		}
	}
	if config.Timeout < 1000 {
		err = errors.New("invalid match timeout values")
		return nil, err
	}

	return config, nil
}

type MatchState interface {
	GetConfig() MatchConfig
	GetPlayers(PlayerType) []Player
	AddPlayer(runtime.Presence)
	RemovePlayer(runtime.Presence)
	GetState() GameState
	SetState(GameState)
	GetPot() float64
	GetPlayer(id string) Player
	IsSeatAvailable(int) bool
	GetBanker() Player
	SetBanker(id string)
	SetDeck(newDeck Deck)
	SetPlayer(id string)
}

type matchState struct {
	Players map[string]Player
	Deck    Deck
	Config  MatchConfig
	// Amount in the bank
	Pot    float64
	Round  int
	Player string
	Game   GameState
}

func (m *matchState) GetPot() float64 {
	return m.Pot
}

func (m *matchState) GetPlayer(id string) Player {
	return m.Players[id]
}

func NewMatchState(deck Deck, config MatchConfig) (m MatchState) {
	m = &matchState{
		Players: make(map[string]Player),
		Deck:    deck,
		Config:  config,
		Pot:     0,
	}
	return m
}
func (m *matchState) GetPlayers(t PlayerType) []Player {
	players := make([]Player, 0)

	for _, p := range m.Players {
		switch t {
		case Seated:
			{
				if p.GetSeat() != -1 {
					players = append(players, p)
				}
			}
		case Watching:
			{
				if p.GetSeat() == -1 {
					players = append(players, p)
				}
			}

		case All:
			{
				players = append(players, p)
			}
		}
	}
	return players
}
func (m *matchState) GetConfig() MatchConfig {
	return m.Config
}
func (m *matchState) AddPlayer(p runtime.Presence) {
	m.Players[p.GetUserId()] = NewPlayer(p)
}
func (m *matchState) GetState() GameState {
	return m.Game
}
func (m *matchState) SetState(s GameState) {
	m.Game = s
}
func (m *matchState) RemovePlayer(p runtime.Presence) {
	delete(m.Players, p.GetUserId())
	// TODO Delete from Table
	// TODO Send message
}
func (m *matchState) IsSeatAvailable(seat int) bool {
	if m.GetConfig().GetSize() < seat {
		return false
	}

	for _, player := range m.Players {
		if player.GetSeat() == seat {
			return false
		}
	}

	return true
}
func (m *matchState) GetBanker() Player {
	for _, player := range m.Players {
		if player.IsBanker() {
			return player
		}
	}
	return nil
}
func (m *matchState) SetBanker(id string) {
	for _, player := range m.Players {
		player.SetBanker(player.GetID() == id)
	}
}
func (m *matchState) SetDeck(newDeck Deck) {
	m.Deck = newDeck
}
func (m *matchState) SetPlayer(id string) {
	m.Player = id
}
