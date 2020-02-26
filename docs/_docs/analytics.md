---
title: Analytics
---

Wash collects anonymous data about how you use it. You can opt out of providing this data.

## What data does Wash collect?
* Version of Wash
* User locale
* Architecture
* Method invocations (for shipped plugins only)
  * This includes any Wash action invocation
  * It also includes the entry's plugin

This data is associated with Bolt analytics' UUID (if available); otherwise, the data is associated with a random, non-identifiable user UUID.

## Why does Wash collect data?
Wash collects data to help us understand how it's being used and make decisions about how to improve it.

## How can I opt out of Wash data collection?
To disable the collection of analytics data add the following line to `~/.puppetlabs/wash/analytics.yaml`:

```
disabled: true
```

You can also disable the collection of analytics data by setting the `WASH_DISABLE_ANALYTICS` environment variable to `true` before starting up the Wash daemon.
