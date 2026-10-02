# UC-IM-02 — Delete a tracked item

**Primary actor:** Operator
**Supporting system:** PriceFollower API
**Trigger:** The operator chooses **Delete** from the item row or its details page.

## Preconditions

- The item exists in the tracked collection.

## Main flow

1. The page opens a confirmation dialog identifying the item.
2. The operator confirms deletion.
3. The page submits `DELETE /api/v1/items/{id}`.
4. The server deletes the item and all associated item-price and second-hand-offer history.
5. The page removes the item from the list. If deletion was initiated from the details page, the page returns to the tracked-items list.

## Alternatives and errors

- If the operator cancels, no request is sent and the item remains tracked.
- If deletion fails, the page shows an error and keeps the item visible.

## Postconditions

- On success, the item and its retained history are removed. Deleted-item history is not retained.
