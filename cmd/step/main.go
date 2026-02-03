// cmd/step is a CLI tool for stepping a Battlesnake game forward one turn.
// It reads a JSON request from stdin and writes the resulting board state to stdout.
// This is used for cross-engine verification with the Rust implementation.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	rules "github.com/BattlesnakeOfficial/rules"
	"github.com/BattlesnakeOfficial/rules/maps"
)

// StepRequest is the JSON input for stepping a game forward one turn.
type StepRequest struct {
	Seed  int64           `json:"seed"`
	Turn  int             `json:"turn"`
	Board BoardStateJSON  `json:"board"`
	Moves []SnakeMoveJSON `json:"moves"`

	// Game settings (optional, defaults to standard)
	FoodSpawnChance int `json:"food_spawn_chance,omitempty"`
	MinimumFood     int `json:"minimum_food,omitempty"`
}

// BoardStateJSON is the JSON representation of a board state.
type BoardStateJSON struct {
	Width   int             `json:"width"`
	Height  int             `json:"height"`
	Food    []PointJSON     `json:"food"`
	Snakes  []SnakeJSON     `json:"snakes"`
	Hazards []PointJSON     `json:"hazards"`
}

type PointJSON struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type SnakeJSON struct {
	ID              string      `json:"id"`
	Body            []PointJSON `json:"body"`
	Health          int         `json:"health"`
	EliminatedCause string     `json:"eliminated_cause,omitempty"`
	EliminatedOnTurn int       `json:"eliminated_on_turn,omitempty"`
	EliminatedBy    string     `json:"eliminated_by,omitempty"`
}

type SnakeMoveJSON struct {
	ID   string `json:"id"`
	Move string `json:"move"`
}

// StepResponse is the JSON output after stepping a game forward.
type StepResponse struct {
	GameOver bool           `json:"game_over"`
	Board    BoardStateJSON `json:"board"`
}

func toRulesPoint(p PointJSON) rules.Point {
	return rules.Point{X: p.X, Y: p.Y}
}

func toJSONPoint(p rules.Point) PointJSON {
	return PointJSON{X: p.X, Y: p.Y}
}

func toRulesPoints(points []PointJSON) []rules.Point {
	result := make([]rules.Point, len(points))
	for i, p := range points {
		result[i] = toRulesPoint(p)
	}
	return result
}

func toJSONPoints(points []rules.Point) []PointJSON {
	result := make([]PointJSON, len(points))
	for i, p := range points {
		result[i] = toJSONPoint(p)
	}
	return result
}

func toBoardState(b BoardStateJSON, turn int) *rules.BoardState {
	state := rules.NewBoardState(b.Width, b.Height)
	state.Turn = turn

	state.Food = toRulesPoints(b.Food)
	state.Hazards = toRulesPoints(b.Hazards)

	state.Snakes = make([]rules.Snake, len(b.Snakes))
	for i, s := range b.Snakes {
		state.Snakes[i] = rules.Snake{
			ID:               s.ID,
			Health:           s.Health,
			Body:             toRulesPoints(s.Body),
			EliminatedCause:  s.EliminatedCause,
			EliminatedOnTurn: s.EliminatedOnTurn,
			EliminatedBy:     s.EliminatedBy,
		}
	}

	return state
}

func toBoardStateJSON(state *rules.BoardState) BoardStateJSON {
	snakes := make([]SnakeJSON, len(state.Snakes))
	for i, s := range state.Snakes {
		snakes[i] = SnakeJSON{
			ID:               s.ID,
			Health:           s.Health,
			Body:             toJSONPoints(s.Body),
			EliminatedCause:  s.EliminatedCause,
			EliminatedOnTurn: s.EliminatedOnTurn,
			EliminatedBy:     s.EliminatedBy,
		}
	}

	return BoardStateJSON{
		Width:   state.Width,
		Height:  state.Height,
		Food:    toJSONPoints(state.Food),
		Snakes:  snakes,
		Hazards: toJSONPoints(state.Hazards),
	}
}

func main() {
	var req StepRequest
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&req); err != nil {
		fmt.Fprintf(os.Stderr, "error decoding request: %v\n", err)
		os.Exit(1)
	}

	// Default settings
	foodSpawnChance := 15
	minimumFood := 1
	if req.FoodSpawnChance > 0 {
		foodSpawnChance = req.FoodSpawnChance
	}
	if req.MinimumFood > 0 {
		minimumFood = req.MinimumFood
	}

	// Build the ruleset
	params := map[string]string{
		rules.ParamFoodSpawnChance:     strconv.Itoa(foodSpawnChance),
		rules.ParamMinimumFood:         strconv.Itoa(minimumFood),
		rules.ParamHazardDamagePerTurn: "14",
	}

	ruleset := rules.NewRulesetBuilder().
		WithSeed(req.Seed).
		WithParams(params).
		NamedRuleset(rules.GameTypeStandard)

	// Convert JSON board to rules BoardState
	boardState := toBoardState(req.Board, req.Turn)

	// Convert moves
	moves := make([]rules.SnakeMove, len(req.Moves))
	for i, m := range req.Moves {
		moves[i] = rules.SnakeMove{
			ID:   m.ID,
			Move: m.Move,
		}
	}

	// Execute the ruleset (applies movement, collisions, feeding, elimination)
	gameOver, nextState, err := ruleset.Execute(boardState, moves)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error executing ruleset: %v\n", err)
		os.Exit(1)
	}

	// Apply post-update (food spawning) via the standard map
	standardMap := maps.StandardMap{}
	nextState, err = maps.PostUpdateBoard(standardMap, nextState, ruleset.Settings())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error in post-update: %v\n", err)
		os.Exit(1)
	}

	// Build response
	resp := StepResponse{
		GameOver: gameOver,
		Board:    toBoardStateJSON(nextState),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(resp); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding response: %v\n", err)
		os.Exit(1)
	}
}
