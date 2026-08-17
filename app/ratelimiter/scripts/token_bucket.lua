-- Lua isn't part of redis, it's a real language Redis just embeds
-- Lua is a scripting language, designed to be embedded inside other programs rather than run standalone.
-- Why Lua matters in this case: when you send a Lua script to Redis via EVAL, Redis runs that entire script inside its own single-threaded event loop, start to finish, with no other command allowed to run in between. (ATOMICITY)
-- READ, DECIDE, WRITE as one indivisble unit.
-- Think of it as Lua is the vehicle that delivers the Atomicity in Redis
-- KEYS AND ARGV, both arrays redis populates for you when it invokes the script.
-- KEYS holds actual Redis key names, kept separate from other arguments because Redis needs to know which keys a script touches for routing purposes.
-- ARGV holds everything else, plain values like capacity or timestamp.
-- - Everything crossing the boundary arrives as a string, which is why you see tonumber() calls converting them before doing math.


-- KEYS[1] = redis key for this client's bucket
-- ARGV[1] = capacity (max tokens)
-- ARGV[2] =  refill_rate (tokens/second)
-- ARGV[3] = now (unix,timestamp,seconds as float)

local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
-- pull the client's bucket key and 3 other numeric inputs, converting from string to number since Lua arithmetic needs actual numbers.


local bucket = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])
-- HMGET reads two fields off a redis hash in one call, the current token count and when it was last touched, if those keys don't exist yet, Redis returns nil for missing fields

if tokens == nil then
    -- this here is meant to handle a brand new client; no bucket exists, so start it off completely full and stamp "last touched" as right now.
    tokens = capacity
    last_refill = now
end

-- elapsed is how many seconds passed since this bucket was last checked
-- math.max guards against -ve numbers if timestamps arrive slightly out of order.
-- second line of tokens is the actual refill math "lazy refill" 
local elapsed = math.max(0, now - last_refill)
tokens = math.min(capacity, tokens + elapsed * refill_rate)

local allowed = 0
if tokens >=1 then
    tokens = tokens -1
    allowed = 1
end

-- write the updated state back, new token count, new "last_checked" timestamp. This is the act - "half of chec then act" and because its happening inside the same atomic script as the read above, there's no window for another concurrent request to sneak in btn them, this line is the actual fix for the race condition, the naive redis has.
redis.call("HMSET", key, "tokens", tokens, "last_refill", now)
redis.call("EXPIRE", key, math.ceil(capacity / refill_rate) + 1)
-- set a TTl so redis cleans up the key automatically once its no longer meaningful.
-- capacity/refill_rate is exactly how long it'd take an empty bucket to refill all the way back to full, so once that much time passes with no activity, an idle client's bucket would be full again anyway, nothing useful is lost by letting redis garbage-collect it.

return {allowed, math.floor(tokens)}
-- sends allowed, remaining back to go, give the client round numbers