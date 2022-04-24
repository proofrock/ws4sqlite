### From the issue tracker

- (See #2) Start ws4sqlite without specifying a db, then manage creation/deletion via REST:

```
PUT http://localhost:12321/mynewdatabase # Creates db
POST http://localhost:12321/mynewdatabase # Queries db
DELETE http://localhost:12321/mynewdatabase # Deletes db
```

- (See #5) Expand the documentation on building under Windows

### From discussions ([here](https://news.ycombinator.com/item?id=30636796))

- Versioning of the call protocol
- Precondition: a query that decides if the transaction can go on
- Control transactions explicitly
- Websockets support
- Compile in sqlite's extensions
- Drivers with "native" APIs (JDBC, Go SQL...)

## Others

- Client library for Deno
