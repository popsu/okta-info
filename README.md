# Okta-info

## What?

Query Okta API for

1) Groups the given user is in
2) All users in the given group
3) Difference of 2 groups

## Installation

```shell
# requires go to be installed
go install github.com/popsu/okta-info@latest
```

Or download binary from the [releases page](https://github.com/popsu/okta-info/releases)

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

3. Query difference of two groups:

    ```shell
    okta-info diff <group-name-1> <group-name-2>
    ```
