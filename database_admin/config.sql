-- Log statements which take more than 2s
ALTER DATABASE patchman SET log_min_duration_statement = 2000;