ALTER TABLE routes
    ADD CONSTRAINT routes_id_actor_city_unique
    UNIQUE (id, actor_id, city_id);

ALTER TABLE walks
    DROP CONSTRAINT walks_route_id_fkey,
    ADD CONSTRAINT walks_route_owner_fk
        FOREIGN KEY (route_id, actor_id, city_id)
        REFERENCES routes (id, actor_id, city_id);
