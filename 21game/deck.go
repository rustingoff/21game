package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

type CardType int

const (
	Seven CardType = 7
	Eight CardType = 8
	Nine  CardType = 9
	Ten   CardType = 10
	Jack  CardType = 2
	Queen CardType = 3
	King  CardType = 4
	Ace   CardType = 11
)

type Suit int

const (
	Heart   Suit = 1
	Diamond Suit = 2
	Club    Suit = 3
	Spade   Suit = 4
)

type Card struct {
	Type CardType
	Suit Suit
}

func (c *Card) Value() int {
	return int(c.Type)
}

type Deck interface {
	Shuffle()
	Deal() Card
	Debug()
}

// Deck holds the cards in the Deck to be shuffled
type deck struct {
	Cards []Card
}

type Player interface {
	Score()
	GetSeat() int
	GetPresence() runtime.Presence
	GetBet() int
	SetSeat(seat int)
	IsBanker() bool
	GetID() string
	SetBanker(b bool)
	SetHand(cards []Card)
	SetBet(bet int)
}

func NewPlayer(presence runtime.Presence) Player {
	p := &player{
		Presence: presence,
		Seat:     -1,
	}
	return p
}

type player struct {
	Presence runtime.Presence
	Hand     []Card
	Banker   bool
	Bet      int
	Seat     int
}

func (p *player) Score() {
	sum := 0
	for _, card := range p.Hand {
		sum += card.Value()
	}
}
func (p *player) GetSeat() int {
	return p.Seat
}
func (p *player) GetBet() int {
	return p.Bet
}
func (p *player) SetSeat(seat int) {
	p.Seat = seat
}
func (p *player) GetPresence() runtime.Presence {
	return p.Presence
}
func (p *player) IsBanker() bool {
	return p.Banker
}
func (p *player) GetID() string {
	return p.Presence.GetUserId()
}
func (p *player) SetBanker(b bool) {
	p.Banker = b
}
func (p *player) SetHand(cards []Card) {
	p.Hand = cards
}
func (p *player) SetBet(bet int) {
	p.Bet = bet
}

// New creates a Deck of cards to be used
func NewDeck() Deck {
	d := &deck{}

	rand.Seed(time.Now().UnixNano())

	types := []CardType{Seven, Eight, Nine, Ten, Jack, Queen, King, Ace}

	// Valid suits include Heart, Diamond, Club & Spade
	suits := []Suit{Heart, Diamond, Club, Spade}

	// Loop over each type and suit appending to the Deck
	for i := 0; i < len(types); i++ {
		for n := 0; n < len(suits); n++ {
			card := Card{
				Type: types[i],
				Suit: suits[n],
			}
			d.Cards = append(d.Cards, card)
		}
	}
	return d
}

// Shuffle the Deck
func (d *deck) Shuffle() {
	for i := 1; i < len(d.Cards); i++ {
		// Create a random int up to the number of cards
		r := rand.Intn(i + 1)

		// If the the current card doesn't match the random
		// int we generated then we'll switch them out
		if i != r {
			d.Cards[r], d.Cards[i] = d.Cards[i], d.Cards[r]
		}
	}
}

func (d *deck) Deal() Card {
	c := d.Cards
	x := d.Cards[len(c)-1]
	return x
}

// Debug helps debugging the Deck of cards
func (d *deck) Debug() {
	if os.Getenv("DEBUG") != "" {
		for i := 0; i < len(d.Cards); i++ {
			fmt.Printf("Card #%v is a %v of %vs\n", i+1, d.Cards[i].Type, d.Cards[i].Suit)
		}
	}
}
