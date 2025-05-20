# ❌ Errors

#### Global Errors

When a statement in a transaction fails, it generates an error. If the error is not managed by specifying `noFail` in the request for that statement, the whole transaction is aborted and rolled back.

In this case, we get a HTTP status of:

* `400 Bad Request`: for formal errors;
* `401 Unauthorized`: for errors related to authorization;
* `404 Not Found`: if the database ID specified in the URL is not among the configured ones;
* `500 Internal Server Error`: for server errors (could be transient).

And a JSON object in the response body, similar to this:

```json
{
    "reqIdx": 1,
    "message": "near \"SELECTS\": syntax error"
}
```

This message indicates that the query with index 1 failed, with the reason for the failure being in the `error`string node.

{% hint style="warning" %}
Indexes are 0-based: the query with index 1 is actually the second one.
{% endhint %}

For errors that are related not to a particular query, but to a general failure, we get an index of `-1`:

```json
{
    "reqIdx": -1,
    "error": "wrong credentials"
}
```

#### Managed Errors

If `noFail` is in a request, as for the 4th node in the [Request example](requests.md), the entire transaction is NOT aborted/rolled back, but an error is reported in the [Response](responses.md#non-blocking-error):

```json
{
    "success": false,
    "error": "UNIQUE constraint failed: TEMP.ID"
},
```

The `success` boolean node is `false`, and the reason for the error is given in the second string node.

In this case, the return code is `200` and the transaction is committed, of course the failed statement is not included in the commit. Pay attention to the implications!
