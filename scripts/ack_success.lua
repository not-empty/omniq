-- ACK_SUCCESS (hybrid + retention + token-gated)
-- ARGV:
-- 1 base
-- 2 job_id
-- 3 now_ms
-- 4 lease_token

local base        = ARGV[1]
local job_id      = ARGV[2]
local now_ms      = tonumber(ARGV[3] or "0")
local lease_token = ARGV[4]

local DEFAULT_GROUP_LIMIT = 1

local k_job       = base .. ":job:" .. job_id
local k_active    = base .. ":active"
local k_completed = base .. ":completed"
local k_gready    = base .. ":groups:ready"

local KEEP_COMPLETED = 100

local function to_i(v)
  if v == false or v == nil or v == '' then return 0 end
  local n = tonumber(v)
  if n == nil then return 0 end
  return math.floor(n)
end

local function dec_floor0(key)
  local v = to_i(redis.call("DECR", key))
  if v < 0 then
    redis.call("SET", key, "0")
    return 0
  end
  return v
end

local function group_limit_for(gid)
  local k_glimit = base .. ":g:" .. gid .. ":limit"
  local lim = to_i(redis.call("GET", k_glimit))
  if lim <= 0 then return DEFAULT_GROUP_LIMIT end
  return lim
end

-- token required
if lease_token == nil or lease_token == "" then
  return {"ERR", "TOKEN_REQUIRED"}
end

-- token must match the current owner attempt
local cur_token = redis.call("HGET", k_job, "lease_token") or ""
if cur_token ~= lease_token then
  return {"ERR", "TOKEN_MISMATCH"}
end

-- must still be active (prevents double-ACK and stale workers after reaper/retry)
if redis.call("ZREM", k_active, job_id) ~= 1 then
  return {"ERR", "NOT_ACTIVE"}
end

-- mark completed + clear token
redis.call("HSET", k_job,
  "state", "completed",
  "updated_ms", tostring(now_ms),
  "lease_token", "",
  "lock_until_ms", ""
)

-- group bookkeeping (if job is grouped)
local gid = redis.call("HGET", k_job, "gid")
if gid and gid ~= "" then
  local k_ginflight = base .. ":g:" .. gid .. ":inflight"
  local inflight = dec_floor0(k_ginflight)
  local limit = group_limit_for(gid)
  local k_gwait = base .. ":g:" .. gid .. ":wait"
  if inflight < limit and to_i(redis.call("LLEN", k_gwait)) > 0 then
    redis.call("ZADD", k_gready, now_ms, gid)
  end
end

-- retention list (trim completed list + delete overflow job hashes)
redis.call("LPUSH", k_completed, job_id)
while redis.call("LLEN", k_completed) > KEEP_COMPLETED do
  local old_id = redis.call("RPOP", k_completed)
  if old_id then
    redis.call("DEL", base .. ":job:" .. old_id)
  end
end

return {"OK"}
