-- KEYS[1] = redis key for this client's bucket
-- ARGV[1] = capacity (max tokens)
-- ARGV[2] =  refill_rate (tokens/second)
-- ARGV[3] = now (unix,timestamp,seconds as float)

local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local bucket = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

if tokens == nil then
    tokens = capcity
    last_refill = now
end

local elapsed = math.max(0, now - last_refill)
tokens = math.min(capacity, token + elapsed * refill_rate)

local allowed = 0
if tokens >=1 then
    tokens = tokens -1
    allowed = 1
end

redis.call("HMSET", key, "tokens", tokens, "last_refill", now)
redis.call("EXPIRE", key, math.ceil(capacity / refill_rate) + 1)

return {allowed, math.floor(tokens)}