# S3 Event Archiver

This app, when deployed, enables an admin to create a Route that periodically uploads batches of Events to S3 for archiving purposes.  Events are batched, rather than uploaded individually, specifically to optimize retrival costs.

## Background

While the Notehub's own Event storage is quite flexible, it is tuned for operational analytics and as such there is a built-in retention period after which the Events are deleted.

For some applications, it can be useful (sometimes critically so) to have an Event archive that can be treated as 'cold storage' - useful for disaster recovery but also potentially for ETL-style loading into a database for analytics purposes.

The least expensive options for cold storage are currently Amazon S3 and the S3-compatible services offered by numerous vendors such as Backblaze with its B2 offering.  Backblaze can be particularly intriguing for some because ingress is free.

The archiving solution herein functions by using an intermediate server running the code in this repo to act as the "archivist" which:
1. receives Events sent outbound by a properly-configured Notehub Route
2. saves those Events in its local file system for some period of time
3. when a configured threshold has been reached, packages up groups of Events and uploads them to S3

Because many different applications have differing archiving period requirements, file format requirements, and folder hierarchy requirements, the configuration variables are quite flexible.

## Configuring your Route for Archiving

The first step involved in creating an archiving solution is to clone this repo, run it on a server, and configure an HTTPS endpoint on a domain to which the data can be routed.

Go into your Notehub project and create a new Route of type "General HTTP/HTTPS Request/Response".  Give it any name that you choose such as "Event Archive".

In the URL field, place your server's domain, such as "https://archive.events".

Then, in the HTTP Headers control, select Additional Headers.  Now you must configure header fields as follows:

### archive_id

This field, which is required, is a simple name (unique on your server) that you must configure for this specific archive configuration.  Generally it's useful to pick something short and descriptive that is associated with your project, such as "airnote" or "refrigerator-monitor".

### archive_count_exceeds

There are two thresholds that will trigger an S3 upload.

First is the count of Events that are pending to be uploaded in the "folder" containing events that have been classified in a folder hierarchy (see below).  This number is that count.  By default this number is 1000, but it can be commonly set to 10000 and has a maximum of 25000.

### archive_every_mins

The other threshold that might trigger an S3 upload to occur is the number of minutes that events have been sitting in the "folder" containing events, even if the count threshold has not been reached.  By default this number is 1440 (one day's worth of minutes), and the maximum is 10080 (the number of minutes in a week).

For example, one might configure the "count" to be 10000 and the "mins" to be 1440, which says "when any folder fills up with 10000 events OR if the oldest pending item in that folder is a day old, archive that folder".

### file_access

This must be either "private" or "public-read", depending upon whether you've configured your S3 bucket to be private or if it is configured to allow files to be openly read-only to the world.

### file_folder

Different applications have different preferences with regard to the hierarchical organization of folders in S3.  For example, some might want the top-level folder to be named something like 2022-07 and then to have the files within that folder to just contain groups of events from *all* devices in the project that occurred within that month.

Another might wish to have an organization of dev-238423423048 / 2022-07, thus having all events first classified by device and then by date.  Or perhaps some might like 2022-07 / dev-238423423048, thus having all events classified by date and then by device.

Others still may want a top-level folder with the Archive ID, so they can archive multiple projects into the same bucket.  Or others might like to folder based upon serial number rather than device ID.

This HTTP header field enables you to configure the layout of your folder by providing a template, such as "[id]/[year]-[month]/[device]".  You can arrange this herarchy any way you wish, using the characters that are valid in S3 bucket keys.  The square bracket keywords are substituted as follows:

#### [id]
This route's Archive ID.

#### [device]
The device's Device UID.

#### [product]
The device's Product UID.

#### [sn]
The device's Serial Number.

#### [file]
The event's Notefile ID.

#### [year]
The four digit year that the event was Received.

#### [month]
The two digit month (1-based) that the event was Received.

#### [day]
The two digit day (1-based) that the event was Received.

#### [hour]
The two digit hour (0-based, 24-hour clock) that the event was Received.

#### [weeknum]
The two digit week (1-based) of the year that the event was Received

### file_format

When a file is uploaded to S3, its filename is AAAAAAAAAAAAAAAA-BBBBBBBBBBBBBBBB-CCC.json, where AAA is the Received timeof the first Event in the file, encoded in unix epoch microseconds, BBB is the Received time of the last Event in the file, and CCC is the number of events encoded in the file.

Using this HTTP Header variable, you may configure one of three data formats for the group of events stored in the JSON file:

#### array

This format is a JSON array of all the objects within the file, such as
```
[
  {event1},
  {event2},
  {event3}
]
```

#### object:myfieldname

This format is a JSON array of all the objects within the file, placed underneath a high level object such as
```
{
  myfieldname:[
    {event1},
    {event2},
    {event3}
  ]
}
```

#### ndjson

This format, known as "Newline-Delimited JSON" (see http://ndjson.org) has one JSON Event per line, delimited with the "\n" character.

### bucket_endpoint

This is the endpoint for the S3 service to be called.  For AWS, it can be ommitted or set to "(default)", whereas for B2 it might be set to something like  "s3.us-west-001.backblazeb2.com" as instructed by Backblaze.

### bucket_region

This is the region of your S3 service, such as "us-east-1" or "us-west-001".

### bucket_name

This is the name of the bucket you created in your S3 service.

### key_id

This is the Identifier portion of the Secret Access Key, such as AKIASL5UXI756LWKJU45

### key_secret

This is the Secret portion of the Secret Access Key, such as hxN6RPv7nKCn72ptdCV2BcTbIynxCgCr042vA2Zl


## Security

This archiving solution is written with no authentication, and requires no explicit configuration outside of what is specified in the Notehub Route's HTTP Header fields.  All data routed to this archiving solution will be kept in cleartext within the file system until such a time when it is archived to S3 and deleted locally.

S3 (or equivalent) authentication is based upon "Secret Access Keys".  The SAK, when configured, will be 'in the clear' in the Route, and so it is important when setting up the access keys that you create a user/key that *only* has permission to Upload to the S3 bucket to which data is being uploaded.  For example, in AWS this involves:
1. Open the IAM console and click Policies
2. Select to Create a new Customer-Managed Policy such as "my-event-archive, with the {}JSON that is:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:PutObject"
            ],
            "Resource": "arn:aws:s3:::my-event-archive/*"
        }
    ]
}
```
3. Open the IAM console, and click Users
4. Click Add User, choose a new IAM user name, and select Access Key as the credential type
5. In the Permissions section, choose "Attach existing policies directly", and select your "my-event-archive" policy
6. Create the user.  At that point, you will be show the Access Key's ID and the Secret for the key.  You'll use these when configuring the Route




## To learn more about Blues Wireless, the Notehub, S3, and B2, see:

https://blues.com
https://notehub.io
https://aws.amazon.com/pm/serv-s3
https://www.backblaze.com/b2/cloud-storage.html

## License

Copyright (c) 2019 Blues Inc. Released under the MIT license. See
[LICENSE](LICENSE) for details.
