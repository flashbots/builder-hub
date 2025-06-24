This is a quick guide on setting setting up and configuring a dev BuilderHub instance.

---

## Getting started

First, use docker-compose to start the three containers:

| Service     | Ports & Notes                                                                                                          |
| ----------- | ---------------------------------------------------------------------------------------------------------------------- |
| builder-hub | **8080** (Instance API), **8081** (Admin API), **8082** (Internal API)                                                 |
| mock-proxy  | **8888** (forwards to builder-hub Instance API on port 8080, adds dummy auth headers that in prod cvm-proxy would add) |
| postgres    | **5432** (with applied migrations)                                                                                     |

See also [`docker-compose.yaml`](../docker/docker-compose.yaml) for more details.

```bash
# Switch into the 'docker' directory
cd docker

# Update and start the services
docker-compose pull
docker-compose up
```

---

## Public API

There are some public API points, most notably the `get-measurements` endpoint, which is accessible without authentication:

```bash
curl http://localhost:8080/api/l1-builder/v1/measurements | jq
```

---

## Admin API

For initial setup, use the [Admin API](https://github.com/flashbots/builder-hub?tab=readme-ov-file#admin-endpoints) (on port 8081) to:
1. Create and enable allowed measurements
2. Create a builder instance
3. Create a builder configuration
4. Enable the builder instance

```bash
# 1a. Create a new measurements entry (empty allowed measurements will allow all client measurements)
curl -v \
  --url http://localhost:8081/api/admin/v1/measurements \
  --data '{
  "measurement_id": "test1",
  "attestation_type": "test",
  "measurements": {}
}'

# 1b. Enable the new measurements
curl -v \
  --request POST \
  --url http://localhost:8081/api/admin/v1/measurements/activation/test1 \
  --data '{
  "enabled": true
}'

# 2. Create a new builder instance (with IP address 1.2.3.4, which is fixed in the mock-proxy)
curl -v \
  --url http://localhost:8081/api/admin/v1/builders \
  --data '{
  "name": "test_builder",
  "ip_address": "1.2.3.4",
  "dns_name": "foobar-v1.a.b.c",
  "network": "production"
}'

# 3. Create (and enable) a new empty builder configuration
curl -v \
  --url http://localhost:8081/api/admin/v1/builders/configuration/test_builder \
  --data '{}'

# 4. Actual configuration is stored in secrets. Put it there
curl -v \
  --url http://localhost:8081/api/admin/v1/builders/secrets/test_builder \
  --data '{
  "rbuilder": {
    "extra_data": "FooBar"
  }
}'
# 5. Enable the new builder instance
curl -v \
  --url http://localhost:8081/api/admin/v1/builders/activation/test_builder \
  --data '{
  "enabled": true
}'
```

---

## Instance API (authenticated)

Now you can use the (authenticated) Instance API, like any production Yocto TDX instance would:
1. Get own instance configuration
2. Get the list of peers
3. Register credentials

The API is authenticated through headers that are attached by the proxy, namely measurements, attestation type and IP address headers.
The mock-proxy on port 8888 [adds these headers](https://github.com/flashbots/builder-hub/blob/main/docker/mock-proxy/nginx-default.conf),
so you can use the API without any additional setup (or cvm-proxy instances).

```bash
# 1. Get the instance configuration
curl http://localhost:8888/api/l1-builder/v1/configuration | jq

# 2. Get the list of peers
curl http://localhost:8888/api/l1-builder/v1/builders | jq

# 3. Register credentials for 'rbuilder' service
curl -v \
  --url http://localhost:8888/api/l1-builder/v1/register_credentials/rbuilder \
  --data '{
  "ecdsa_pubkey_address": "0x321f3426eEc20DE1910af1CD595c4DD83BEA0BA5"
}'

# If you now call get-peers again, it will contain the newly registered address:
curl http://localhost:8888/api/l1-builder/v1/builders | jq
```
