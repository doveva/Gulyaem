DROP TABLE IF EXISTS street_segments;
DROP TABLE IF EXISTS streets;
ALTER TABLE geo_data_versions
    DROP CONSTRAINT IF EXISTS geo_data_versions_id_city_unique;
DROP TYPE IF EXISTS street_segment_classification;
