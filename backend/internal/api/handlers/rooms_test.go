package handlers

import (
	"testing"

	"github.com/clawarena/clawarena/internal/models"
)

func TestRemainingRoomAgentsAfterForfeit_MultiplayerTerminalLeave(t *testing.T) {
	agents := []models.RoomAgent{
		{AgentID: 1, Status: models.RoomAgentKIA},
		{AgentID: 2, Status: models.RoomAgentKIA},
		{AgentID: 3, Status: models.RoomAgentKIA},
		{AgentID: 4, Status: models.RoomAgentKIA},
		{AgentID: 5, Status: models.RoomAgentActive},
		{AgentID: 6, Status: models.RoomAgentActive},
	}

	remaining := remainingRoomAgentsAfterForfeit(agents, 5)
	if len(remaining) != 1 {
		t.Fatalf("expected one remaining agent after terminal multiplayer leave, got %d", len(remaining))
	}
	if remaining[0].AgentID != 6 {
		t.Fatalf("expected agent 6 to remain, got %d", remaining[0].AgentID)
	}
}

func TestRemainingRoomAgentsAfterForfeit_ExcludesOnlyKIAAndForfeiter(t *testing.T) {
	agents := []models.RoomAgent{
		{AgentID: 10, Status: models.RoomAgentActive},
		{AgentID: 11, Status: models.RoomAgentDisconnected},
		{AgentID: 12, Status: models.RoomAgentKIA},
	}

	remaining := remainingRoomAgentsAfterForfeit(agents, 10)
	if len(remaining) != 1 {
		t.Fatalf("expected disconnected non-KIA agent to remain eligible, got %d agents", len(remaining))
	}
	if remaining[0].AgentID != 11 {
		t.Fatalf("expected agent 11 to remain, got %d", remaining[0].AgentID)
	}
}
