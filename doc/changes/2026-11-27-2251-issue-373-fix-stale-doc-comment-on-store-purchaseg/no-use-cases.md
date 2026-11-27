# No use cases for issue #373

This change only rewrites the doc comment of `Store.PurchaseGoals` in `internal/store/store.go` so it says the method returns the goals of the owner's tracked LeBoncoin and Amazon items (FR-GOAL-CHOICE-017). The method's signature, query and behaviour do not change (FR-GOAL-CHOICE-018, -019), and the operator observes no change in the application. FR-GOAL-CHOICE-017 to 019 have no user-visible behaviour, so no use cases are needed; the existing purchase goal choice use cases still apply unchanged.
