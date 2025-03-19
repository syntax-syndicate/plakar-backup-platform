# plakar - Effortless backup

## Introduction

plakar is designed for effortless and secure data protection. It provides an intuitive, powerful, and scalable backup solution.

plakar is:
- **effortless**: easy to use
- **secure**: provide true end-to-end encryption for data and metadata
- **reliable**: backups are stored in an immutable repository
- **scalable vertically**: back up and restore very large data set with little RAM
- **scalable horizontally**: high concurrency support, multiple backups in a single repository
- **searchable**: browse your snapshots from plakar UI, sort, search and compare data
- **fast**: backup, check, sync and restore are fast (beta is optimized for large data sets, optimization for small one are in progress)
- **efficient**: more restore points, less storage space, with unmatched deduplication and compression rates
- **Open Source and well maintained**: open source forever and now maintained by Plakar Korp

plakar is providing useful features:
- **instant recovery**: mount instantly large backup on any devices
- **distributed backup**: synchronize your backup repository to implement simple (3,2,1 rule) or complex (push, pull, sync) strategies
- **granular restore**: restore a complete snapshot or only a set of files
- **cross-storage restore**: for example back up from S3 to restore on a file system
- **production safe-guarding**: automatically increase or decrease backup speed to protect your production workloads (very limited on beta)
- **lock free maintenance**: garbage collect without production interruptions (under testing, beta has a security lock)
- **connectors**: back up any source (file systems, object stores, SaaS applications...) with granular restore (limited in beta)

Simplicity and efficiency are plakar's main priorities.


## Current version

The current version is v1.0.0-beta.7

It is our fifth beta version with a stable storage format.


## Requirement

`plakar` requires Go 1.23.3 or higher,
it may work on older versions but hasn't been tested.


## Installing the CLI

```
go install github.com/PlakarKorp/plakar/cmd/plakar@latest
```

## Quickstart

plakar quickstart: https://docs.plakar.io/en/quickstart/

A taste of plakar (please follow the quickstart to begin):
```
$ plakar at /var/backups create                             # Create a repository
$ plakar at /var/backups backup /private/etc                # Backup /private/etc
$ plakar at /var/backups ls                                 # List all repository backup
$ plakar at /var/backups restore -to /tmp/restore 9abc3294  # Restore a backup to /tmp/restore
$ plakar at /var/backups ui                                 # Start the UI
$ plakar at /var/backups sync to @s3                        # Synchronise a backup repository to S3

```

## Documentation

For the latest information,
you can read the documentation available at https://docs.plakar.io

## Community

You can join our very active [Discord](https://discord.gg/uuegtnF2Q5) to discuss the project !

## Warning

plakar is currently in beta and **NOT** production ready yet but it is most definitely stable enough to be tested by others.

Feel free to give it a try, give feedback on what you like/dislike and report bugs.
