# Transactions Service

## Multi-environment setup (Portainer)

The same `docker-compose.yml` is deployed as two separate Portainer stacks with different `stack.env` values.

### Staging environment

Use `stack.env.staging` as the stack environment when creating the staging stack in Portainer.

Key differences from production:

| Variable | Production | Staging |
|---|---|---|
| `POSTGRES_CONTAINER_NAME` | `transactions_postgres` | `transactions_postgres_staging` |
| `SERVICE_CONTAINER_NAME` | `transaction-service-v2` | `transaction-service-v2-staging` |
| `POSTGRES_PORT` | `5433` | `5434` |
| `SERVICE_PORT` | `1235` | `1236` |
| `LOG_LEVEL` | `info` | `debug` |

Volumes and networks are automatically isolated by Portainer (prefixed with the stack name).

### Sync production database to staging

```bash
docker exec transactions_postgres pg_dump -U admin -d transactions --clean --if-exists | docker exec -i transactions_postgres_staging psql -U admin -d transactions
```
