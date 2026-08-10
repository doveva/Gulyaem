# Stage 1.6 routing fixtures

This fixture compares Valhalla, GraphHopper and OSRM using the same local PBF and the same five
Stage 1.5 reference routes.

`cases.json` stores explicit indexes into the reference LineStrings. Each route request therefore
contains only two to four semantic waypoints instead of the full reference geometry. The engines
remain free to choose their own paths between those points.

The ordinary Admiralteyskaya trace and the intentionally ambiguous Capella trace are also used for
native map-matching smoke tests. Engine graph IDs are diagnostic only and never become Gulyaem
StreetSegment IDs.
