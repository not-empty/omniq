-- PAUSE QUEUE
-- Creates a pause flag. Does not move jobs.

-- ARGV:
-- 1 base

local base = ARGV[1]
local k_paused = base .. ":paused"

redis.call("SET", k_paused, "1")

return "OK"
