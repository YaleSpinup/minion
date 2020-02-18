# minion

[![CircleCI](https://circleci.com/gh/YaleSpinup/minion.svg?style=svg)](https://circleci.com/gh/YaleSpinup/minion)

Minion is a naive distributed job scheduler.

## Endpoints

```
GET /v1/minion/ping
GET /v1/minion/version
GET /v1/minion/metrics
```

## Usage

## Authentication

Authentication is accomplished via a pre-shared key.  This is done via the `X-Auth-Token` header.

## Author

E Camden Fisher <camden.fisher@yale.edu>

## License

GNU Affero General Public License v3.0 (GNU AGPLv3)  
Copyright (c) 2020 Yale University
