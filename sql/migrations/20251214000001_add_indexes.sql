-- =============================================================================
-- Migration: Add Performance Indexes
-- Description: Creates indexes for frequently used queries to optimize performance
-- =============================================================================

-- =============================================================================
-- PLAYERS TABLE INDEXES
-- =============================================================================

-- Index for GetPlayersLevels query: SELECT name, level FROM players WHERE world = $1
-- Most frequently used - called every tracking interval for each world
CREATE INDEX IF NOT EXISTS idx_players_world ON players (world);

-- Index for DeleteOldPlayers query: DELETE FROM players WHERE world = $1 AND updated_at < threshold
-- Composite index for world + updated_at for efficient cleanup queries
CREATE INDEX IF NOT EXISTS idx_players_world_updated_at ON players (world, updated_at);

-- Index for BatchTouchPlayers: UPDATE players SET updated_at = NOW() WHERE name = ANY(@names)
-- Primary key on name already covers this, no additional index needed

-- =============================================================================
-- GUILD_CONFIGS TABLE INDEXES
-- =============================================================================

-- Index for queries filtering by world (if needed in future)
CREATE INDEX IF NOT EXISTS idx_guild_configs_world ON guild_configs (world);