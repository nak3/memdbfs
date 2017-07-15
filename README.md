# memdbfs

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
