package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/logutils"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountPoint := flag.Arg(0)

	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("DEBUG"),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)

	// Create the DB schema
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"Node": &memdb.TableSchema{
				Name: "Node",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Path"},
					},
				},
			},
		},
	}

	// Create a new data base
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		log.Fatal(err)
	}

	mdbfs := MDBFS{db}

	c, err := fuse.Mount(
		mountPoint,
		fuse.FSName("MDBFS"),
		fuse.Subtype("MDBFS"),
		fuse.LocalVolume(),
		fuse.VolumeName(""),
	)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		_ = c.Close()
	}()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		for range sig {
			if err := fuse.Unmount(mountPoint); err != nil {
				log.Printf("Signal received, but could not unmount: %s", err)
			} else {
				break
			}
		}
	}()

	err = fs.Serve(c, mdbfs)
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
