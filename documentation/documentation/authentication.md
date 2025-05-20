# 🔓 Authentication

Going back to these snippets of [the configuration file](configuration-file.md):

```yaml
[...]
    auth:
      mode: HTTP
      customErrorCode: 499
      byCredentials:
        - user: myUser1
          password: myCoolPassword
        - user: myUser2
          hashedPassword: b133a0c0e9bee3be20163d2ad31d6248db292aa6dcb1ee087a2aa50e0fc75a[e2
 [...]
    auth:
      mode: INLINE
      byQuery: SELECT 1 FROM AUTH WHERE USER = :user AND PASSWORD = :password
```

The `auth`nodes represent the structure that instructs ws4sql to protect that db with authentication.

{% hint style="info" %}
If a database is protected with auth and the client provides wrong credentials, or doesn't provide any, the HTTP answer will be `401 Unauthorised`.
{% endhint %}

### On the server

#### Authentication `mode`

_Lines 3, 12; string; mandatory_

The first, common parameter is `mode`, that indicates the means that the client is required to use to authenticate. It can be:

* `HTTP`: the client needs to use [HTTP basic authentication](https://it.wikipedia.org/wiki/Basic\_access\_authentication);
* `INLINE`: the credentials needs to be specified in the JSON request. See [below](authentication.md#on-the-client).

#### Custom error code

_Line 4; number_

If this parameter is not specified, an authentication error will return the standard `401 Not Authorized`. Often a
browser will react to this by displaying a standard authentication dialog; if this is not desired (because the auth has
a custom implementation, for example) it may be needed to specify an alternative error code. The `customErrorCode`
configuration allows to do exactly this.

#### Specifying the credentials

_Lines 5-9, 13; object; mandatory_

You can see that there are two methods to configure the resolution of the credentials on the server:

* Provide a query that will be executed in the database, as in Line 13.\
  \
  The query SQL must contain two placeholders, `:user` and `:password`, that will be replaced by the server with the username and password provided by the client.\
  \
  If the query returns at least one result, the credentials are valid; if it returns 0 records, access will be denied.\\
* Provide a set of credentials in the config file itself, as in Lines 6-9.\
  \
  You can specify the password as plain text (ensure that the file is not world-readable...) or as SHA-256 hashes. See [below](authentication.md#generating-the-token) to learn how to hash passwords.

The `auth` block is not mandatory. If provided, the database will be protected with it; if omitted, no authentication is requested. If you provide one, it will be ignored.

{% hint style="danger" %}
The password are passed in cleartext, so it is better to be on a protected connection like HTTPS (e.g. by using a reverse proxy). See the [security](../security.md#authentication) page for further information.
{% endhint %}

#### Generating hashes

{% hint style="warning" %}
Be careful not to include any whitespace in the text to hash, including any carriage return. If using `echo` it's better to specify the `-n` flag.
{% endhint %}

In order to generate hashes for the password, you can use an online service like [this](https://emn178.github.io/online-tools/sha256.html), but it's better not to trust anything online. In Linux or MacOS you can instead use this one-liner:

```bash
read -p Key: -rs ws4s_token && echo && echo -n $ws4s_token | shasum -a 256 -|head -c 64 && echo && ws4s_token=
```

This will read a string from the stdin without echoing it, and outputs the hash to use.

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
