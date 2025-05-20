# 🌐 Web Server

Ws4sqlite embeds a web server for static resources. This may be useful if you want to call it from a web page, to serve
it on the same port as ws4sqlite itself, thus avoiding the need for CORS configurations.

To activate it, specify the `--serve-dir` commandline switch, with the directory to serve. The contents of that directory
will be served under the "`/`" virtual path (i.e. the root), with the GET method. Of course, the specified directory must
exist. 

{% hint style="info" %}
The connections to the database(s) are served as POST, so there's no overlap.
{% endhint %}

The web server is configured with the following capabilities:

| Feature             |              |
|---------------------|--------------|
| Default resource    | `index.html` |
| Byte range requests | Enabled      |
| Dir. navigation     | Disabled     |
| Compression         | Disabled     |
| Max age             | Disabled     |
