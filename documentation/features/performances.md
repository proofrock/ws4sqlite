# 🚄 Performances

{% hint style="warning" %}
This page is quite outdated. New, more complete measurements are needed. Help appreciated ;-)
{% endhint %}

ws4sqlite v0.9.1 with Apache JMeter. Debian Linux system, Azure Standard B2s (2 vcpus).

Tested with 100, 1000, ... concurrent requests. Single SELECT on a file-based database, by primary key, on a 2000-records table. Fits in cache, WAL mode (not read only).

```
   100 in 00:00:00 =  221.7/s Avg: 14 Min: 1 Max: 54 Err: 0
  1000 in 00:00:02 =  460.8/s Avg:  2 Min: 0 Max: 42 Err: 0
 10000 in 00:00:12 =  834.1/s Avg:  1 Min: 0 Max: 82 Err: 0
100000 in 00:01:02 = 1621.1/s Avg:  0 Min: 0 Max: 54 Err: 0
```

Sub-millisecond response time, scales better than linear. Notice that requests to the underlying db are serialized.
