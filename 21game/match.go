package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/heroiclabs/nakama-common/runtime"
)

type Match struct{}

func (m *Match) MatchInit(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]interface{}) (interface{}, int, string) {
	config, _ := NewMatchConfig(params)
	//if err != nil {
	//	logger.Error("Failed to create match with Config error %v", err)
	//	return nil, 0, "", err
	//}

	deck := NewDeck()
	state := NewMatchState(deck, config)
	tickRate := 1
	label := config.GetTitle()
	return state, tickRate, label
}

func (m *Match) MatchJoinAttempt(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presence runtime.Presence, metadata map[string]string) (interface{}, bool, string) {
	mState := state.(MatchState)
	acceptUser := true
	var password = fmt.Sprintf("%v", metadata[Password])
	config := mState.GetConfig()
	if password != config.GetPassword() {
		logger.Warn("failed to join match with bc wrong password")
		acceptUser = false
	}
	return state, acceptUser, ""
}

func (m *Match) getWallet(nk runtime.NakamaModule, ctx context.Context, presence runtime.Presence, logger runtime.Logger) (float64, error) {
	account, err := nk.AccountGetId(ctx, presence.GetUserId())
	if err != nil {
		return 0, err
	}

	var wallet map[string]interface{}
	err = json.Unmarshal([]byte(account.Wallet), &wallet)
	if err != nil {
		return 0, err
	}

	coins := float64(0)
	if c, ok := wallet[Coins]; ok {
		if cv, ok := c.(float64); ok {
			coins = cv
		}
	}
	return coins, nil
}

func (m *Match) MatchJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	mState, _ := state.(MatchState)
	for _, p := range presences {
		mState.AddPlayer(p)
	}
	return mState
}

func (m *Match) MatchLeave(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	mState, _ := state.(MatchState)
	for _, p := range presences {
		mState.RemovePlayer(p)
	}
	return mState
}

func (m *Match) MatchLoop(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, messages []runtime.MatchData) interface{} {
	mState, _ := state.(MatchState)

	// If the match is now empty stop it now.
	if len(mState.GetPlayers(All)) == 0 && tick > 2*60 {
		logger.Info("Terminating match as there are no Players")
		err := dispatcher.BroadcastMessage(OpCodeTerminate, nil, nil, nil, false)
		if err != nil {
			logger.Warn("Error broadcasting match terminate event: %v", err)
		}
		return nil
	}

	if len(mState.GetPlayers(Seated)) == 1 {
		if mState.GetState() == Playing {
			// Stop and reset
			for _, player := range mState.GetPlayers(Seated) {
				player.SetBanker(false)
				player.SetHand([]Card{})
				player.SetBet(0)
				logger.Info("User left playing: %v", player.GetPresence())
			}

			mState.SetDeck(NewDeck())
			mState.SetBanker("")
			mState.SetPlayer("")
			mState.SetState(Idle)
			// TODO What we do with the money int the bank - add to the IsBanker wallet
			err := dispatcher.BroadcastMessage(OpCodeWaiting, nil, nil, nil, true)
			if err != nil {
				logger.Warn("Error broadcasting terminate event: %v", err)
			}
		}
		// TODO Should assign Player as it passed by value?
		return mState
	} else if len(mState.GetPlayers(Seated)) > 1 {
		// Check if we have at least 2 Players sited
		if mState.GetState() == Idle {
			if mState.GetBanker() == nil && len(mState.GetPlayers(Seated)) > 1 {
				players := mState.GetPlayers(Seated)
				banker := rand.Intn(len(players) + 1)
				mState.SetBanker(players[banker].GetID())
				err := dispatcher.BroadcastMessage(OpCodeBanker, nil, nil, nil, true)
				if err != nil {
					logger.Warn("failed broadcasting IsBanker event: %v", err)
				}
				mState.SetState(Betting)
				err = dispatcher.BroadcastMessage(OpCodeBetting, nil, nil, nil, true)
				if err != nil {
					logger.Warn("failed broadcasting betting event: %v", err)
				}

				return state
			}
			// 1. Wait for bets from Players
			// 2.1 Deal 2 cards to every Player except banker
			// 2.2 Deal 2 cards to the banker
		} else {
			if mState.GetState() == Betting {
				for _, message := range messages {
					var matchMsg MatchMessage
					err := json.Unmarshal(message.GetData(), &matchMsg)
					if err != nil {
						logger.Warn("Unable to unmarshal MatchMessage: %v", err)
						continue
					}

					switch message.GetOpCode() {
					case OpCodeBet:
						if user := mState.GetPlayer(message.GetUserId()); user.GetPresence() != nil && user.GetBet() != 0 {
							continue
						}
					}
				}
			}

			if mState.GetPot() < mState.GetConfig().GetMinBet() {
				return nil
			}
		}
	}

	// TODO Handle Sit
	if mState.GetState() != Betting && mState.GetState() != Terminated {
		for _, message := range messages {
			var matchMsg MatchMessage
			err := json.Unmarshal(message.GetData(), &matchMsg)
			if err != nil {
				logger.Warn("unable to unmarshal MatchMessage: %v", err)
				continue
			}

			switch message.GetOpCode() {
			case OpCodeSit:
				player := mState.GetPlayer(message.GetUserId())
				if player == nil {
					logger.Warn("unable to find user id: %v", message.GetUserId())
					continue
				}

				seat, err := strconv.Atoi(string(matchMsg.Data))

				if err != nil {
					logger.Warn("unable to parse seat: %v with error %v", string(matchMsg.Data), err)
					SendResponse(matchMsg, ResponseCodeBadRequest, []runtime.Presence{message}, logger, dispatcher)
					continue
				}
				if mState.GetConfig().GetSize() <= len(mState.GetPlayers(Seated)) {
					logger.Warn("unable to join match %v bc its full", mState.GetConfig().GetTitle())
					SendResponse(matchMsg, ResponseCodeTableFull, []runtime.Presence{message}, logger, dispatcher)
					continue
				}
				if mState.IsSeatAvailable(int(seat)) {
					logger.Warn("invalid seat %v to join match %v", seat, mState.GetConfig().GetTitle())
					SendResponse(matchMsg, ResponseCodeBadRequest, []runtime.Presence{message}, logger, dispatcher)
					continue
				}

				if user := mState.GetPlayer(message.GetUserId()); user.GetPresence() != nil && user.GetSeat() == 0 {
					wallet, err := m.getWallet(nk, ctx, message, logger)
					if err != nil {
						logger.Warn("failed to check wallet for user %v", err)
						SendResponse(matchMsg, ResponseCodeBadRequest, []runtime.Presence{message}, logger, dispatcher)
						continue
					}
					if wallet < mState.GetConfig().GetMinBet()*3 {
						logger.Info("not enough coins to join match")
						SendResponse(matchMsg, ResponseCodeBadRequest, []runtime.Presence{message}, logger, dispatcher)
						continue
					}

					mState.GetPlayer(message.GetUserId()).SetSeat(seat)
					err = dispatcher.BroadcastMessage(OpCodeSeat, []byte(message.GetUserId()), nil, nil, true)
					if err != nil {
						logger.Warn("error dispatching OpCodeSeat: %v", err)
						continue
					}
				} else {
					logger.Warn("invalid seat %v to join match %v", seat, mState.GetConfig().GetTitle())
					SendResponse(matchMsg, ResponseCodeBadRequest, []runtime.Presence{message}, logger, dispatcher)
					continue
				}
			}
		}
	}

	return mState
}

func (m *Match) MatchTerminate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, graceSeconds int) interface{} {
	message := "Server shutting down in " + strconv.Itoa(graceSeconds) + " seconds."
	dispatcher.BroadcastMessage(2, []byte(message), nil, nil, false)
	return state
}

func SendResponse(partyMsg MatchMessage, responseCode int64, to []runtime.Presence, logger runtime.Logger, dispatcher runtime.MatchDispatcher) {
	if partyMsg.Id != nil {
		response := MatchMessageResponse{
			Id:           *partyMsg.Id,
			ResponseCode: responseCode,
		}

		responseBytes, err := json.Marshal(response)
		if err != nil {
			logger.Error("Error marshalling match message response: %v", err)
			return
		}

		if err := dispatcher.BroadcastMessage(OpCodeMatchMessageResponse, responseBytes, to, nil, true); err != nil {
			logger.Error("Error broadcasting match message response: %v", err)
		}
	}
}
