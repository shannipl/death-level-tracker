-- name: SaveGuildWorld :exec
INSERT INTO guild_configs (guild_id, world, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (guild_id) DO UPDATE
SET world = EXCLUDED.world, updated_at = NOW();

-- name: GetWorldsMap :many
SELECT guild_id, world FROM guild_configs;

-- name: GetPlayersLevels :many
SELECT name, level FROM players WHERE world = $1;

-- name: UpsertPlayerLevel :exec
INSERT INTO players (name, level, world, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (name) DO UPDATE
SET level = EXCLUDED.level, world = EXCLUDED.world, updated_at = NOW();

-- name: BatchTouchPlayers :exec
UPDATE players SET updated_at = NOW() WHERE name = ANY(@names::text[]);

-- name: DeleteOldPlayers :execresult
DELETE FROM players WHERE world = $1 AND updated_at < NOW() - @threshold::interval;

-- name: DeleteGuildConfig :exec
DELETE FROM guild_configs WHERE guild_id = $1;
