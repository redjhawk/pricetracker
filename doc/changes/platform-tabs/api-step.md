# API contract preservation

Status: ready; unchanged approved contract. Date: 2026-10-03.

After completing the [technical specification](../../specifications/platform-tabs/technical.md), inspected [API_SPECIFICATION.md](../../../API_SPECIFICATION.md). Its latest listed approval is consecutive item price periods on 2026-10-02; the platform discriminator and nullable LeBoncoin second-hand offer were approved on 2026-09-26.

All required data already exists in `GET /api/v1/items`. Tabs filter its existing global result locally. Global refresh remains `POST /api/v1/items/refresh` without a body, retaining `202`, `requestedAt`, `itemsQueued`, and existing pending/polling semantics. Detail, add, delete, per-item refresh, response fields, status codes and error envelopes are unchanged.

No canonical contract edit or backend change is proposed. Repository workflow stage 3 explicitly permits proceeding with a frontend-only change preserving an approved contract; no renewed user confirmation is required. This record does not assert a new approval. Any newly discovered required contract change must return to the API proposal/confirmation stage before implementation.
