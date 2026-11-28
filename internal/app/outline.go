package app

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/cionz0/hdsort/internal/domain"
)

// OutlineNode is one visible line in the folder outline. A directory uses
// Path as its cleaned absolute path. A set uses RowKey to find its row.
type OutlineNode struct {
	Path      string
	Name      string
	Depth     int
	Directory bool
	RowKey    string
}

// DirectoryName is the label the outline draws for a folder. Inside the
// scan root that is the folder's own name. Outside it, the name is the
// cleaned absolute path.
func DirectoryName(root, folder string) string {
	root = filepath.Clean(root)
	folder = filepath.Clean(folder)
	if folder == root || domain.PathInside(root, folder) {
		return filepath.Base(folder)
	}
	return folder
}

// FolderTrashLabel is the accent button for the selected directory.
func FolderTrashLabel(name string) string {
	return "Trash " + name + " and its contents"
}

type outlineDir struct {
	path     string
	name     string
	children []*outlineDir
	sets     []FolderRow
}

// Outline returns the visible nodes. expanded holds the cleaned absolute
// paths of directories the owner has opened. A missing path is collapsed.
// The scan root is always the first place a set inside it can hang from.
func Outline(root string, rows []FolderRow, expanded []string) []OutlineNode {
	root = filepath.Clean(root)
	open := map[string]bool{}
	for _, path := range expanded {
		open[filepath.Clean(path)] = true
	}
	top := &outlineDir{path: root, name: DirectoryName(root, root)}
	var outside []*outlineDir
	byOutside := map[string]*outlineDir{}
	for _, row := range rows {
		folder := filepath.Clean(row.Folder)
		if folder == root || domain.PathInside(root, folder) {
			dir := top.add(root, folder)
			dir.sets = append(dir.sets, row)
			continue
		}
		node := byOutside[folder]
		if node == nil {
			node = &outlineDir{path: folder, name: DirectoryName(root, folder)}
			byOutside[folder] = node
			outside = append(outside, node)
		}
		node.sets = append(node.sets, row)
	}
	dirs := append([]*outlineDir{top}, outside...)
	sort.Slice(dirs, func(i, j int) bool {
		if dirs[i].name != dirs[j].name {
			return dirs[i].name < dirs[j].name
		}
		return dirs[i].path < dirs[j].path
	})
	var nodes []OutlineNode
	for _, dir := range dirs {
		dir.walk(0, open, &nodes)
	}
	return nodes
}

func (d *outlineDir) add(root, folder string) *outlineDir {
	if folder == root {
		return d
	}
	rel, err := filepath.Rel(root, folder)
	if err != nil || rel == "." {
		return d
	}
	cur := d
	curPath := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		curPath = filepath.Join(curPath, part)
		next := cur.find(curPath)
		if next == nil {
			next = &outlineDir{path: curPath, name: part}
			cur.children = append(cur.children, next)
		}
		cur = next
	}
	return cur
}

func (d *outlineDir) find(path string) *outlineDir {
	for _, child := range d.children {
		if child.path == path {
			return child
		}
	}
	return nil
}

type outlineSibling struct {
	name string
	dir  *outlineDir
	row  *FolderRow
}

func (d *outlineDir) walk(depth int, open map[string]bool, nodes *[]OutlineNode) {
	*nodes = append(*nodes, OutlineNode{
		Path:      d.path,
		Name:      d.name,
		Depth:     depth,
		Directory: true,
	})
	if !open[d.path] {
		return
	}
	kids := make([]outlineSibling, 0, len(d.children)+len(d.sets))
	for _, child := range d.children {
		kids = append(kids, outlineSibling{name: child.name, dir: child})
	}
	for i := range d.sets {
		kids = append(kids, outlineSibling{name: d.sets[i].Title, row: &d.sets[i]})
	}
	sort.Slice(kids, func(i, j int) bool {
		if kids[i].name != kids[j].name {
			return kids[i].name < kids[j].name
		}
		if (kids[i].dir != nil) != (kids[j].dir != nil) {
			return kids[i].dir != nil
		}
		if kids[i].row != nil && kids[j].row != nil {
			if kids[i].row.Title != kids[j].row.Title {
				return kids[i].row.Title < kids[j].row.Title
			}
			return kids[i].row.Key < kids[j].row.Key
		}
		return kids[i].dir.path < kids[j].dir.path
	})
	for _, kid := range kids {
		if kid.dir != nil {
			kid.dir.walk(depth+1, open, nodes)
			continue
		}
		*nodes = append(*nodes, OutlineNode{
			Path:   kid.row.Key,
			Name:   kid.row.Title,
			Depth:  depth + 1,
			RowKey: kid.row.Key,
		})
	}
}
