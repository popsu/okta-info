# Okta-info

## What?

Query Okta API for

1) Groups the given user is in
2) All users in the given group

## Installation

```shell
# requires go to be installed (sorry no binaries yet)
go install github.com/popsu/okta-info@latest
```

## Usage

Set the following environment variables:

```shell
OKTA_INFO_ORG_URL=https://<your-org>.okta.com
OKTA_INFO_API_TOKEN=<your-api-token>
```

1. Query for groups the given user is in:

    ```shell
    # user.name without @<your-org>
    okta-info user <user.name>
    ```

2. Query for all users in the given group:

    ```shell
    okta-info group <group-name>
    ```
