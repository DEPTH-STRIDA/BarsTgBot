.PHONY: postgres-up postgres-down postgres-restart postgres-redeploy barbot-up barbot-down barbot-restart barbot-redeployd

postgres-up:
	- docker network create barBot_network
	docker compose -f ./deployments/postgres/docker-compose.yml up -d

postgres-down:
	docker compose -f ./deployments/postgres/docker-compose.yml down

postgres-restart:
	docker compose -f ./deployments/postgres/docker-compose.yml restart

postgres-redeploy: postgres-down
	docker compose -f ./deployments/postgres/docker-compose.yml pull
	docker compose -f ./deployments/postgres/docker-compose.yml up -d --force-recreate

barbot-up:
	- docker network create barBot_network
	docker compose -f ./deployments/barBot/docker-compose.yml up -d

barbot-down:
	docker compose -f ./deployments/barBot/docker-compose.yml down

barbot-restart:
	docker compose -f ./deployments/barBot/docker-compose.yml restart

barbot-redeploy: barbot-down
	docker compose -f ./deployments/barBot/docker-compose.yml pull
	docker compose -f ./deployments/barBot/docker-compose.yml up -d --force-recreate --build