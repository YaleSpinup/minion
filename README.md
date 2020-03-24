# minion

[![CircleCI](https://circleci.com/gh/YaleSpinup/minion.svg?style=svg)](https://circleci.com/gh/YaleSpinup/minion)

Minion is a naive distributed job scheduler.

## Endpoints

```
GET /v1/minion/ping
GET /v1/minion/version
GET /v1/minion/metrics

GET /v1/minion/{account}/jobs
GET /v1/minion/{account}/jobs/{group}
POST /v1/minion/{account}/jobs/{group}
GET /v1/minion/{account}/jobs/{group}/{id}
PUT /v1/minion/{account}/jobs/{group}/{id}
DELETE /v1/minion/{account}/jobs/{group}
DELETE /v1/minion/{account}/jobs/{group}/{id}

PATCH /v1/minion/{account}/jobs/{group}/{id}
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

POST `/v1/minion/{account}/jobs/space-xy`

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

PUT `/v1/minion/{account}/jobs/space-xy/6bcfa79f-615e-470d-97c1-687f3357497d`

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

## List Jobs in an account

GET `/v1/minion/{account}/jobs`

### Response

```json
[
    "space-xy/11e7b876-7a10-4c3e-93a7-e77eb3c68b58",
    "space-xy/6bcfa79f-615e-470d-97c1-687f3357497d",
    "space-ab/747f437a-a4af-48b2-a021-888bb8943a9b",
    "space-ab/7cf7433f-f6c2-496e-8fe9-ed4776d130c1"
]
```

## List Jobs in a space

GET `/v1/minion/{account}/jobs/space-xy`

### Response

```json
[
    "11e7b876-7a10-4c3e-93a7-e77eb3c68b58",
    "6bcfa79f-615e-470d-97c1-687f3357497d"
]
```

## Get a Job

GET `/v1/minion/{account}/jobs/space-xy/6bcfa79f-615e-470d-97c1-687f3357497d`

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

DELETE `/v1/minion/{account}/jobs/space-xy/6bcfa79f-615e-470d-97c1-687f3357497d`

## Delete all jobs in a group

DELETE `/v1/minion/{account}/jobs/space-xy`

## Run a Job

PATCH `/v1/minion/{account}/jobs/space-xy/6bcfa79f-615e-470d-97c1-687f3357497d`

Note: At the moment, this checks if the job exists in the jobs repository and then adds it to the
jobs queue.  There is a potential race condition, since the jobs queue reads from the local cache
when executing jobs, so it may be missing if it was just created and hasn't been cached by the
loader yet.

## Author

E Camden Fisher <camden.fisher@yale.edu>

## License

GNU Affero General Public License v3.0 (GNU AGPLv3)  
Copyright (c) 2020 Yale University
