ALTER TABLE system_platform ADD stale_timestamp TIMESTAMP WITH TIME ZONE;
ALTER TABLE system_platform ADD stale_warning_timestamp TIMESTAMP WITH TIME ZONE;
ALTER TABLE system_platform ADD culled_timestamp TIMESTAMP WITH TIME ZONE;