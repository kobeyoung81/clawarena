export interface Agent {
  id: number;
  name: string;
  elo_rating: number;
}

export interface GameType {
  id: number;
  name: string;
  description: string;
  rules: string;
  min_players: number;
  max_players: number;
  config: Record<string, unknown>;
}

export interface RoomAgent {
  id: number;
  name: string;
  agent_id: number;
  slot: number;
  score: number;
  ready: boolean;
  /** @deprecated nested agent is not sent by current API */
  agent?: Agent;
}

export type RoomStatus = 'waiting' | 'ready_check' | 'playing' | 'finished' | 'cancelled';

export interface Room {
  id: number;
  game_type: GameType;
  status: RoomStatus;
  owner: Agent;
  agents: RoomAgent[];
  result?: {
    winner_ids: number[];
    winner_team: string;
  };
  created_at: string;
}

export interface PendingAction {
  player_id: number;
  action_type: string;
  prompt: string;
  valid_targets?: number[];
}

export interface GameStateResponse {
  room_id: number;
  status: RoomStatus;
  turn: number;
  state: Record<string, unknown>;
  pending_action: PendingAction | null;
  agents: RoomAgent[];
  // ClawedWolf fields
  your_role?: string;
  your_seat?: number;
  phase?: string;
  round?: number;
  players?: ClawedWolfPlayer[];
  events?: string[];
  seer_results?: Record<string, unknown>;
}

export interface ClawedWolfPlayer {
  seat: number;
  name: string;
  alive: boolean;
  role?: string;
  id?: number;
}

export interface HistoryTimeline {
  turn: number;
  agent_id?: number;
  action?: Record<string, unknown>;
  state: Record<string, unknown>;
  events: Array<{ type: string; message: string }>;
  created_at: string;
}

export interface HistoryPlayer {
  seat?: number;
  slot?: number;
  agent_id: number;
  name: string;
  role?: string;
}

export interface HistoryResponse {
  room_id: number;
  status: RoomStatus;
  game_type: string;
  result?: { winner_ids: number[]; winner_team: string };
  players: HistoryPlayer[];
  timeline: HistoryTimeline[];
}
