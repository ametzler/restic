package archiver

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPathComponents(t *testing.T) {
	var tests = []struct {
		p   string
		c   []string
		rel bool
		win bool
	}{
		{
			p: "/foo/bar/baz",
			c: []string{"foo", "bar", "baz"},
		},
		{
			p:   "/foo/bar/baz",
			c:   []string{"foo", "bar", "baz"},
			rel: true,
		},
		{
			p: "foo/bar/baz",
			c: []string{"foo", "bar", "baz"},
		},
		{
			p:   "foo/bar/baz",
			c:   []string{"foo", "bar", "baz"},
			rel: true,
		},
		{
			p: "../foo/bar/baz",
			c: []string{"foo", "bar", "baz"},
		},
		{
			p:   "../foo/bar/baz",
			c:   []string{"..", "foo", "bar", "baz"},
			rel: true,
		},
		{
			p:   "c:/foo/bar/baz",
			c:   []string{"c", "foo", "bar", "baz"},
			rel: true,
			win: true,
		},
		{
			p:   "c:/foo/../bar/baz",
			c:   []string{"c", "bar", "baz"},
			win: true,
		},
		{
			p:   "c:/foo/../bar/baz",
			c:   []string{"c", "bar", "baz"},
			rel: true,
			win: true,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if test.win && runtime.GOOS != "windows" {
				t.Skip("skip test on unix")
			}

			c := pathComponents(filepath.FromSlash(test.p), test.rel)
			if !cmp.Equal(test.c, c) {
				t.Error(test.c, c)
			}
		})
	}
}

func TestRootDirectory(t *testing.T) {
	var tests = []struct {
		target string
		root   string
		unix   bool
		win    bool
	}{
		{target: ".", root: "."},
		{target: "foo/bar/baz", root: "."},
		{target: "../foo/bar/baz", root: ".."},
		{target: "..", root: ".."},
		{target: "../../..", root: "../../.."},
		{target: "/home/foo", root: "/", unix: true},
		{target: "c:/home/foo", root: "c:/", win: true},
		{target: "//host/share/foo", root: "//host/share/", win: true},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if test.unix && runtime.GOOS == "windows" {
				t.Skip("skip test on windows")
			}
			if test.win && runtime.GOOS != "windows" {
				t.Skip("skip test on unix")
			}

			root := rootDirectory(filepath.FromSlash(test.target))
			want := filepath.FromSlash(test.root)
			if root != want {
				t.Fatalf("wrong root directory, want %v, got %v", want, root)
			}
		})
	}
}

func TestNewArchiveTree(t *testing.T) {
	var tests = []struct {
		targets   []string
		want      ArchiveTree
		unix      bool
		mustError bool
	}{
		{
			targets: []string{"foo"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"foo": ArchiveTree{Path: "foo", Root: "."},
			}},
		},
		{
			targets: []string{"foo", "bar", "baz"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"foo": ArchiveTree{Path: "foo", Root: "."},
				"bar": ArchiveTree{Path: "bar", Root: "."},
				"baz": ArchiveTree{Path: "baz", Root: "."},
			}},
		},
		{
			targets: []string{"foo/user1", "foo/user2", "foo/other"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"foo": ArchiveTree{Root: ".", Nodes: map[string]ArchiveTree{
					"user1": ArchiveTree{Path: "foo/user1"},
					"user2": ArchiveTree{Path: "foo/user2"},
					"other": ArchiveTree{Path: "foo/other"},
				}},
			}},
		},
		{
			targets: []string{"foo/work/user1", "foo/work/user2"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"foo": ArchiveTree{Root: ".", Nodes: map[string]ArchiveTree{
					"work": ArchiveTree{Nodes: map[string]ArchiveTree{
						"user1": ArchiveTree{Path: "foo/work/user1"},
						"user2": ArchiveTree{Path: "foo/work/user2"},
					}},
				}},
			}},
		},
		{
			targets: []string{"foo/user1", "bar/user1", "foo/other"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"foo": ArchiveTree{Root: ".", Nodes: map[string]ArchiveTree{
					"user1": ArchiveTree{Path: "foo/user1"},
					"other": ArchiveTree{Path: "foo/other"},
				}},
				"bar": ArchiveTree{Root: ".", Nodes: map[string]ArchiveTree{
					"user1": ArchiveTree{Path: "bar/user1"},
				}},
			}},
		},
		{
			targets: []string{"foo/user1", "../work/other", "foo/user2"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"foo": ArchiveTree{Root: ".", Nodes: map[string]ArchiveTree{
					"user1": ArchiveTree{Path: "foo/user1"},
					"user2": ArchiveTree{Path: "foo/user2"},
				}},
				"work": ArchiveTree{Root: "..", Nodes: map[string]ArchiveTree{
					"other": ArchiveTree{Path: "../work/other"},
				}},
			}},
		},
		{
			targets: []string{"foo/user1", "../foo/other", "foo/user2"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"foo": ArchiveTree{Root: ".", Nodes: map[string]ArchiveTree{
					"user1": ArchiveTree{Path: "foo/user1"},
					"user2": ArchiveTree{Path: "foo/user2"},
				}},
				"foo-1": ArchiveTree{Root: "..", Nodes: map[string]ArchiveTree{
					"other": ArchiveTree{Path: "../foo/other"},
				}},
			}},
		},
		{
			targets: []string{"foo/work", "foo/work/user2"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"foo": ArchiveTree{Root: ".", Nodes: map[string]ArchiveTree{
					"work": ArchiveTree{
						Path: "foo/work",
						Nodes: map[string]ArchiveTree{
							"user2": ArchiveTree{Path: "foo/work/user2"},
						},
					},
				}},
			}},
		},
		{
			targets: []string{"foo/work/user2", "foo/work"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"foo": ArchiveTree{Root: ".", Nodes: map[string]ArchiveTree{
					"work": ArchiveTree{
						Path: "foo/work",
						Nodes: map[string]ArchiveTree{
							"user2": ArchiveTree{Path: "foo/work/user2"},
						},
					},
				}},
			}},
		},
		{
			unix:    true,
			targets: []string{"/mnt/driveA", "/mnt/driveA/work/driveB"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"mnt": ArchiveTree{Root: "/", Nodes: map[string]ArchiveTree{
					"driveA": ArchiveTree{
						Path: "/mnt/driveA",
						Nodes: map[string]ArchiveTree{
							"work": ArchiveTree{Nodes: map[string]ArchiveTree{
								"driveB": ArchiveTree{Path: "/mnt/driveA/work/driveB"},
							}},
						},
					},
				}},
			}},
		},
		{
			targets: []string{"foo/work/user", "foo/work/user"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"foo": ArchiveTree{Root: ".", Nodes: map[string]ArchiveTree{
					"work": ArchiveTree{Nodes: map[string]ArchiveTree{
						"user": ArchiveTree{Path: "foo/work/user"},
					}},
				}},
			}},
		},
		{
			targets: []string{"./foo/work/user", "foo/work/user"},
			want: ArchiveTree{Nodes: map[string]ArchiveTree{
				"foo": ArchiveTree{Root: ".", Nodes: map[string]ArchiveTree{
					"work": ArchiveTree{Nodes: map[string]ArchiveTree{
						"user": ArchiveTree{Path: "foo/work/user"},
					}},
				}},
			}},
		},
		{
			targets:   []string{"."},
			mustError: true,
		},
		{
			targets:   []string{".."},
			mustError: true,
		},
		{
			targets:   []string{"../.."},
			mustError: true,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if test.unix && runtime.GOOS == "windows" {
				t.Skip("skip test on windows")
			}

			tree, err := NewArchiveTree(test.targets)
			if test.mustError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				t.Logf("found expected error: %v", err)
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if !cmp.Equal(&test.want, tree) {
				t.Error(cmp.Diff(&test.want, tree))
			}
		})
	}
}
