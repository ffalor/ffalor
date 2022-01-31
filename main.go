package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/notnil/chess"
	"github.com/notnil/chess/image"
	"gopkg.in/yaml.v2"
)

type stats struct {
	Moves        int               `yaml:"moves"`
	Wins         map[string]int    `yaml:"Wins"`
	PreviousGame map[string]string `yaml:"previous"`
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func outcome(o chess.Outcome) string {
	switch o {
	case chess.WhiteWon:
		return "White Won"
	case chess.BlackWon:
		return "Black Won"
	case chess.Draw:
		return "Draw"
	default:
		return "Game in Progress"
	}
}

func renderReadMe(game *chess.Game, f *os.File, boardFilePath string, stats stats) {

	moves := game.ValidMoves()

	outcome := outcome(game.Outcome())

	if outcome != "Game in Progress" {
		var msg string

		if outcome == "Draw" {
			msg = " by a draw."
		} else {
			msg = fmt.Sprint(outcome, " won by ", game.Method().String())
		}
		fmt.Fprintf(f, "**Current State**: Game over %s.\n\n", msg)

		fmt.Fprintf(f, "Click Here to reset the game")
	} else {
		fmt.Fprintf(f, "**Current State**: %s, %s's move.\n\n", outcome, game.Position().Turn().Name())
	}

	fmt.Fprintf(f, "![board](%s)", "https://raw.githubusercontent.com/ffalor/ffalor/main/state/board.svg?")
	fmt.Fprintln(f, "\n## Valid Moves")

	headers := [2]string{"Move From", "Move To (Click One)"}

	fmt.Fprintln(f, "|", headers[0], "|", headers[1], "|")
	fmt.Fprintln(f, "|", "---", "|", "---", "|")

	moveByStartingPosition := make(map[string][]string)

	for _, move := range moves {
		moveStr := strings.ToUpper(move.String())

		title := url.QueryEscape(fmt.Sprintf("move|%s", strings.ToLower(moveStr)))
		body := url.QueryEscape("Just press 'Submit Issue' to make this move. Please do not edit the title.")

		issueLink := fmt.Sprintf("[%s](https://github.com/ffalor/ffalor/issues/new?title=%s&body=%s)", moveStr[2:], title, body)

		moveByStartingPosition[moveStr[:2]] = append(moveByStartingPosition[moveStr[:2]], issueLink)
	}

	for key, value := range moveByStartingPosition {
		fmt.Fprintln(f, "|", key, "|", strings.Join(value, ", "), "|")

	}

	fmt.Fprintln(f, "\n## Previous Game")
	fmt.Fprintf(f, "**Winner:** %s\n\n", stats.PreviousGame["winner"])
	fmt.Fprintf(f, "**Method:** %s\n\n", stats.PreviousGame["method"])

	blackWins := stats.Wins["black"]
	whiteWins := stats.Wins["white"]
	draws := stats.Wins["draw"]

	fmt.Fprintln(f, "## Overall Stats")
	fmt.Fprintf(f, "**Total Games:** %d\n\n", (blackWins + whiteWins + draws))
	fmt.Fprintf(f, "**Total Moves:** %d \n\n", stats.Moves)
	fmt.Fprintf(f, "**White Wins:** %d\n\n", whiteWins)
	fmt.Fprintf(f, "**Black Wins:** %d\n\n", blackWins)
	fmt.Fprintf(f, "**Draws:** %d\n\n", draws)
}

func writeBoardImage(path string, board chess.Board) {

	board_image, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	check(err)
	defer board_image.Close()
	// Go to the start of the file to overwrite the previous image.
	board_image.Truncate(0)
	board_image.Seek(0, 0)
	check(err)
	err = image.SVG(board_image, &board)
	check(err)
}

func initializeGame(pgnFilePath string) *chess.Game {
	gameState, err := os.OpenFile(pgnFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	check(err)
	defer gameState.Close()

	pgn, err := chess.PGN(gameState)
	check(err)
	game := chess.NewGame(pgn, chess.UseNotation(chess.UCINotation{}))

	return game
}

func writeStatsFile(statsFilePath string, stats stats) {
	statsFile, err := os.OpenFile(statsFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	check(err)
	defer statsFile.Close()

	statsDump, err := yaml.Marshal(stats)
	check(err)
	fmt.Fprintf(statsFile, "%s\n", string(statsDump))
}

func main() {

	var gameStats string
	var readmeFilePath string
	var boardFilePath string
	var pgnFilePath string
	var userMove string
	var action string
	var title string

	flag.StringVar(&boardFilePath, "board", "./state/board.svg", "Path to write the board image to.")
	flag.StringVar(&pgnFilePath, "pgn", "./state/game.pgn", "Path to write the current game PGN file to.")
	flag.StringVar(&readmeFilePath, "readme", "./README.md", "Path to write the readme file to.")
	flag.StringVar(&title, "title", "", "Github title string to parse ex. move|a2a3.")
	flag.StringVar(&gameStats, "stats", "./state/stats.yml", "Path to write the stats file to.")

	flag.Parse()

	args := strings.Split(title, "|")

	action = args[0]

	// Load the stats file.
	stats := stats{
		Moves:        0,
		Wins:         map[string]int{"white": 0, "black": 0, "draw": 0},
		PreviousGame: map[string]string{"winner": "N/A", "method": "N/A"},
	}
	data, _ := os.ReadFile("./state/stats.yml")
	err := yaml.Unmarshal(data, &stats)
	check(err)

	// Initialize the game form the pgn file.
	game := initializeGame(pgnFilePath)

	if action == "move" {
		userMove = strings.ToLower(args[1])

		// Check if user move is valid. Handle accordingly.
		if userMove == "" {
			fmt.Println("No move provided.")
			return
		} else {
			fmt.Println(game.ValidMoves())
			err := game.MoveStr(userMove)
			check(err)

			stats.Moves++
			os.WriteFile(pgnFilePath, []byte(game.String()), 0755)
		}
	}

	if action == "reset" {

		// Check if someone won the game.

		switch game.Outcome() {
		case chess.WhiteWon:
			stats.Wins["white"]++
			stats.PreviousGame["winner"] = "White"
			stats.PreviousGame["method"] = game.Method().String()
		case chess.BlackWon:
			stats.Wins["black"]++
			stats.PreviousGame["winner"] = "Black"
			stats.PreviousGame["method"] = game.Method().String()
		case chess.Draw:
			stats.Wins["draw"]++
			stats.PreviousGame["winner"] = "N/A"
			stats.PreviousGame["method"] = game.Method().String()
		}

		// Reset the game by removing the pgn file and initialize a new game.
		// We create a new game so we can render the board. valid moves, and new pgn file.
		err := os.Remove(pgnFilePath)
		check(err)
		game = initializeGame(pgnFilePath)
	}

	// Render the board image.
	writeBoardImage(boardFilePath, *game.Position().Board())

	// Render the readme.
	readmeFile, err := os.OpenFile(readmeFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	check(err)
	defer readmeFile.Close()
	renderReadMe(game, readmeFile, boardFilePath, stats)

	// Render the stats.
	writeStatsFile(gameStats, stats)
}
