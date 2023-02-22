// nolint:lll
package controllers

import "time"

type SystemsMetaTagTotal struct {
	MetaTotalHelper
	// Just helper field to get tags from db in plain string, then parsed to "Tags" attr., excluded from output data.
	TagsStr string `json:"-" csv:"-" query:"ih.tags" gorm:"column:tags_str"`
}

type MetaTotalHelper struct {
	// a helper to get total number of systems
	Total int `json:"-" csv:"-" query:"count(*) over ()" gorm:"column:total"`
}

type OSAttributes struct {
	OS   string `json:"os" csv:"os" query:"ih.system_profile->'operating_system'->>'name' || ' ' || coalesce(ih.system_profile->'operating_system'->>'major' || '.' || (ih.system_profile->'operating_system'->>'minor'), '')" order_query:"ih.system_profile->'operating_system'->>'name',cast(substring(ih.system_profile->'operating_system'->>'major','^\\d+') as int),cast(substring(ih.system_profile->'operating_system'->>'minor','^\\d+') as int)" gorm:"column:os"`
	Rhsm string `json:"rhsm" csv:"rhsm" query:"ih.system_profile->'rhsm'->>'version'" gorm:"column:rhsm"`
}

type SystemTimestamps struct {
	StaleTimestamp        *time.Time `json:"stale_timestamp" csv:"stale_timestamp" query:"ih.stale_timestamp" gorm:"column:stale_timestamp"`
	StaleWarningTimestamp *time.Time `json:"stale_warning_timestamp" csv:"stale_warning_timestamp" query:"ih.stale_warning_timestamp" gorm:"column:stale_warning_timestamp"`
	CulledTimestamp       *time.Time `json:"culled_timestamp" csv:"culled_timestamp" query:"ih.culled_timestamp" gorm:"column:culled_timestamp"`
	Created               *time.Time `json:"created" csv:"created" query:"ih.created" gorm:"column:created"`
}

type SystemTags struct {
	Tags SystemTagsList `json:"tags" csv:"tags" gorm:"-"`
}

type BaselineAttributes struct {
	BaselineNameAttr
	BaselineUpToDate *bool `json:"baseline_uptodate" csv:"baseline_uptodate" query:"sp.baseline_uptodate" gorm:"column:baseline_uptodate"`
}

type BaselineNameAttr struct {
	BaselineName string `json:"baseline_name" csv:"baseline_name" query:"bl.name" gorm:"column:baseline_name"`
}

type SystemDisplayName struct {
	DisplayName string `json:"display_name" csv:"display_name" query:"sp.display_name" gorm:"column:display_name"`
}

type SystemLastUpload struct {
	LastUpload *time.Time `json:"last_upload" csv:"last_upload" query:"sp.last_upload" gorm:"column:last_upload"`
}

type SystemStale struct {
	Stale bool `json:"stale" csv:"stale" query:"sp.stale" gorm:"column:stale"`
}

type SystemIDAttribute struct {
	ID string `json:"id" csv:"id" query:"sp.inventory_id" gorm:"column:id"`
}

type SystemAdvisoryStatus struct {
	Status string `json:"status" csv:"status" query:"st.name" gorm:"column:name"`
}
