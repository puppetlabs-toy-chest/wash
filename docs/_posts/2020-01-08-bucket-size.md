---
title: "The size of a cloud storage bucket"
description: "The obvious way to get the size of an S3 bucket doesn't scale, and the right way is hard to find. Wash makes this easy."
author: michaelsmith
---

A recent joke about finding the size of an S3 bucket made its way across my feed recently
<blockquote class="twitter-tweet"><p lang="en" dir="ltr">Me: ok so you bill based on storage used?<br>AWS: yes<br>Me: can I find out how much storage I am using?<br>AWS: haha haha of course not<br>Me: internet, please assist<br>Internet: oh it’s simple just list all your billions of objects and sum by size!!!</p>&mdash; @jordansissel (@jordansissel) <a href="https://twitter.com/jordansissel/status/1199073196885983232?ref_src=twsrc%5Etfw">November 25, 2019</a></blockquote> <script async src="https://platform.twitter.com/widgets.js" charset="utf-8"></script>

If you [Google it](https://www.google.com/search?q=size+of+s3+bucket), you'll find a number of sites telling you how to do this, with prominent questions on [serverfault](https://serverfault.com/questions/84815/how-can-i-get-the-size-of-an-amazon-s3-bucket) and [stackoverflow](https://stackoverflow.com/questions/32192391/how-do-i-find-the-total-size-of-my-aws-s3-storage-bucket-or-folder). The first answer is usually
```
aws s3 ls --summarize --human-readable --recursive s3://bucket/folder
```
or something similar. As the comments mention, this gets excrutiatingly slow for buckets with a lot of files because it's doing an API call to get the size of each object.

Scrolling down far enough on serverfault/stackoverflow will get you a much more efficient method
```
aws cloudwatch get-metric-statistics --namespace AWS/S3 --start-time 2015-07-15T10:00:00  --end-time 2015-07-31T01:00:00 --period 86400 --statistics Average --region us-east-1  --metric-name BucketSizeBytes --dimensions Name=BucketName,Value=myBucketNameGoesHere Name=StorageType,Value=StandardStorage
```
Just type that out, make sure you get the right time period, and voila, instant answer.

This seemed like essential information about a bucket, and relatively low-cost to get. So we added it to Wash's metadata on S3 buckets
```
wash aws/proj/resources/s3 > meta my-bucket
Crtime: "2019-06-21T18:03:13Z"
Region: us-west-2
Size:
  Average: 153423000248
  HumanAvg: 153 GB
  Maximum: 153423000248
  Minimum: 153423000248
TagSet: null
```
or using the handy [yq](https://github.com/kislyuk/yq)
```
wash aws/profile/resources/s3 > meta my-bucket | yq -r .Size.HumanAvg
153 GB
```

I can now use that to filter buckets based on their size, such as finding all buckets over a gigabyte
```
wash . > find -fullmeta -meta .Size.Minimum +1G
my-bucket
other-bucket
...
```

A similar item was added for Google Cloud Storage buckets, although it differs slightly because GCP makes the current size easily accessible instead of giving you average/min/max
```
wash gcp/proj/storage > find -fullmeta -maxdepth 1 -meta .Size +1G
my-gcp-bucket
...
```

This kind of information is essential to managing cloud storage. Our tools should make it easy to see.