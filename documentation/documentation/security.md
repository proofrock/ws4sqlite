# 🛡 Security

Here is a more detailed discussion of the various security features.

### Authentication

The first and most effective line of defense is certainly authentication, [documented here](authentication.md).

ws4sql allows a client to authenticate with it using a traditional credentials system. In other words, the client supplies an username and a password.

The client can do this in one of two ways:

* Using the standard HTTP Basic Authentication; or
* Inserting the credentials in the request JSON.

> ⚠️ Beware that the password is sent by the client in plaintext; if you are serving a database on an unprotected connection, it could be snooped. It's strongly advised to provide HTTPS with a reverse proxy.

On the server, the set of valid credentials can be configured in two ways:

* Defining a list of credentials in the configuration file (the password is to be provided as a BCrypt hash);
* Defining a SQL query that authenticates them, probably against a table in the database itself.

This mechanism is designed to be flexible and effective. Of course, a password is only safe if it's kept secret, so if you connect from a web app, it can't be hard-coded in the javascript. A better approach would be to ask for it to the user.

### Read-only databases

If you want to ensure that a database cannot be changed, even if access to it is somehow gained, it's possible to specify to open it as read only. Docs [here](configuration-file.md#readonly).

### Stored Statements to prevent SQL Injection

If you call ws4sql from a web page, or even if you expose it on an unsecured channel, it's all too easy for someone to call it as if they were the web page, and execute random SQL on it.

In such a case, it may be useful to [define](stored-statements.md) a set of "safe" SQL statements, on the server, and [restrict](configuration-file.md#useonlystoredqueries) _any_ client to use just them.

For example, suppose you have a database with a list of personal information, keyed by social security number. If someone "ask" with a valid SSN, you want to give them the information, but of course you don't want to allow anyone to extract the whole list.

Just define a query by SSN as the only executable statement, and it's a pretty effective measure. The complete discussion of such a case is [here](stored-statements.md#limiting-the-server-to-executing-stored-queries).

Also, to prevent SQL injection, be sure to use [parameters](requests.md#parameter-values-for-the-query-statement)!

### CORS Allowed Origin

If you access ws4sql from a web context, you'll probably need to configure CORS. The service lets you do it, and instead of `*` you can specify the calling address (to use as value for the Allow Origin header), so that only the calls from that address are allowed.

See the [docs](configuration-file.md#corsorigin). Also, be sure to use HTTPS!

### Binding to a network interface

As ny services of the same type, ws4sql allows to limit the server to a network interface, by binding to an address of the host. Docs [here](running.md#bind-host).

For example, this can be used to restrict access to the service only to requests coming from a VPN or another trusted network.

### Use a reverse proxy (if going on the internet)

All these measures are effective, but to complete them it's advisable to use HTTPS, especially if you plan to expose ws4sql on the internet.

HTTPS configuration is _not_ part of ws4sql, because it would add a lot of code that is outside the scope of the application (KISS!). Instead, configure a reverse proxy to secure the HTTP connection - for example with a Let'sEncrypt certificate.

Some options that are very easy to configure are [Caddy](https://caddyserver.com) or [LinuxServer's Swag](https://docs.linuxserver.io/general/swag) Docker image. Documentation [here](../integrations/reverse-proxy.md).
