import http from "k6/http";
import { Counter } from "k6/metrics";
// counter is a K6 metric that just increments, and its being created via the allowed_count

const allowedCount = new Counter("allowed_count");

export const options = {
  scenarios: {
    burst: {
      // run exactly 20 iterations total, shared out across 20 virtual users, then stop
      executor: "shared-iterations",
      vus: 20,
      iterations: 20,
      maxDuration: "5s",
    },
  },
};

// the 20 concurrent requests are fighting over the same counter in redis, which is exactly the scenario that exposes the check-then-act race, two instances reading the same "current count" before either has written its update back.
export default function () {
  const res = http.post(
    // "http://localhost:8081/check/naive-redis",
    "http://localhost:8081/check/token-bucket",
    JSON.stringify({ client_key: "race-test" }),
    { headers: { "Content-Type": "application/json" } },
  );

  console.log("we hit here, and this is the response ==>>", res);
  const body = JSON.parse(res.body);
  if (body.allowed) {
    allowedCount.add(1);
  }
}
