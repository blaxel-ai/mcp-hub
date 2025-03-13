package build

type Build struct {
	tag   string
	debug bool
}

const (
	tmpDir       = "tmp"
	githubPrefix = "https://github.com/"
	dockerfile   = "Dockerfile"
)

func NewBuild(tag string, debug bool) *Build {
	buildInstance := &Build{
		tag:   tag,
		debug: debug,
	}
	buildInstance.Clean()
	return buildInstance
}
