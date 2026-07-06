package tcp

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func resetHomeStatsForTest(t *testing.T) {
	t.Helper()

	homeStatsMu.Lock()
	homeStatsConfig = homeStatsConfigState{}
	homeStatsHistory = homeStatsHistoryState{}
	homeStatsMu.Unlock()

	t.Cleanup(func() {
		homeStatsMu.Lock()
		homeStatsConfig = homeStatsConfigState{}
		homeStatsHistory = homeStatsHistoryState{}
		homeStatsMu.Unlock()
	})
}

func updateHomeStatsForTest(at time.Time, aggregate homeStatsAggregate) {
	homeStatsMu.Lock()
	updateHomeStatsHistoryLocked(at, aggregate, 0, 0)
	homeStatsMu.Unlock()
}

func snapshotHomeStatsHistoryForTest() homeStatsHistoryState {
	homeStatsMu.Lock()
	defer homeStatsMu.Unlock()
	return homeStatsHistory
}

func TestWebSocketConnectDoesNotResetHomeStatsEfficiencyWindow(t *testing.T) {
	resetHomeStatsForTest(t)

	start := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	updateHomeStatsForTest(start, homeStatsAggregate{
		TotalCount:     100,
		TotalCupNum:    100,
		TotalExitCount: 80,
		TotalWeight:    100 * homeStatsWeightScale,
		HasStats:       true,
	})
	updateHomeStatsForTest(start.Add(homeStatsHistoryInterval), homeStatsAggregate{
		TotalCount:     200,
		TotalCupNum:    200,
		TotalExitCount: 170,
		TotalWeight:    200 * homeStatsWeightScale,
		HasStats:       true,
	})

	before := snapshotHomeStatsHistoryForTest()
	if before.CupFillEfficiency != 90 {
		t.Fatalf("setup CupFillEfficiency = %.1f, want 90.0", before.CupFillEfficiency)
	}
	if !before.HasPrev {
		t.Fatalf("setup HasPrev = false, want true")
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/ws/data", handleWebSocketData)
	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/data"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	_, _, _ = conn.ReadMessage()
	_ = conn.Close()

	after := snapshotHomeStatsHistoryForTest()
	if !after.HasPrev {
		t.Fatalf("HasPrev = false after websocket connect, want true")
	}
	if after.CupFillEfficiency != before.CupFillEfficiency {
		t.Fatalf("CupFillEfficiency = %.1f after websocket connect, want %.1f", after.CupFillEfficiency, before.CupFillEfficiency)
	}
	if !after.PrevAt.Equal(before.PrevAt) {
		t.Fatalf("PrevAt = %s after websocket connect, want %s", after.PrevAt, before.PrevAt)
	}
}

func TestHomeStatsRealtimeOutputUsesActualElapsedWindow(t *testing.T) {
	resetHomeStatsForTest(t)

	start := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	updateHomeStatsForTest(start, homeStatsAggregate{
		TotalCount:     100,
		TotalCupNum:    100,
		TotalExitCount: 80,
		TotalWeight:    100 * homeStatsWeightScale,
		HasStats:       true,
	})
	updateHomeStatsForTest(start.Add(30*time.Second), homeStatsAggregate{
		TotalCount:     200,
		TotalCupNum:    200,
		TotalExitCount: 170,
		TotalWeight:    110 * homeStatsWeightScale,
		HasStats:       true,
	})

	history := snapshotHomeStatsHistoryForTest()
	const want = 1200.0
	if history.RealtimeOutputTonPerHour != want {
		t.Fatalf("RealtimeOutputTonPerHour = %.1f, want %.1f", history.RealtimeOutputTonPerHour, want)
	}
}
