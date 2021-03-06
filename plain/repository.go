package plain

import (
	"github.com/src-d/go-borges"
	"github.com/src-d/go-borges/util"

	"gopkg.in/src-d/go-billy.v4"
	butil "gopkg.in/src-d/go-billy.v4/util"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/storage"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
	"gopkg.in/src-d/go-git.v4/storage/transactional"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

// Repository represents a git plain repository.
type Repository struct {
	id           borges.RepositoryID
	l            *Location
	mode         borges.Mode
	temporalPath string
	fs           billy.Filesystem

	*git.Repository
}

func initRepository(l *Location, id borges.RepositoryID) (*Repository, error) {
	s, fs, tempPath, err := repositoryStorer(l, id, borges.RWMode)
	if err != nil {
		return nil, err
	}

	r, err := git.Init(s, nil)
	if err != nil {
		return nil, err
	}

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{id.String()},
	})

	if err != nil {
		return nil, err
	}

	return &Repository{
		id:           id,
		l:            l,
		mode:         borges.RWMode,
		temporalPath: tempPath,
		fs:           fs,
		Repository:   r,
	}, nil
}

// openRepository, is the basic operation of open a repository without any checking.
func openRepository(l *Location, id borges.RepositoryID, mode borges.Mode) (*Repository, error) {
	s, fs, tempPath, err := repositoryStorer(l, id, mode)
	if err != nil {
		return nil, err
	}

	r, err := git.Open(s, nil)
	if err != nil {
		return nil, err
	}

	return &Repository{
		id:           id,
		l:            l,
		mode:         mode,
		temporalPath: tempPath,
		fs:           fs,
		Repository:   r,
	}, nil
}

func repositoryStorer(
	l *Location,
	id borges.RepositoryID,
	mode borges.Mode,
) (s storage.Storer, fs billy.Filesystem, tempPath string, err error) {
	fs, err = l.fs.Chroot(l.RepositoryPath(id))
	if err != nil {
		return nil, nil, "", err
	}

	c := l.opts.Cache
	if c == nil {
		c = cache.NewObjectLRUDefault()
	}

	opts := filesystem.Options{
		ExclusiveAccess: l.opts.Performance,
		KeepDescriptors: l.opts.Performance,
	}

	s = filesystem.NewStorageWithOptions(fs, c, opts)

	switch mode {
	case borges.ReadOnlyMode:
		return &util.ReadOnlyStorer{Storer: s}, fs, "", nil
	case borges.RWMode:
		if l.opts.Transactional {
			return repositoryTemporalStorer(l, id, s)
		}

		return s, fs, "", nil
	default:
		return nil, nil, "", borges.ErrModeNotSupported.New(mode)
	}
}

func repositoryTemporalStorer(
	l *Location,
	id borges.RepositoryID,
	parent storage.Storer,
) (s storage.Storer, fs billy.Filesystem, tempPath string, err error) {
	tempPath, err = butil.TempDir(l.opts.TemporalFilesystem, "transactions", "")
	if err != nil {
		return nil, nil, "", err
	}

	fs, err = l.opts.TemporalFilesystem.Chroot(tempPath)
	if err != nil {
		return nil, nil, "", err
	}

	ts := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	s = transactional.NewStorage(parent, ts)

	return
}

// R returns the git.Repository.
func (r *Repository) R() *git.Repository {
	return r.Repository
}

// ID returns the RepositoryID.
func (r *Repository) ID() borges.RepositoryID {
	return r.id
}

// Location implements the borges.Repository interface.
func (r *Repository) Location() borges.Location {
	return r.l
}

// Mode returns the Mode how it was opened.
func (r *Repository) Mode() borges.Mode {
	return r.mode
}

// Close closes the repository, if the repository was opened in transactional
// Mode, will delete any write operation pending to be written.
func (r *Repository) Close() error {
	if !r.l.opts.Transactional {
		return nil
	}

	return r.cleanupTemporal()
}

func (r *Repository) cleanupTemporal() error {
	return butil.RemoveAll(r.l.opts.TemporalFilesystem, r.temporalPath)
}

// Commit persists all the write operations done since was open, if the
// repository wasn't opened in a Location with Transactions enable returns
// ErrNonTransactional.
func (r *Repository) Commit() (err error) {
	if !r.l.opts.Transactional {
		return borges.ErrNonTransactional.New()
	}

	defer ioutil.CheckClose(r, &err)
	ts, ok := r.Storer.(transactional.Storage)
	if !ok {
		panic("unreachable code")
	}

	err = ts.Commit()
	return
}

// FS returns the filesystem to read or write directly to the repository or
// nil if not available.
func (r *Repository) FS() billy.Filesystem {
	return r.fs
}
