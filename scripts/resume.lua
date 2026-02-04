-- RESUME QUEUE
-- Removes pause flag.

-- ARGV:
-- 1 base

local base = ARGV[1]
local k_paused = base .. ":paused"

local removed = redis.call("DEL", k_paused)

return removed
