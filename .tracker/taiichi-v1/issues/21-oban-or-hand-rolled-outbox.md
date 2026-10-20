# Oban or the hand-rolled outbox

Type: grilling
Status: open
Blocked by: 20

## Question

Given [Does Oban fit the outbox and the pollers?](20-oban-fit.md), should the server use Oban for the effects after a save, and possibly for the Ticket and Shipment checks, in place of the `outbox` table and sender settled in [An operation stopping partway](15-operation-stopping-partway.md)? If so, which settled rules change (ordering, giving up, retrying at startup, keeping finished rows, the Job object's `outbox` field), and what does [Server internals](12-server-internals.md) lose or gain?
