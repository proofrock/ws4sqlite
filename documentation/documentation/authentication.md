# 🔓 Authentication

Going back to these snippets of [the configuration file](configuration-file.md):

```yaml
[...]
    auth:
      mode: HTTP
      customErrorCode: 499
      byCredentials:
        - user: myUser
          hashedPassword: "$2b$12$Xo7tQh0BDzDAiPghc7AU1Ocx2MnGls46Ot55y4MQNtPRhK0nemyWq"
 [...]
    auth:
      mode: INLINE
      byQuery: SELECT 1 FROM AUTH WHERE USER = :user AND PASSWORD = :password
```

The `auth`nodes represent the structure that instructs ws4sql to protect that db with authentication.

> ℹ️ If a database is protected with auth and the client provides wrong credentials, or doesn't provide any, the HTTP answer will be `401 Unauthorised` or the custom code specified in `customErrorCode` (see above).

### On the server

#### Authentication `mode`

_Lines 3, 9; string; mandatory_

The first, common parameter is `mode`, that indicates the means that the client is required to use to authenticate. It can be:

* `HTTP`: the client needs to use [HTTP basic authentication](https://it.wikipedia.org/wiki/Basic\_access\_authentication);
* `INLINE`: the credentials needs to be specified in the JSON request. See [below](authentication.md#on-the-client).

#### Custom error code

_Line 4; number_

If this parameter is not specified, an authentication error will return the standard `401 Not Authorized`. 

Often a browser will react to this by displaying a standard authentication dialog; if this is not desired 
(because the auth has a custom implementation, for example) it may be needed to specify an alternative error code. 
The `customErrorCode` configuration allows to do exactly this.

#### Specifying the credentials

_Lines 5-7, 13; object; mandatory_

You can see that there are two methods to configure the resolution of the credentials on the server:

* Provide a query that will be executed in the database, as in Line 13.\
  \
  The query SQL must contain two placeholders, `:user` and `:password`, that will be replaced by the server with the username and password provided by the client.\
  If DuckDB is being used, the placeholders will be `$user` and `$password`.\
  \
  If the query returns at least one result, the credentials are valid; if it returns 0 records, access will be denied.\\
* Provide a set of credentials in the config file itself, as in Lines 6-7.\
  \
  The password is to be provided as a BCrypt hash. See [below](authentication.md#generating-the-token) to learn how to hash passwords.

The `auth` block is not mandatory. If provided, the database will be protected with it; if omitted, no authentication is requested. If you provide one, it will be ignored.

> ⚠️ A client sends the credentials to ws4sql in plaintext, so it is better to be on a protected connection like HTTPS (e.g. by using a reverse proxy). See the [security](../security.md#authentication) page for further information.

#### Generating hashes

> ℹ️ When including the hash in the YAML, be aware that there may be characters to escape. Best thing is to use single quotes around the string.

You can:

- Use a website, google for it. Usually these sites send the secret to their servers, so you shouldn’t use them for "production" secrets.

- Use htpasswd from apache-utils (or the relevant package for your distribution). Run the following commandand remove the initial : from the result.

```bash
htpasswd -nbBC 10 "" mysecret
```

- Use docker and the caddy image, with the following commandline.

```bash
docker run --rm caddy caddy hash-password -p 'mysecret'
```

### Credentials in the request (`INLINE` mode)

When a database is protected with authentication in [`INLINE` mode](authentication.md#mode), the client needs to specify the credentials in the request itself. Simply include a node like this:

```json
{
    "credentials": {
        "user": "myUser1",
        "password": "myCoolPassword"
    },
    [...]
```

If the token verification fails, the response will be returned after 1 second, to prevent brute forcing. The wait time is per database: different failed requests for the same database will "stack", while different databases will work concurrently.

> ⚠️ Again, really: a client sends the credentials to ws4sql in plaintext, so it is better to be on a protected connection like HTTPS (e.g. by using a reverse proxy). See the [security](../security.md#authentication) page for further information.
