-- Run only while subscription_manager_id still stores owner IDs.
-- Do not run after subscription_manager_id starts storing actual subscription-manager IDs.
-- Related to: RHINENG-30223
UPDATE system_inventory
SET owner_id = subscription_manager_id
WHERE owner_id IS NULL
  AND subscription_manager_id IS NOT NULL;
