# Routing spike scripts

- `prepare.sh` validates the shared PBF and creates ignored local graph directories.
- `up.sh` starts each pinned engine, waits for readiness and records relative setup/resource metrics.
- `measure.py` samples container memory while graph preparation or startup is running.
- `reset.sh` removes only generated `.routing` data so a cold graph build can be measured again.

Use the Make targets documented in the project README instead of calling these files directly.
The committed fixture contract is in `data/routing-spike/spb-stage1`; the final JSON report is
written to `frontend/public/routing-spike/comparison.json` for the debug UI.
