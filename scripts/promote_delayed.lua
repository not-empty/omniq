-- PROMOTE_DELAYED (hybrid)
-- ARGV:
-- 1 base
-- 2 now_ms
-- 3 max_promote

local base = ARGV[1]
local now_ms = tonumber(ARGV[2] or "0")
local max_promote = tonumber(ARGV[3] or "1000")

local DEFAULT_GROUP_LIMIT = 1

local k_delayed = base .. ":delayed"
local k_wait    = base .. ":wait"
local k_gready  = base .. ":groups:ready"

local function to_i(v)
  if v == false or v == nil or v == '' then return 0 end
  local n = tonumber(v)
  if n == nil then return 0 end
  return math.floor(n)
end

local function group_limit_for(gid)
  local k_glimit = base .. ":g:" .. gid .. ":limit"
  local lim = to_i(redis.call("GET", k_glimit))
  if lim <= 0 then return DEFAULT_GROUP_LIMIT end
  return lim
end

local ids = redis.call("ZRANGEBYSCORE", k_delayed, "-inf", now_ms, "LIMIT", 0, max_promote)
local promoted = 0

for i=1,#ids do
  local job_id = ids[i]
  if redis.call("ZREM", k_delayed, job_id) == 1 then
    local k_job = base .. ":job:" .. job_id
    redis.call("HSET", k_job, "state", "wait", "updated_ms", tostring(now_ms))

    local gid = redis.call("HGET", k_job, "gid")
    if gid and gid ~= "" then
      local k_gwait = base .. ":g:" .. gid .. ":wait"
      redis.call("RPUSH", k_gwait, job_id)

      -- if group has capacity, mark it as ready
      local inflight = to_i(redis.call("GET", base .. ":g:" .. gid .. ":inflight"))
      local limit = group_limit_for(gid)
      if inflight < limit then
        redis.call("ZADD", k_gready, now_ms, gid)
      end
    else
      redis.call("RPUSH", k_wait, job_id)
    end

    promoted = promoted + 1
  end
end

return {"OK", tostring(promoted)}
