# 💡 Client Libraries

Client libraries are provided, with bindings to contact ws4sql from your code.

The reasons are several:

* You don't have to deal directly with JSON, which can be cumbersome in some languages;
* You have the convenience of type safety and compile-time checks, in languages that support them;
* The library performs checks for the requests being well formed, preventing errors;
* It maps the errors to the language platform's idiomatic facilities.

With clients, ws4qlite acts as a remote protocol for SQLite.

#### Platforms

We provide libraries for:

* **Java/JDK**: [https://github.com/proofrock/ws4sqlite-client-jvm](https://github.com/proofrock/ws4sqlite-client-jvm)
* **Go(lang)**: [https://github.com/proofrock/ws4sqlite-client-go](https://github.com/proofrock/ws4sqlite-client-go)
* ...more to follow
