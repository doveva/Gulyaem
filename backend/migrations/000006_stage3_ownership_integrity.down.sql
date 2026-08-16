ALTER TABLE walks
    DROP CONSTRAINT walks_route_owner_fk,
    ADD CONSTRAINT walks_route_id_fkey
        FOREIGN KEY (route_id)
        REFERENCES routes (id);

ALTER TABLE routes
    DROP CONSTRAINT routes_id_actor_city_unique;
