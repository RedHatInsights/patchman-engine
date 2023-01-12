CREATE OR REPLACE FUNCTION hash_partition_id(id int, parts int)
    RETURNS int AS
$$
    BEGIN
        -- src/include/common/hashfn.h:83
        --  a ^= b + UINT64CONST(0x49a0f4dd15e5a8e3) + (a << 54) + (a >> 7);
        -- => 8816678312871386365
        -- src/include/catalog/partition.h:20
        --  #define HASH_PARTITION_SEED UINT64CONST(0x7A5B22367996DCFD)
        -- => 5305509591434766563
        RETURN (((hashint4extended(id, 8816678312871386365)::numeric + 5305509591434766563) % parts + parts)::int % parts);
    END;
$$ LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE;
