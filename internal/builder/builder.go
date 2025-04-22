package builder

type Build struct {
	tag      string
	debug    bool
	registry string
}

const (
	tmpDir       = "tmp"
	githubPrefix = "https://github.com/"
	dockerfile   = "Dockerfile"
)

func NewBuild(tag string, registry string, debug bool) *Build {
	buildInstance := &Build{
		tag:      tag,
		debug:    debug,
		registry: registry,
	}
	buildInstance.Clean()
	return buildInstance
}
