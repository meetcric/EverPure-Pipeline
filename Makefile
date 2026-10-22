.PHONY: seed

seed:
	docker compose exec -T db psql -U user -d pulse -v ON_ERROR_STOP=1 \
		-v data="$$(cat seed/seed.json)" < scripts/seed.sql
