# ZealDefense AI Sentinel — demo harness entry points (thin wrappers over scripts/).
# On Windows without make, call the scripts directly: bash scripts/demo-up.sh etc.

.PHONY: demo-up demo-seed demo-reset demo-status demo-down

demo-up:        ## build (if needed) -> up -> wait -> seed
	bash scripts/demo-up.sh

demo-seed:      ## run all scenarios against the running stack
	bash scripts/demo-seed.sh

demo-reset:     ## wipe volumes, restart, re-seed (between talks)
	bash scripts/demo-reset.sh

demo-status:    ## health + seeded-data counts + dashboard URLs
	bash scripts/demo-status.sh

demo-down:      ## stop the stack (keep volumes)
	cd deploy/compose && docker compose down
