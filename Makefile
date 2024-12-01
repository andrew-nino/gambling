.PHONY:  run build

build:
	docker compose  build $(c)

up:
	docker compose  up -d $(c)

start:
	docker compose  start $(c)

down:
	docker compose  down $(c)

destroy:
	docker compose  down -v $(c)

stop:
	docker compose  stop $(c)

restart:
	docker compose  stop $(c)

	docker compose  up -d $(c)

logs:
	docker compose  logs --tail=100 -f $(c)

logs-parser:
	docker compose  logs --tail=100 -f parser

ps:
	docker compose  ps

login-parser:
	docker compose exec -it parser /bin/sh