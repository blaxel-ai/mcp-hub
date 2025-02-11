package git

import (
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func CloneRepository(path string, branch string, url string) (*git.Repository, error) {
	return git.PlainClone(path, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
		Progress:      os.Stdout,
	})
}

func DeleteRepository(path string) error {
	return os.RemoveAll(path)
}
