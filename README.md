# memdbfs

[![Build Status](https://travis-ci.org/nak3/memdbfs.svg?branch=master)](https://travis-ci.org/nak3/memdbfs)

What is memdbfs?
---
memdbfs is a file system built on [go-memdb](https://github.com/hashicorp/go-memdb). It uses memdb as a backend storage.

Status
---
memdbfs is an experimental project by [@nak3](https://github.com/nak3). Currently it does not aim to practicable use.

Usage
---
~~~
make deps
make build
mkdir -p /tmp/memdbfs/mnt 
./bin/memdbfs /tmp/memdbfs/mnt
~~~

Now you can test the mount point `/tmp/memdbfs/mnt`. Again, this repository is still experimental status so there are some unsupported file operations.

_NOTE_ You need fuse package for your host first. Please follow each distribution's steps.

Acknowledgment
---
memdb is built on [go-memdb](https://github.com/hashicorp/go-memdb)@hashicorp and [fuse](bazil.org/fuse/fs)@bazil.org. I have refered to [example-go/filesystem](https://github.com/cockroachdb/examples-go/tree/master/filesystem)@cockroachdb for the fuse implementation.
