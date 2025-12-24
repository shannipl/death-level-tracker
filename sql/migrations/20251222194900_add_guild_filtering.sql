-- Add tibia_guilds column to guild_configs table
ALTER TABLE guild_configs ADD COLUMN IF NOT EXISTS tibia_guilds TEXT[] DEFAULT NULL;
