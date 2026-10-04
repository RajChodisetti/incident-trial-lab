# Discount rounding baseline (synthetic only)

Business contract: calculate an order's subtotal in integer cents, apply its
percentage discount once to the subtotal, and truncate fractional cents once.
For unit price 105 cents, quantity 3, and discount 10%, the correct total is
`(105 * 3 * 90) / 100 = 283` cents.

The deliberately faulty baseline discounts/truncates each unit before multiplying
quantity: `(105 * 90 / 100) * 3 = 282`. This is a seeded application defect, not a
production incident or a claim of autonomous discovery.

Workload: the three requests in `workload.json`, sequentially, once per fresh
baseline. Both control orders must satisfy the business contract. Only the
`bulk-discount` request must violate it with the known faulty value. Any missing,
malformed, unavailable, or unexpected response makes reproduction unproven and
the checker exits nonzero. HTTP 200 and readiness alone do not prove correctness.

No improvement/latency/resource-budget claim is made. Candidate comparison and
repeated workload benchmarks belong to milestone 3. CPU/memory limits below bound
the application pod, not the entire Docker VM or Kubernetes control plane.
