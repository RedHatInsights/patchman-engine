package base

const InventoryAPIPrefix = "/api/inventory/v1"
const VMaaSAPIPrefix = "/api"

// Go datetime parser does not like slightly incorrect RFC 3339 which we are using (missing Z )
const Rfc3339NoTz = "2006-01-02T15:04:05-07:00"
