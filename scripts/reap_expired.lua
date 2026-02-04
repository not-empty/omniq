-- REAP_EXPIRED (stalled recovery)
-- ARGV:
-- 1 base
-- 2 now_ms
-- 3 max_reap

local base = ARGV[1]
local now_ms = tonumber(ARGV[2] or "0")
local max_reap = tonumber(ARGV[3] or "1000")

local DEFAULT_GROUP_LIMIT = 1

local k_active  = base .. ":active"
local k_delayed = base .. ":delayed"
local k_failed  = base .. ":failed"
local k_gready  = base .. ":groups:ready"

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

local ids = redis.call("ZRANGEBYSCORE", k_active, "-inf", now_ms, "LIMIT", 0, max_reap)
local reaped = 0

for i=1,#ids do
  local job_id = ids[i]

  -- re-check current score in case lease was extended after scan
  local score = redis.call("ZSCORE", k_active, job_id)
  if score and tonumber(score) and tonumber(score) > now_ms then
    -- still leased; skip
  else
    if redis.call("ZREM", k_active, job_id) == 1 then
      local k_job = base .. ":job:" .. job_id

      -- if job hash missing, nothing else to do
      if redis.call("EXISTS", k_job) == 0 then
        reaped = reaped + 1
      else
        -- IMPORTANT: invalidate attempt ownership
        redis.call("HSET", k_job, "lease_token", "")

        -- group bookkeeping (if grouped)
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

        local attempt      = to_i(redis.call("HGET", k_job, "attempt"))
        local max_attempts = to_i(redis.call("HGET", k_job, "max_attempts"))
        if max_attempts <= 0 then max_attempts = 1 end
        local backoff_ms   = to_i(redis.call("HGET", k_job, "backoff_ms"))

        if attempt >= max_attempts then
          redis.call("HSET", k_job,
            "state", "failed",
            "updated_ms", tostring(now_ms),
            "lease_token", "",
            "lock_until_ms", ""
          )
          redis.call("LPUSH", k_failed, job_id)
        else
          local due_ms = now_ms + backoff_ms
          redis.call("HSET", k_job,
            "state", "delayed",
            "due_ms", tostring(due_ms),
            "updated_ms", tostring(now_ms),
            "lease_token", "",
            "lock_until_ms", ""
          )
          redis.call("ZADD", k_delayed, due_ms, job_id)
        end

        reaped = reaped + 1
      end
    end
  end
end

return {"OK", tostring(reaped)}
