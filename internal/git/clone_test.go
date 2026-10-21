package git

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// testSignature returns a deterministic commit signature for test fixtures.
func testSignature() object.Signature {
	return object.Signature{
		Name:  "mcp-hub-test",
		Email: "test@example.com",
		When:  time.Unix(1700000000, 0),
	}
}

// writeBlob stores data as a blob object and returns its hash.
func writeBlob(t *testing.T, s storer.EncodedObjectStorer, data []byte) plumbing.Hash {
	t.Helper()
	obj := s.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	w, err := obj.Writer()
	if err != nil {
		t.Fatalf("blob writer: %v", err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("blob write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("blob close: %v", err)
	}
	hash, err := s.SetEncodedObject(obj)
	if err != nil {
		t.Fatalf("store blob: %v", err)
	}
	return hash
}

// storeRawTree encodes and stores a tree object built from the given entries.
func storeRawTree(t *testing.T, s storer.EncodedObjectStorer, entries []object.TreeEntry) plumbing.Hash {
	t.Helper()
	sort.Sort(object.TreeEntrySorter(entries))
	tree := &object.Tree{Entries: entries}
	obj := s.NewEncodedObject()
	if err := tree.Encode(obj); err != nil {
		t.Fatalf("encode tree: %v", err)
	}
	hash, err := s.SetEncodedObject(obj)
	if err != nil {
		t.Fatalf("store tree: %v", err)
	}
	return hash
}

// buildMaliciousCommit builds a commit on top of parent whose tree adds a
// single nested file at filePath (e.g. "s/config"), containing "exploit".
// Intermediate path components become real tree (directory) entries — the
// crafted tree never contains a symlink itself. The symlink is planted
// separately, directly on disk, exactly as CVE-2026-71556 / GHSA-hc8v-wwc9-vgxm
// describes: a symlink already present in the worktree, followed by a later
// worktree operation whose path string is innocent but resolves through it.
func buildMaliciousCommit(t *testing.T, s storer.Storer, parent *object.Commit, parentHash plumbing.Hash, filePath string) plumbing.Hash {
	t.Helper()

	leafHash := writeBlob(t, s, []byte("exploit"))
	mode := filemode.Regular

	parts := strings.Split(filePath, "/")
	for i := len(parts) - 1; i >= 1; i-- {
		entry := object.TreeEntry{Name: parts[i], Mode: mode, Hash: leafHash}
		leafHash = storeRawTree(t, s, []object.TreeEntry{entry})
		mode = filemode.Dir
	}

	parentTree, err := parent.Tree()
	if err != nil {
		t.Fatalf("parent tree: %v", err)
	}
	entries := make([]object.TreeEntry, len(parentTree.Entries), len(parentTree.Entries)+1)
	copy(entries, parentTree.Entries)
	entries = append(entries, object.TreeEntry{Name: parts[0], Mode: mode, Hash: leafHash})
	rootHash := storeRawTree(t, s, entries)

	sig := testSignature()
	commit := &object.Commit{
		Author:       sig,
		Committer:    sig,
		Message:      "attack: write through masking symlink " + filePath + "\n",
		TreeHash:     rootHash,
		ParentHashes: []plumbing.Hash{parentHash},
	}
	obj := s.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		t.Fatalf("encode commit: %v", err)
	}
	commitHash, err := s.SetEncodedObject(obj)
	if err != nil {
		t.Fatalf("store commit: %v", err)
	}
	return commitHash
}

// newSourceRepo creates a plain git repository on disk with a single commit
// (a README file) and returns its directory and default branch name.
func newSourceRepo(t *testing.T) (dir string, branch string) {
	t.Helper()
	dir = t.TempDir()

	r, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init source repo: %v", err)
	}
	w, err := r.Worktree()
	if err != nil {
		t.Fatalf("source worktree: %v", err)
	}

	f, err := w.Filesystem.Create("README")
	if err != nil {
		t.Fatalf("create README: %v", err)
	}
	if _, err := f.Write([]byte("hello\n")); err != nil {
		t.Fatalf("write README: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close README: %v", err)
	}
	if _, err := w.Add("README"); err != nil {
		t.Fatalf("add README: %v", err)
	}
	sig := testSignature()
	if _, err := w.Commit("initial commit\n", &gogit.CommitOptions{Author: &sig}); err != nil {
		t.Fatalf("initial commit: %v", err)
	}

	head, err := r.Head()
	if err != nil {
		t.Fatalf("source head: %v", err)
	}
	return dir, head.Name().Short()
}

// TestCloneRepository_ClonesSuccessfully proves the go-git 5.19.2 bump did
// not break basic cloning, the only thing CloneRepository is asked to do in
// production (hub imports run this exact function against real MCP server
// repositories).
func TestCloneRepository_ClonesSuccessfully(t *testing.T) {
	srcDir, branch := newSourceRepo(t)
	dstDir := filepath.Join(t.TempDir(), "clone")

	repo, err := CloneRepository(dstDir, branch, srcDir)
	if err != nil {
		t.Fatalf("CloneRepository: %v", err)
	}
	if repo == nil {
		t.Fatal("CloneRepository returned a nil repository")
	}

	data, err := os.ReadFile(filepath.Join(dstDir, "README"))
	if err != nil {
		t.Fatalf("read cloned README: %v", err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("unexpected README content: %q", data)
	}

	if err := DeleteRepository(dstDir); err != nil {
		t.Fatalf("DeleteRepository: %v", err)
	}
	if _, err := os.Stat(dstDir); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, stat err = %v", dstDir, err)
	}
}

// TestCloneRepository_RejectsSymlinkTraversalOnCheckout targets the attack
// class behind GHSA-hc8v-wwc9-vgxm / CVE-2026-71556 ("go-git: Worktree
// operations may follow symlinks", fixed in go-git v5.19.2) through
// mcp-hub's actual production entry point, CloneRepository: plant a
// masking symlink directly in a cloned worktree, then force-checkout a
// commit that writes a nested, innocent-looking path ("s/config") through
// it, and confirm the write lands inside the worktree rather than through
// the symlink into the repository's real .git.
//
// IMPORTANT, read before trusting this as proof of the version bump: this
// exact scenario, and the two-level-nested variant ("a/b/s/config"), were
// verified NOT to discriminate go-git v5.19.1 from v5.19.2 for this call
// pattern. Both versions already defeat it, and for a reason unrelated to
// the v5.19.2 fix: go-git's Reset/Checkout path does a full worktree-vs-
// target diff (diffStagingWithWorktree) before writing, and Lstat-ing the
// planted symlink there finds a type mismatch against the target tree (a
// leaf where the target expects a directory), which merkletrie reports as
// Delete("s") + Insert("s/config") rather than a bare Insert. The Delete
// removes the symlink before the Insert ever calls checkoutFile, so the
// v5.19.1-only code path this test was meant to catch (checkoutFile's
// unguarded OpenFile/MkdirAll, the thing v5.19.2's clearBlockingSymlinks
// exists to guard) is never reached for a single Worktree.Checkout call —
// verified by disassembling the v5.19.1 test binary (`go tool nm`) to
// confirm clearBlockingSymlinks/validNoLeadingSymlink are genuinely absent
// from the linked code, not a stale build.
//
// So this test does not prove the v5.19.2 bump changed CloneRepository's
// behavior — it doesn't, for this call pattern. It is kept anyway as a
// general regression guard (it would fail if a *future* go-git version, or
// a switch to a different Worktree entry point such as Add/Remove/
// submodule updates, reopened this class of bug), and because upstream
// go-git's own dedicated test suite for this advisory
// (TestForceCheckoutReplacesLeadingSymlink and
// TestWorktreeFilesystemRejectsSymlinkTraversal in go-git/go-git's
// worktree_fs_test.go) is the actual proof the fix works; reproducing
// *that* would require exercising go-git's internal, unexported
// worktreeFilesystem type, which is not reachable from mcp-hub as a
// library consumer.
func TestCloneRepository_RejectsSymlinkTraversalOnCheckout(t *testing.T) {
	srcDir, branch := newSourceRepo(t)
	dstDir := filepath.Join(t.TempDir(), "clone")

	repo, err := CloneRepository(dstDir, branch, srcDir)
	if err != nil {
		t.Fatalf("CloneRepository: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("head commit: %v", err)
	}

	gitConfigPath := filepath.Join(dstDir, ".git", "config")
	before, err := os.ReadFile(gitConfigPath)
	if err != nil {
		t.Fatalf("read .git/config before checkout: %v", err)
	}

	// Plant the masking symlink directly on disk, as an attacker (or a
	// leftover from an earlier run) would. "s" -> ".git" resolves, from the
	// worktree root, to dstDir/.git.
	if err := w.Filesystem.Symlink(".git", "s"); err != nil {
		t.Fatalf("plant masking symlink: %v", err)
	}

	// The attack commit writes "s/config" — an entirely ordinary nested
	// file from git's point of view.
	attackHash := buildMaliciousCommit(t, repo.Storer, headCommit, head.Hash(), "s/config")

	if err := w.Checkout(&gogit.CheckoutOptions{Hash: attackHash, Force: true}); err != nil {
		t.Fatalf("force checkout of attack commit: %v", err)
	}

	after, err := os.ReadFile(gitConfigPath)
	if err != nil {
		t.Fatalf("read .git/config after checkout: %v", err)
	}
	if bytes.Contains(after, []byte("exploit")) || !bytes.Equal(before, after) {
		t.Fatalf(".git/config was modified by checkout through the masking symlink: before=%q after=%q", before, after)
	}

	// The fix replaces the blocking leading symlink with a real directory
	// and writes the tracked file safely inside the worktree instead of
	// erroring outright — matching plain git's own force-checkout
	// behaviour (create_directories + unlink-before-write).
	fi, err := os.Lstat(filepath.Join(dstDir, "s"))
	if err != nil {
		t.Fatalf("lstat s: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("leading symlink \"s\" was not replaced by a real directory")
	}
	data, err := os.ReadFile(filepath.Join(dstDir, "s", "config"))
	if err != nil {
		t.Fatalf("read s/config: %v", err)
	}
	if string(data) != "exploit" {
		t.Fatalf("expected checked-out content inside the worktree, got %q", data)
	}
}
