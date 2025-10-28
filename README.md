# Okta-info

## What?

Query Okta API for

1) Groups the given user is in
2) All users in the given group
3) Difference of 2 groups
4) Rules related to a group

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

4. Query rules related to a group:

    Currently only works for few rules, so this might not work as expected

    ```shell
    okta-info rule group <group-name> # Search using group name
    okta-info rule name <rule name> # Search using rule name
    ```

5. Query user email by user ID:

    ```shell
    okta-info userid <user-id>
    ```

## Deprovisioned users

By default deprovisioned users are not shown. To show them, set the following environment variable to truthy value: `OKTA_INFO_SHOW_DEPROVISIONED_USERS=true`
