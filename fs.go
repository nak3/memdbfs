package main

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"os"
	"strings"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/hashicorp/go-memdb"
)

var _ fs.FS = &MDBFS{} // Root

// MDBFS implements the hello world file system.
type MDBFS struct {
	db *memdb.MemDB
}

//const rootNodeID = 1

// Root creates top direcoty.
func (mdbfs MDBFS) Root() (fs.Node, error) {
	return &Node{mdbfs: mdbfs, Path: "", ID: mdbfs.newUniqueID(""), Mode: os.ModeDir | defaultDirPerms}, nil
}

// remove removes a node give its name and its parent ID.
// If 'checkChildren' is true, fails if the node has children.
func (mdbfs MDBFS) remove(parentPath string, name string, checkChildren bool) error {
	txn := mdbfs.db.Txn(true)
	defer txn.Abort()
	_, err := txn.DeleteAll("Node", "id", fmt.Sprintf(parentPath+"/"+name))
	if err != nil {
		return err
	}
	// Check if there are any children.
	if checkChildren {
		// TODO
	}
	txn.Commit()
	return nil
}

// lookup gets node from file or directory path.
func (mdbfs MDBFS) lookup(n *Node, name string) (*Node, error) {
	txn := mdbfs.db.Txn(false)
	defer txn.Abort()
	raw, err := txn.First("Node", "id", fmt.Sprintf(n.Path+"/"+name))
	if err != nil {
		return nil, err
	}
	if raw != nil && raw.(*Node).ID == n.ID {
		return nil, sql.ErrNoRows
	}
	node := &Node{}
	if raw != nil {
		node = raw.(*Node)
	} else {
		return nil, sql.ErrNoRows
	}
	return node, err
}

// list gets node list under the directory.
func (mdbfs MDBFS) list(n *Node) ([]fuse.Dirent, error) {
	txn := mdbfs.db.Txn(false)
	defer txn.Abort()
	nodes, err := txn.Get("Node", "id_prefix", n.Path)
	if err != nil {
		return nil, err
	}
	var results []fuse.Dirent
	for {
		e := nodes.Next()
		if e == nil {
			break
		}
		// fullPath to single basename (e.g /top/foo/bar -> bar)
		ret := strings.Replace(e.(*Node).Path, fmt.Sprintf(n.Path+"/"), "", 1)
		if ret != e.(*Node).Name {
			continue
		}
		dirent := fuse.Dirent{Type: fuse.DT_Unknown, Name: e.(*Node).Name, Inode: e.(*Node).ID}
		results = append(results, dirent)
	}
	return results, nil
}

// newDirNode returns a new node struct corresponding to a directory.
func (mdbfs MDBFS) newDirNode(path string, name string) *Node {
	fullPath := fmt.Sprintf(path + "/" + name)
	return &Node{
		mdbfs: mdbfs,
		ID:    mdbfs.newUniqueID(fullPath),
		Path:  fullPath,
		Mode:  os.ModeDir | defaultDirPerms,
		Name:  name,
	}
}

// create inserts a new node.
func (mdbfs MDBFS) create(inode *Node) (*Node, error) {
	txn := mdbfs.db.Txn(true)
	if err := txn.Insert("Node", inode); err != nil {
		return nil, err
	}
	txn.Commit()
	return inode, nil
}

// newUniqueID creates hash ID from path.
func (mdbfs MDBFS) newUniqueID(path string) (id uint64) {
	h := fnv.New64()
	h.Write([]byte(path))
	return h.Sum64()
}

// newFileNode returns a new node struct corresponding to a file.
func (mdbfs MDBFS) newFileNode(path string, name string) *Node {
	fullPath := fmt.Sprintf(path + "/" + name)
	return &Node{
		mdbfs: mdbfs,
		ID:    mdbfs.newUniqueID(fullPath),
		Path:  fullPath,
		Mode:  defaultFilePerms,
		Name:  name,
	}
}
