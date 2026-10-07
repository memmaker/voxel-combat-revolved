package server

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/memmaker/battleground/game"
)

func TestMain(m *testing.M) {
	// assets are loaded relative to the repository root
	if err := os.Chdir(".."); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestContentLoads(t *testing.T) {
	content, err := game.LoadContent()
	if err != nil {
		t.Fatal(err)
	}
	if len(content.Factions) < 2 || len(content.Weapons) == 0 || len(content.Items) == 0 {
		t.Fatalf("content looks incomplete: %d factions, %d weapons, %d items", len(content.Factions), len(content.Weapons), len(content.Items))
	}
}

type matchResult struct {
	winner   uint64
	duration time.Duration
}

// playAIMatch runs a complete game between two AI players on an in-memory server.
func playAIMatch(t *testing.T, gameID string, mapName string, mission *game.MissionDetails, difficultyA, difficultyB game.AIDifficulty) matchResult {
	t.Helper()
	battleServer, err := NewBattleServerFromAssets()
	if err != nil {
		t.Fatal(err)
	}
	game.Dial = battleServer.DialMem

	results := make(chan game.GameOverMessage, 2)
	start := time.Now()

	creator := game.NewDummyClient("mem")
	creator.SetDifficulty(difficultyA)
	creator.SetOnGameOver(func(msg game.GameOverMessage) { results <- msg })
	setupA := game.DefaultAISetup()
	setupA.GameID = gameID
	setupA.Mission = mission
	setupA.MapFile = mapName
	creator.RunSetupSequence(setupA)

	joiner := game.NewDummyClient("mem")
	joiner.SetDifficulty(difficultyB)
	joiner.SetOnGameOver(func(msg game.GameOverMessage) { results <- msg })
	joiner.RunSetupSequence(game.AIGameSetup{
		Username:    "joiner",
		GameID:      gameID,
		FactionName: "Deep Ones",
		Units: []game.UnitChoice{
			{UnitTypeID: 11, Name: "Gnarg", Weapon: "M1911 Pistol", Items: []string{"Frag Grenade"}},
			{UnitTypeID: 10, Name: "Gorn", Weapon: "M16 Rifle", Items: []string{"Frag Grenade"}},
			{UnitTypeID: 8, Name: "Grimbel", Weapon: "Mossberg 500", Items: []string{"Smoke Grenade"}},
		},
	})

	select {
	case msg := <-results:
		return matchResult{winner: msg.WinnerID, duration: time.Since(start)}
	case <-time.After(90 * time.Second):
		t.Fatalf("match %s did not finish within 90 seconds", gameID)
	}
	return matchResult{}
}

func TestTurnTimerEndsIdleTurns(t *testing.T) {
	battleServer, err := NewBattleServerFromAssets()
	if err != nil {
		t.Fatal(err)
	}
	game.Dial = battleServer.DialMem
	results := make(chan game.GameOverMessage, 2)

	mission := game.NewRandomDeathmatch()
	mission.TurnSeconds = 1
	active := game.NewDummyClient("mem")
	active.SetOnGameOver(func(msg game.GameOverMessage) { results <- msg })
	setup := game.DefaultAISetup()
	setup.GameID = "timer-test"
	setup.Mission = mission
	setup.MapFile = "gen:outpost:1"
	active.RunSetupSequence(setup)

	idle := game.NewDummyClient("mem")
	idle.SetPassive(true)
	idle.RunSetupSequence(game.AIGameSetup{
		Username: "idle", GameID: "timer-test", FactionName: "Deep Ones",
		Units: []game.UnitChoice{{UnitTypeID: 2, Name: "Sleepy", Weapon: "M1911 Pistol"}},
	})
	select {
	case msg := <-results:
		if msg.WinnerID == 0 {
			t.Fatal("expected a winner")
		}
	case <-time.After(150 * time.Second):
		t.Fatal("the idle player's turns were never ended by the timer")
	}
}

func TestPlayerCanRejoinAfterDisconnect(t *testing.T) {
	battleServer, err := NewBattleServerFromAssets()
	if err != nil {
		t.Fatal(err)
	}
	game.Dial = battleServer.DialMem
	results := make(chan game.GameOverMessage, 3)

	host := game.NewDummyClient("mem")
	host.SetOnGameOver(func(msg game.GameOverMessage) { results <- msg })
	setup := game.DefaultAISetup()
	setup.GameID = "rejoin-test"
	setup.MapFile = "gen:urban:2"
	host.RunSetupSequence(setup)

	guestSetup := game.AIGameSetup{
		Username: "guest", GameID: "rejoin-test", FactionName: "Deep Ones",
		Units: []game.UnitChoice{
			{UnitTypeID: 2, Name: "Gnarg", Weapon: "M16 Rifle", Items: []string{"Frag Grenade"}},
			{UnitTypeID: 9, Name: "Gorn", Weapon: "Steyr SSG 69"},
		},
	}
	guest := game.NewDummyClient("mem")
	guest.RunSetupSequence(guestSetup)

	time.Sleep(40 * time.Millisecond) // the first turns are under way
	select {
	case <-results:
		t.Skip("the match ended before the disconnect could be tested")
	default:
	}
	guest.Close()
	time.Sleep(200 * time.Millisecond)

	rejoined := game.NewDummyClient("mem")
	rejoined.SetOnGameOver(func(msg game.GameOverMessage) { results <- msg })
	rejoined.Join("guest", "rejoin-test")

	select {
	case <-results:
		if !rejoined.Rejoined() {
			t.Fatal("the match ended, but the guest never got its seat back")
		}
	case <-time.After(90 * time.Second):
		t.Fatal("the match did not continue after the rejoin")
	}
}

func TestAIMatchFinishes(t *testing.T) {
	if testing.Short() {
		t.Skip("plays full matches")
	}
	matches := 3
	if v := os.Getenv("BT_MATCHES"); v != "" {
		fmt.Sscanf(v, "%d", &matches)
	}
	wins := map[uint64]int{}
	for i := 0; i < matches; i++ {
		mapName := []string{"map", "gen:urban:%d", "gen:ruins:%d", "gen:outpost:%d"}[i%4]
		if strings.Contains(mapName, "%d") {
			mapName = fmt.Sprintf(mapName, i)
		}
		result := playAIMatch(t, fmt.Sprintf("ai-match-%d", i), mapName, game.NewRandomDeathmatch(), game.AINormal, game.AINormal)
		wins[result.winner]++
		t.Logf("match %d on %s: winner %d after %s", i, mapName, result.winner, result.duration.Round(time.Millisecond))
	}
	t.Logf("wins per player: %v", wins)
}
