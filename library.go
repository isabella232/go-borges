package borges

import (
	"path"
	"strings"

	"gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

var (
	ErrNotImplemented      = errors.NewKind("not implemented")
	ErrModeNotSupported    = errors.NewKind("repository mode %q not supported")
	ErrLocationNotExists   = errors.NewKind("location %s not exists")
	ErrRepositoryExists    = errors.NewKind("repository %s already exists")
	ErrRepositoryNotExists = errors.NewKind("repository %s not exists")
)

type Library interface {
	GetOrInit(RepositoryID) (Repository, error)
	Init(RepositoryID) (Repository, error)
	Has(RepositoryID) (bool, LocationID, error)
	Get(RepositoryID, Mode) (Repository, error)
	Repositories(Mode) (RepositoryIterator, error)

	Location(id LocationID) (Location, error)
	//Locations() (LocationIter, error)
}

type Mode int

const (
	RWMode Mode = iota
	TransactionalRWMode
	ReadOnlyMode
)

type LocationID string

func MustLocationID(id string) LocationID {
	return LocationID(id)
}

type Location interface {
	ID() LocationID
	GetOrInit(RepositoryID) (Repository, error)
	Init(RepositoryID) (Repository, error)
	Has(RepositoryID) (bool, error)
	Get(RepositoryID, Mode) (Repository, error)
	Repositories(Mode) (RepositoryIterator, error)
}

type RepositoryID string

// NewRepositoryID returns a new RepositoryID based on a given endpoint.
// Eg.: git@github.com:src-d/go-borges becomes github.com/src-d/go-borges.git
func NewRepositoryID(endpoint string) (RepositoryID, error) {
	e, err := transport.NewEndpoint(endpoint)
	if err != nil {
		return "", err
	}

	if !strings.HasSuffix(e.Path, ".git") {
		e.Path += ".git"
	}

	return RepositoryID(path.Join(e.Host, e.Path)), nil
}

func (id RepositoryID) String() string {
	return string(id)
}

type Repository interface {
	ID() RepositoryID
	LocationID() LocationID
	Mode() Mode
	Commit() error
	Rollback() error
	R() *git.Repository
}
