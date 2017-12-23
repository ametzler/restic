package archiver

import (
	"fmt"
	"path/filepath"

	"github.com/restic/restic/internal/debug"
	"github.com/restic/restic/internal/errors"
)

// ArchiveTree defines how a snapshot should look like when archived.
type ArchiveTree struct {
	Nodes        map[string]ArchiveTree
	Path         string // where the files/dirs to be saved are found
	FileInfoPath string // where the dir can be found that is not included itself, but its subdirs
	Root         string // parent directory of the tree
}

// pathComponents returns all path components of p.
func pathComponents(p string, includeRelative bool) (components []string) {
	volume := filepath.VolumeName(p)

	if !filepath.IsAbs(p) {
		if !includeRelative {
			p = filepath.Join(string(filepath.Separator), p)
		}
	}

	p = filepath.Clean(p)

	for {
		dir, file := filepath.Dir(p), filepath.Base(p)

		if p == dir {
			break
		}

		components = append(components, file)
		p = dir
	}

	// reverse components
	for i := len(components)/2 - 1; i >= 0; i-- {
		opp := len(components) - 1 - i
		components[i], components[opp] = components[opp], components[i]
	}

	if volume != "" {
		// strip colon
		if len(volume) == 2 && volume[1] == ':' {
			volume = volume[:1]
		}

		components = append([]string{volume}, components...)
	}

	return components
}

// rootDirectory returns the directory which contains the first element of target.
func rootDirectory(target string) string {
	if target == "" {
		return ""
	}

	if filepath.IsAbs(target) {
		return filepath.Join(filepath.VolumeName(target), string(filepath.Separator))
	}

	target = filepath.Clean(target)
	pc := pathComponents(target, true)

	rel := "."
	for _, c := range pc {
		if c == ".." {
			rel = filepath.Join(rel, c)
		}
	}

	return rel
}

// Add adds a new target path into the tree.
func (t *ArchiveTree) Add(target string) error {
	debug.Log("%v (%v nodes)", target, len(t.Nodes))
	if target == "" {
		return errors.New("invalid target (empty string)")
	}

	if t.Nodes == nil {
		t.Nodes = make(map[string]ArchiveTree)
	}

	pc := pathComponents(target, false)
	if len(pc) == 0 {
		return errors.New("invalid target (no path components)")
	}

	name := pc[0]
	root := rootDirectory(target)
	tree := ArchiveTree{Root: root}

	origName := name
	i := 0
	for {
		other, ok := t.Nodes[name]
		if !ok {
			break
		}

		i++
		if other.Root == root {
			tree = other
			break
		}

		// resolve conflict and try again
		name = fmt.Sprintf("%s-%d", origName, i)
		continue
	}

	if len(pc) > 1 {
		subroot := filepath.Join(root, origName)
		err := tree.add(target, subroot, pc[1:])
		if err != nil {
			return err
		}
		tree.FileInfoPath = subroot
	} else {
		debug.Log("leaf node, nodes: %v", len(tree.Nodes))
		tree.Path = target
	}

	t.Nodes[name] = tree
	return nil
}

// add adds a new target path into the tree.
func (t *ArchiveTree) add(target, root string, pc []string) error {
	debug.Log("%v/%v", target, pc[0])

	debug.Log("subtree, path is %q", t.Path)
	if t.Path != "" {
		debug.Log("parent directory of %v already included", target)
	}

	if len(pc) == 0 {
		return errors.Errorf("invalid path %q", target)
	}

	if t.Nodes == nil {
		t.Nodes = make(map[string]ArchiveTree)
	}

	name := pc[0]

	if len(pc) == 1 {
		tree, ok := t.Nodes[name]

		if !ok {
			t.Nodes[name] = ArchiveTree{Path: target}
			return nil
		}

		if tree.Path != "" {
			return errors.Errorf("path is already set for target %v", target)
		}
		tree.Path = target
		t.Nodes[name] = tree
		return nil
	}

	tree := ArchiveTree{}
	if other, ok := t.Nodes[name]; ok {
		tree = other
	}

	subroot := filepath.Join(root, name)
	tree.FileInfoPath = subroot

	err := tree.add(target, subroot, pc[1:])
	if err != nil {
		return err
	}
	t.Nodes[name] = tree

	return nil
}

// NewArchiveTree creates an ArchiveTree from the target files/directories.
func NewArchiveTree(targets []string) (*ArchiveTree, error) {
	debug.Log("targets: %v", targets)
	tree := ArchiveTree{}
	seen := make(map[string]struct{})
	for _, target := range targets {
		target = filepath.Clean(target)

		// skip duplicate targets
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}

		err := tree.Add(target)
		if err != nil {
			return nil, err
		}
	}

	return &tree, nil
}
