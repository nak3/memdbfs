package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"sync"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

var _ fs.Node = &Node{}               // Attr
var _ fs.NodeStringLookuper = &Node{} // Lookup
var _ fs.HandleReadDirAller = &Node{} // HandleReadDirAller
var _ fs.NodeMkdirer = &Node{}        // Mkdir
var _ fs.NodeCreater = &Node{}        // Create
var _ fs.NodeRemover = &Node{}        // Remove
var _ fs.HandleWriter = &Node{}       // Write
var _ fs.HandleReadAller = &Node{}    // ReadAll
var _ fs.HandleReader = &Node{}       // Read
var _ fs.NodeReadlinker = &Node{}     // Readlink

var _ fs.NodeFsyncer = &Node{} // Fsync (vi) TBD
var _ fs.NodeRenamer = &Node{} // Rename     TBD
//var _ fs.NodeSetattrer = &Node{} // Setattr (touch)
//var _ fs.NodeSymlinker = &Node{} // Symlink

// Default permissions: we don't have any right now.
const defaultDirPerms = 0775
const defaultFilePerms = 0644

// Maximum file size.
const maxSize = math.MaxUint64

// Maximum length of a symlink target.
const maxSymlinkTargetLength = 4096

type Node struct {
	mdbfs MDBFS

	// Path to this node
	Path string
	// basename of this node
	Name string

	// ID is a unique ID allocated at node creation time.
	ID uint64
	// Used for type only, permissions are ignored.
	Mode os.FileMode
	// SymlinkTarget is the path a symlink points to.
	SymlinkTarget string

	// Other fields to add:
	// nLinks: number of hard links
	// openFDs: number of open file descriptors
	// timestamps (probably just ctime and mtime)
	// mode bits: permissions

	// Data blocks are addressed by inode number and offset.
	// Any op accessing Size and blocks must lock 'mu'.
	mu   sync.RWMutex
	Size uint64

	// Data is data stord in this node.
	Data []uint8
}

// convenience functions to query the mode.
func (n *Node) isDir() bool {
	return n.Mode.IsDir()
}

func (n *Node) isRegular() bool {
	return n.Mode.IsRegular()
}

func (n *Node) isSymlink() bool {
	return n.Mode&os.ModeSymlink != 0
}

// Attr fills attr with the standard metadata for the node.
func (n *Node) Attr(_ context.Context, a *fuse.Attr) error {
	log.Printf("[DEBUG] Fsync: node: %#v \t attr %#v", n, a)
	a.Inode = n.ID
	a.Mode = n.Mode

	if n.isRegular() {
		n.mu.RLock()
		defer n.mu.RUnlock()
		a.Size = n.Size
	} else if n.isSymlink() {
		// 	// Symlink: use target name length.
		a.Size = uint64(len(n.SymlinkTarget))
	}
	return nil
}

// Lookup looks up a specific entry in the receiver,
// which must be a directory.  Lookup should return a Node
// corresponding to the entry.  If the name does not exist in
// the directory, Lookup should return ENOENT.
//
// Lookup need not to handle the names "." and "..".
func (n *Node) Lookup(_ context.Context, name string) (fs.Node, error) {
	log.Printf("[DEBUG] Lookup: node: %#v", n)
	if !n.isDir() {
		return nil, fuse.Errno(syscall.ENOTDIR)
	}
	node, err := n.mdbfs.lookup(n, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fuse.ENOENT
		}
		return nil, err
	}
	node.mdbfs = n.mdbfs
	return node, nil
}

// ReadDirAll returns the list of child inodes.
func (n *Node) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	log.Printf("[DEBUG] ReadDirAll: node: %#v", n)
	if !n.isDir() {
		return nil, fuse.Errno(syscall.ENOTDIR)
	}
	return n.mdbfs.list(n)
}

// Mkdir creates a directory in 'n'.
// We let the sql query fail if the directory already exists.
// TODO: better handling of errors.
func (n *Node) Mkdir(_ context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	log.Printf("[DEBUG] Mkdir: node: %#v \t req: %#v", n, req)
	if !n.isDir() {
		return nil, fuse.Errno(syscall.ENOTDIR)
	}
	if !req.Mode.IsDir() {
		return nil, fuse.Errno(syscall.ENOTDIR)
	}
	node := n.mdbfs.newDirNode(n.Path, req.Name)
	node, err := n.mdbfs.create(node)
	if err != nil {
		return nil, err
	}
	return node, nil
}

// Create creates a new file in the receiver directory.
func (n *Node) Create(_ context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (
	fs.Node, fs.Handle, error) {
	log.Printf("[DEBUG] Create: node: %#v \t req: %#v", n, req)
	if !n.isDir() {
		return nil, nil, fuse.Errno(syscall.ENOTDIR)
	}
	if req.Mode.IsDir() {
		return nil, nil, fuse.Errno(syscall.EISDIR)
	} else if !req.Mode.IsRegular() {
		return nil, nil, fuse.Errno(syscall.EINVAL)
	}
	node := n.mdbfs.newFileNode(n.Path, req.Name)
	node, err := n.mdbfs.create(node)
	if err != nil {
		return nil, nil, err
	}
	return node, node, nil
}

// Remove may be unlink or rmdir.
func (n *Node) Remove(_ context.Context, req *fuse.RemoveRequest) error {
	log.Printf("[DEBUG] Remove: node: %#v \t req: %#v", n, req)
	if !n.Mode.IsDir() {
		return fuse.Errno(syscall.ENOTDIR)
	}

	if req.Dir {
		// Rmdir.
		return n.mdbfs.remove(n.Path, req.Name, true /* checkChildren */)
	}
	// Unlink.
	return n.mdbfs.remove(n.Path, req.Name, false /* !checkChildren */)
}

// Write writes data to node. It may overwrite existing data, or grow it.
func (n *Node) Write(_ context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	log.Printf("[DEBUG] Write: node: %#v \t req: %#v \t resp: %#v", n, req, resp)
	if !n.Mode.IsRegular() {
		return fuse.Errno(syscall.EINVAL)
	}
	if req.Offset < 0 {
		return fuse.Errno(syscall.EINVAL)
	}
	if len(req.Data) == 0 {
		return nil
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.Data = req.Data
	n.Size = uint64(len(req.Data))

	txn := n.mdbfs.db.Txn(true)
	defer txn.Abort()
	err := txn.Insert("Node", n)
	if err != nil {
		return err
	}
	txn.Commit()

	// We always write everything.
	resp.Size = len(req.Data)
	return nil
}

func (n *Node) ReadAll(ctx context.Context) ([]byte, error) {
	log.Printf("[DEBUG] ReadAll: node: %#v", n)
	txn := n.mdbfs.db.Txn(false)
	defer txn.Abort()
	raw, err := txn.First("Node", "id", n.Path)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		// TODO
	}
	return raw.(*Node).Data, nil
}

// Read reads data from 'n'.
func (n *Node) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	log.Printf("[DEBUG] Read: node: %#v \t req: %#v \t resp: %#v", n, req, resp)
	if !n.Mode.IsRegular() {
		return fuse.Errno(syscall.EINVAL)
	}
	if req.Offset < 0 {
		// Before beginning of file.
		return fuse.Errno(syscall.EINVAL)
	}
	if req.Size == 0 {
		// No bytes requested.
		return nil
	}

	txn := n.mdbfs.db.Txn(false)
	defer txn.Abort()
	raw, err := txn.First("Node", "id", n.Path)
	if err != nil {
		return err
	}
	resp.Data = raw.(*Node).Data
	return nil
}

func (n *Node) Fsync(_ context.Context, req *fuse.FsyncRequest) error {
	log.Printf("[DEBUG] Fsync: node: %#v \t req %#v", n, req)
	return nil
}

/*
func (n *Node) Setattr(_ context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	return fmt.Errorf("not implemented yet")
}
*/

// Rename renames old name to new name. (not implemented yet)
func (n *Node) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	log.Printf("[DEBUG] Rename: node: %#v \t req: %#v \t newDir: %#v", n, req, newDir)
	// TODO: search old Node -> delete old Node-> inert new Node
	return fmt.Errorf("not implemented yet")
}

// Readlink reads a symbolic link. (not supported yet)
func (n *Node) Readlink(_ context.Context, req *fuse.ReadlinkRequest) (string, error) {
	log.Printf("[DEBUG] Readlink: node: %#v \t req: %#v", n, req)
	if !n.isSymlink() {
		return "", fuse.Errno(syscall.EINVAL)
	}
	return n.SymlinkTarget, nil
}
