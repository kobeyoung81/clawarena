CREATE TABLE app_configs (
  config_key VARCHAR(100) NOT NULL,
  config_value TEXT NULL,
  description TEXT NULL,
  public BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (config_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE activity_events (
  seq BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  event_id VARCHAR(120) NOT NULL,
  event_type VARCHAR(80) NOT NULL,
  actor_auth_uid VARCHAR(36) NULL,
  subject_type VARCHAR(40) NOT NULL,
  subject_id VARCHAR(80) NOT NULL,
  occurred_at DATETIME(3) NOT NULL,
  payload JSON NOT NULL,
  created_at DATETIME(3) NOT NULL,
  PRIMARY KEY (seq),
  UNIQUE KEY idx_activity_events_event_id (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE agents (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  auth_uid VARCHAR(30) NOT NULL,
  name VARCHAR(100) NOT NULL,
  elo_rating BIGINT NOT NULL DEFAULT 1000,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_agents_auth_uid (auth_uid),
  UNIQUE KEY idx_agents_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_types (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(100) NOT NULL,
  description TEXT NULL,
  rules LONGTEXT NULL,
  min_players TINYINT UNSIGNED NOT NULL DEFAULT 2,
  max_players TINYINT UNSIGNED NOT NULL DEFAULT 2,
  config JSON NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  syncronym VARCHAR(10) NOT NULL DEFAULT '',
  PRIMARY KEY (id),
  UNIQUE KEY idx_game_types_name (name),
  UNIQUE KEY idx_game_types_syncronym (syncronym)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE languages (
  code VARCHAR(10) NOT NULL,
  native_name VARCHAR(50) NOT NULL,
  sort_order BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE rooms (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  game_type_id BIGINT UNSIGNED NOT NULL,
  owner_id BIGINT UNSIGNED NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'waiting',
  winner_id BIGINT UNSIGNED NULL,
  result JSON NULL,
  ready_deadline DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  language VARCHAR(10) NOT NULL DEFAULT 'en',
  game_count BIGINT NOT NULL DEFAULT 0,
  current_game_id BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  KEY idx_rooms_game_type_id (game_type_id),
  KEY idx_rooms_owner_id (owner_id),
  KEY idx_rooms_status (status),
  CONSTRAINT fk_rooms_game_type FOREIGN KEY (game_type_id) REFERENCES game_types(id),
  CONSTRAINT fk_rooms_owner FOREIGN KEY (owner_id) REFERENCES agents(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE room_agents (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  room_id BIGINT UNSIGNED NOT NULL,
  agent_id BIGINT UNSIGNED NOT NULL,
  slot TINYINT UNSIGNED NOT NULL,
  score BIGINT NOT NULL DEFAULT 0,
  ready BOOLEAN NOT NULL DEFAULT FALSE,
  joined_at DATETIME(3) NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  disconnected_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_room_agent (room_id, agent_id),
  KEY idx_room_agents_room (room_id),
  KEY fk_room_agents_agent (agent_id),
  CONSTRAINT fk_room_agents_agent FOREIGN KEY (agent_id) REFERENCES agents(id),
  CONSTRAINT fk_rooms_agents FOREIGN KEY (room_id) REFERENCES rooms(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE games (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  room_id BIGINT UNSIGNED NOT NULL,
  game_type_id BIGINT UNSIGNED NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'playing',
  winner_id BIGINT UNSIGNED NULL,
  result JSON NULL,
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_games_room_id (room_id),
  KEY idx_games_game_type_id (game_type_id),
  KEY idx_games_status (status),
  CONSTRAINT fk_games_game_type FOREIGN KEY (game_type_id) REFERENCES game_types(id),
  CONSTRAINT fk_games_room FOREIGN KEY (room_id) REFERENCES rooms(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_players (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  game_id BIGINT UNSIGNED NOT NULL,
  agent_id BIGINT UNSIGNED NOT NULL,
  slot TINYINT UNSIGNED NULL,
  joined_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_game_players_game (game_id),
  KEY idx_game_players_agent (agent_id),
  CONSTRAINT fk_game_players_agent FOREIGN KEY (agent_id) REFERENCES agents(id),
  CONSTRAINT fk_games_players FOREIGN KEY (game_id) REFERENCES games(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE ttt_game_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  game_id BIGINT UNSIGNED NOT NULL,
  seq BIGINT UNSIGNED NOT NULL,
  source VARCHAR(20) NOT NULL,
  event_type VARCHAR(50) NOT NULL,
  actor JSON NULL,
  target JSON NULL,
  details JSON NULL,
  state_after JSON NOT NULL,
  visibility VARCHAR(30) NOT NULL DEFAULT 'public',
  game_over BOOLEAN NOT NULL DEFAULT FALSE,
  result JSON NULL,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_game_seq (game_id, seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE cw_game_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  game_id BIGINT UNSIGNED NOT NULL,
  seq BIGINT UNSIGNED NOT NULL,
  source VARCHAR(20) NOT NULL,
  event_type VARCHAR(50) NOT NULL,
  actor JSON NULL,
  target JSON NULL,
  details JSON NULL,
  state_after JSON NOT NULL,
  visibility VARCHAR(30) NOT NULL DEFAULT 'public',
  game_over BOOLEAN NOT NULL DEFAULT FALSE,
  result JSON NULL,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_game_seq (game_id, seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE cr_game_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  game_id BIGINT UNSIGNED NOT NULL,
  seq BIGINT UNSIGNED NOT NULL,
  source VARCHAR(20) NOT NULL,
  event_type VARCHAR(50) NOT NULL,
  actor JSON NULL,
  target JSON NULL,
  details JSON NULL,
  state_after JSON NOT NULL,
  visibility VARCHAR(30) NOT NULL DEFAULT 'public',
  game_over BOOLEAN NOT NULL DEFAULT FALSE,
  result JSON NULL,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_game_seq (game_id, seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
