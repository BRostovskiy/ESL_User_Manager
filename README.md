# USER MANAGER SERVICE

## Simple web/gRPC service which provide functionality to create/list/update/delete users
List of available functions:

1. Create new user
2. Get list of users (pagination and filtering are available) sorted by date created (newest first).
3. Update user
4. Delete user
5. Get API health

## Setup

The easiest way to get the server application (which relies on a postgres server) up and running is run it in the docker.
You can spin both(back and database) services up using the following command:
1. Adjust `compose/um_config.yaml` according to your needs(db user/password)
2. Adjust db user/password for `challengedb` in docker-compose.yaml
3. Run `docker-compose up --build -d`

## Basic usage

Please use this commands to perform basic communication with HTTP/gRPC service:
>**_GRPCURL:_** If you want to perform `grpcurl` operations, pls make sure it's installed in your system. See more: `https://github.com/fullstorydev/grpcurl`

>**_GRPC_HEALTH_PROBE:_** If you want to perform `grpc_health_probe` check, pls make sure it's installed in your system. See more: `https://github.com/grpc-ecosystem/grpc-health-probe?tab=readme-ov-file#installation`

1. ### Create new user
- REQUEST PAYLOAD:
```json
{
  "first_name": "User1", 
  "last_name": "Lastname1", 
  "nickname": "user1_lastname", 
  "email": "user1@gmail.com", 
  "password": "qwerty123", 
  "country": "NL"
}
```
- HTTP: 
```bash
curl -X POST -H "Content-type: application/json" -d '<PAYLOAD>' http://localhost:8091/v1/users
```
- GRPC:
```bash
grpcurl -d '<PAYLOAD>' --plaintext -import-path=./internal/servers/grpc/proto/external -import-path=./internal/servers/grpc/proto/user-manager/v1/ -proto service.proto localhost:8091 user_manager.v1.UserManager/CreateUser
```
- RESPONSE PAYLOAD
```json
{
  "id": "4009ed7d-5e5d-4910-8430-d8f841cd2bc2",
  "firstName": "User1",
  "lastName": "Lastname1",
  "nickname": "user1_lastname",
  "email": "user1@gmail.com",
  "country": "NL",
  "createdAt": "2024-08-07T12:57:35Z",
  "updatedAt": "0001-01-01T00:00:00Z"
}
```

2. ### List all users created
- HTTP: 
```bash
curl http://localhost:8091/v1/users
```
- HTTP PAGINATED: 
```bash
curl http://localhost:8091/api/v1/users?pagination=2
```
- HTTP PAGINATED AND FILTERED:
```bash
curl http://localhost:8091/api/v1/users?pagination=2&filterBy=country&filter=NL
```
- HTTP INCLUDING NEXT_PAGE:
```bash
curl http://localhost:8091/api/v1/users?next_page=eyJsaW1pdCI6Miwib2Zmc2V0IjoyLCJmaWx0ZXJfYnkiOiIiLCJmaWx0ZXIiOiIiLCJ0aW1lIjoiMjAyNC0wOC0wNFQyMjo1NDo1My4xMTIzNzErMDI6MDAifQ==
```
- GRPC:
```bash
grpcurl --plaintext -import-path=./internal/servers/grpc/proto/external -import-path=./internal/servers/grpc/proto/user-manager/v1/ -proto service.proto localhost:8091 user_manager.v1.UserManager/ListUsers
```
- GRPC PAGINATED:
```bash
grpcurl -d '{"pagination":2}' --plaintext -import-path=./internal/servers/grpc/proto/external -import-path=./internal/servers/grpc/proto/user-manager/v1/ -proto service.proto localhost:8091 user_manager.v1.UserManager/ListUsers
```
- GRPC PAGINATED AND FILTERED:
```bash
grpcurl -d '{"pagination":2, "filter_by": "country", "filter": "NL"}' --plaintext -import-path=./internal/servers/grpc/proto/external -import-path=./internal/servers/grpc/proto/user-manager/v1/ -proto service.proto localhost:8091 user_manager.v1.UserManager/ListUsers
```
- GRPC INCLUDING NEXT_PAGE:
```bash
grpcurl -d '{"next_page": "eyJsaW1pdCI6Miwib2Zmc2V0IjoyLCJmaWx0ZXJfYnkiOiIiLCJmaWx0ZXIiOiIiLCJ0aW1lIjoiMjAyNC0wOC0wNFQyMjo1NDo1My4xMTIzNzErMDI6MDAifQ=="}' --plaintext -import-path=./internal/servers/grpc/proto/external -import-path=./internal/servers/grpc/proto/user-manager/v1/ -proto service.proto localhost:8091 user_manager.v1.UserManager/ListUsers
```
- RESPONSE PAYLOAD
```json
{
  "users": [
    {
      "id": "22e57170-a622-4281-8d7a-048a52b8075c",
      "firstName": "User5",
      "lastName": "Lastname5",
      "nickname": "user5_lastname",
      "email": "user5@gmail.com",
      "country": "NL",
      "createdAt": "2024-08-07T12:01:52Z",
      "updatedAt": "2024-08-07T12:01:52Z"
    }
  ]
}
```

3. ### Update user
- HTTP:
```bash
curl -X PUT -H "Content-type: application/json" -d '{"first_name": "User1_Updated"}' http://localhost:8091/api/v1/users/22e57170-a622-4281-8d7a-048a52b8075c
```
- GRPC:
```bash
grpcurl -d '{"first_name":"User3_Updated1", "id": "22e57170-a622-4281-8d7a-048a52b8075c"}' --plaintext -import-path=./internal/servers/grpc/proto/external -import-path=./internal/servers/grpc/proto/user-manager/v1/ -proto service.proto localhost:8091 user_manager.v1.UserManager/UpdateUser
```

4. ### Delete user
- HTTP:
```bash
curl -X DELETE localhost:8091/api/v1/users/22e57170-a622-4281-8d7a-048a52b8075c
```
- GRPC:
```bash
grpcurl -d '{"id": "22e57170-a622-4281-8d7a-048a52b8075c"}' --plaintext -import-path=./internal/servers/grpc/proto/external -import-path=./internal/servers/grpc/proto/user-manager/v1/ -proto service.proto localhost:8091 user_manager.v1.UserManager/DeleteUser
```
5. ### Health probe
- HTTP
```bash
curl http://localhost:8091/api/v1/health
```
OUTPUT:
```json
{
  "status": "OK",
  "timestamp": "2024-08-07T13:19:06.525772798Z",
  "system": {
    "version": "go1.22.6",
    "goroutines_count": 9,
    "total_alloc_bytes": 1979584,
    "heap_objects_count": 8948,
    "alloc_bytes": 1979584
  },
  "component": {
    "name": "User's manager HTTP service",
    "version": "v1.0"
  }
}
```

- GRPC_HEALTH_PROBE
```bash
grpc_health_probe -addr=localhost:8091
```
OUTPUT:
```text
status: SERVING
```

## Tests ##
Simple tests for both handlers added. Please, explore them in `internal/servers/(http|grpc)`

## Future improvements ##
* Remove boilerplate code
* Add more tests(api part first)
* Add more operations
* Add authorization


