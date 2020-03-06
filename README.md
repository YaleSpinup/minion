# minion

[![CircleCI](https://circleci.com/gh/YaleSpinup/minion.svg?style=svg)](https://circleci.com/gh/YaleSpinup/minion)

Minion is a naive distributed job scheduler.

## Endpoints

```
GET /v1/minion/ping
GET /v1/minion/version
GET /v1/minion/metrics

GET /v1/minion/{account}/jobs
POST /v1/minion/{account}/jobs
GET /v1/minion/{account}/jobs/{id}
PUT /v1/minion/{account}/jobs/{id}
DELETE /v1/minion/{account}/jobs/{id}
```

## Usage

## Authentication

Authentication is accomplished via a pre-shared key.  This is done via the `X-Auth-Token` header.

## Job Types

### dummy

A dummy runner job just attempts to execute a template with the given account name and return that string.

#### example dummy job

```json
{
    "description": "Do some dumb thing to my server",
    "details": {
        "runner": "dummyRunner",
    },
    "group": "space-xy",
    "id": "6bcfa79f-615e-470d-97c1-687f3357497d",
    "modified_at": "2020-02-27T16:22:09Z",
    "modified_by": "someone",
    "name": "dummy-spin1234567",
    "schedule_expression": "* * * ? *",
    "enabled": true
}
```

### instance

An instance runner job executes an action on an instance.  Currently supported actions are `stop`, `start` and `reboot`

#### example instance job

```json
{
    "description": "Start my server",
    "details": {
        "instance_action": "start",
        "instance_id": "i-aaaabbbb11112222",
        "runner": "instanceRunner"
    },
    "group": "space-xy",
    "id": "7cf7433f-f6c2-496e-8fe9-ed4776d130c1",
    "modified_at": "2020-02-28T18:14:26Z",
    "modified_by": "someone",
    "name": "start-spinaaaabbbb11112222",
    "schedule_expression": "* * * ? *",
    "enabled": true
}
```

## Create a Job

POST `/v1/minion/{account}/jobs`

### Request

```json
{
    "description": "Do some dumb thing to my server",
    "details": {
        "runner": "dummyRunner",
    },
    "group": "space-xy",
    "modified_by": "someone",
    "name": "dummy-spin1234567",
    "schedule_expression": "* * * ? *",
    "enabled": true
}
```

### Response

```json
{
    "description": "Do some dumb thing to my server",
    "details": {
        "runner": "dummyRunner",
    },
    "group": "space-xy",
    "id": "6bcfa79f-615e-470d-97c1-687f3357497d",
    "modified_at": "2020-02-27T16:22:09Z",
    "modified_by": "someone",
    "name": "dummy-spin1234567",
    "schedule_expression": "* * * ? *",
    "enabled": true
}
```

## Update a Job

PUT `/v1/minion/{account}/jobs/{id}`

### Request

```json
{
    "description": "Do some dumb thing to my server",
    "details": {
        "runner": "dummyRunner",
    },
    "group": "space-xy",
    "id": "6bcfa79f-615e-470d-97c1-687f3357497d",
    "modified_by": "someone",
    "name": "dummy-spin1234567",
    "schedule_expression": "* * * ? *",
    "enabled": false
}
```

### Response

```json
{
    "description": "Do some dumb thing to my server",
    "details": {
        "runner": "dummyRunner",
    },
    "group": "space-xy",
    "id": "6bcfa79f-615e-470d-97c1-687f3357497d",
    "modified_at": "2020-02-28T16:22:09Z",
    "modified_by": "someone",
    "name": "dummy-spin1234567",
    "schedule_expression": "* * * ? *",
    "enabled": false
}
```

## List Jobs

GET `/v1/minion/{account}/jobs`

### Response

```json
[
    "11e7b876-7a10-4c3e-93a7-e77eb3c68b58",
    "6bcfa79f-615e-470d-97c1-687f3357497d",
    "747f437a-a4af-48b2-a021-888bb8943a9b",
    "7cf7433f-f6c2-496e-8fe9-ed4776d130c1"
]
```

## Get a Job

GET `/v1/minion/{account}/jobs/{id}`

```json
{
    "description": "Do some dumb thing to my server",
    "details": {
        "runner": "dummyRunner",
    },
    "group": "space-xy",
    "id": "6bcfa79f-615e-470d-97c1-687f3357497d",
    "modified_at": "2020-02-28T16:22:09Z",
    "modified_by": "someone",
    "name": "dummy-spin1234567",
    "schedule_expression": "* * * ? *",
    "enabled": true
}
```

## Delete a Job

DELETE `/v1/minion/{account}/jobs/{id}`

## Author

E Camden Fisher <camden.fisher@yale.edu>

## License

GNU Affero General Public License v3.0 (GNU AGPLv3)  
Copyright (c) 2020 Yale University
