package base

const INVENTORY_API_PREFIX = "/api/inventory/v1"
const VMAAS_API_PREFIX = "/api"
// Go datetime parser does not like slightly incorrect RFC 3339 which we are using (missing Z )
const RFC_3339_NO_TZ = "2006-01-02T15:04:05-07:00"
