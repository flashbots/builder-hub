#
# MEASUREMENTS SETUP
#
# [Admin API] Create a new measurement
POST http://localhost:8081/api/admin/v1/measurements
{
  "measurement_id": "test1",
  "attestation_type": "test",
  "measurements": {}
}
HTTP 200

# [Admin API] Enable the measurement
POST http://localhost:8081/api/admin/v1/measurements/activation/test1
{
  "enabled": true
}
HTTP 200

# [Internal API] Verify that the measurement is now available
GET http://localhost:8082/api/l1-builder/v1/measurements
HTTP 200
[Asserts]
jsonpath "$.[0].measurement_id" == "test1"
jsonpath "$.[0].attestation_type" == "test"

#
# BUILDER SETUP
#
# [Admin API] Create a new builder instance (with IP address 1.2.3.4, which is fixed in the mock-proxy)
POST http://localhost:8081/api/admin/v1/builders
{
  "name": "test_builder",
  "ip_address": "1.2.3.4",
  "dns_name": "foobar-v1.a.b.c",
  "network": "production"
}
HTTP 200

# [Admin API] Create (and enable) a new builder configuration
POST http://localhost:8081/api/admin/v1/builders/configuration/test_builder
{
  "rbuilder": {
    "extra_data": "FooBar"
  }
}
HTTP 200

# [Admin API] Enable the new builder instance
POST http://localhost:8081/api/admin/v1/builders/activation/test_builder
{
  "enabled": true
}
HTTP 200

# [Internal API] Get the list of peers
GET http://localhost:8080/api/internal/l1-builder/v1/builders
HTTP 200
[Asserts]
jsonpath "$.[0].ip" == "1.2.3.4"
jsonpath "$.[0].name" == "test_builder"

# [Internal API] Get the list of peers (v2)
GET http://localhost:8080/api/internal/l1-builder/v2/network/production/builders
HTTP 200
[Asserts]
jsonpath "$.[0].ip" == "1.2.3.4"
jsonpath "$.[0].name" == "test_builder"

# [Builder API] Verify that the builder can get it's configuration (NOT WORKING, TODO)
#GET http://localhost:8888/api/l1-builder/v1/configuration
#HTTP 200
#[Asserts]
#jsonpath "$.[0].dns_name" == "foobar-v1.a.b.c"
#jsonpath "$.[0].rbuilder.extra_data" == "FooBar"

#
# BUILDER APIs (via proxy)
#
# [Builder API] Get the list of peers
GET http://localhost:8888/api/l1-builder/v1/builders
HTTP 200
[Asserts]
jsonpath "$.[0].ip" == "1.2.3.4"
jsonpath "$.[0].name" == "test_builder"

# [Builder API] Register credentials for 'rbuilder' service
POST http://localhost:8888/api/l1-builder/v1/register_credentials/rbuilder
{
  "ecdsa_pubkey_address": "0x321f3426eEc20DE1910af1CD595c4DD83BEA0BA5"
}
HTTP 200

# [Builder API] Register credentials for 'orderflow_proxy' service
POST http://localhost:8888/api/l1-builder/v1/register_credentials/orderflow_proxy
{
  "ecdsa_pubkey_address": "0x321f3426eEc20DE1910af1CD595c4DD83BEA0BA5",
  "tls_cert": "abcdefghijklmnopqrstuvwxyz"
}
HTTP 200

# [Builder API] Register credentials for 'instance' service
POST http://localhost:8888/api/l1-builder/v1/register_credentials/instance
{
  "tls_cert": "1234567890"
}
HTTP 200

# [Builder API] Register credentials for custom service
POST http://localhost:8888/api/l1-builder/v1/register_credentials/foobar123
{
  "tls_cert": "1234567890",
  "ecdsa_pubkey_address": "0x321f3426eEc20DE1910af1CD595c4DD83BEA0BA5"
}
HTTP 200

# [Builder API] Get the list of peers
GET http://localhost:8888/api/l1-builder/v1/builders
HTTP 200
[Asserts]
jsonpath "$.[0].orderflow_proxy.tls_cert" == "abcdefghijklmnopqrstuvwxyz"
jsonpath "$.[0].orderflow_proxy.ecdsa_pubkey_address" == "0x321f3426eec20de1910af1cd595c4dd83bea0ba5"
jsonpath "$.[0].rbuilder.ecdsa_pubkey_address" == "0x321f3426eec20de1910af1cd595c4dd83bea0ba5"
jsonpath "$.[0].instance.tls_cert" == "1234567890"
jsonpath "$.[0].foobar123.tls_cert" == "1234567890"
jsonpath "$.[0].foobar123.ecdsa_pubkey_address" == "0x321f3426eec20de1910af1cd595c4dd83bea0ba5"
