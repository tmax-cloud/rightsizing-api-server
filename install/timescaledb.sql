CREATE EXTENSION timescaledb_toolkit;

create materialized view if not exists ":container_cpu_usage:10min"
with (timescaledb.continuous) as 
select time_bucket('10m', time) as bucket, 
series_id, 
approx_percentile(0.9, percentile_agg(value)) as value 
from prom_data."container:container_cpu_usage:rate"
group by bucket, series_id;

SELECT
add_continuous_aggregate_policy(':container_cpu_usage:10min', 
start_offset => INTERVAL '1h',  
end_offset => INTERVAL '10m', 
schedule_interval => INTERVAL '10m');

create materialized view if not exists ":container_memory_working_set_bytes:10min" 
with (timescaledb.continuous) as 
select time_bucket('10m', time) as bucket, 
series_id, 
approx_percentile(0.9, percentile_agg(value)) as value 
from prom_data.container_memory_working_set_bytes 
group by bucket, series_id;

SELECT add_continuous_aggregate_policy(':container_memory_working_set_bytes:10min',
start_offset => INTERVAL '1h',
end_offset => INTERVAL '10m',
schedule_interval => INTERVAL '10m');
