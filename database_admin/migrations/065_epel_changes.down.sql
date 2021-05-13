alter table advisory_metadata
    add constraint advisory_metadata_solution_check
        check (NOT empty(solution));

