package libraries

import (
	"context"
	"time"

	"github.com/src-d/go-borges"
	"github.com/src-d/go-borges/util"

	"gopkg.in/src-d/go-errors.v1"
)

var (
	// ErrLibraryExists an error returned when a borges.Library
	// added before is attempted to be added again.
	ErrLibraryExists = errors.NewKind("library %s already exists")
)

// FilterLibraryFunc stands for a borges.Library filter function.
type FilterLibraryFunc func(borges.Library) (bool, error)

// RepositoryIterFunc stands for a function returning a
// borges.RepositoryIterator which iters in a certain order.
type RepositoryIterFunc func(*Libraries, borges.Mode) (borges.RepositoryIterator, error)

// Options hold configuration options for a Libraries.
type Options struct {
	// Timeout set a timeout for library operations. Some operations could
	// potentially take long so timing out them will make an error be
	// returned. A 0 value sets a default value of 60 seconds.
	Timeout             time.Duration
	RepositoryIterOrder RepositoryIterFunc
}

// Libraries is an implementation to aggregate borges.Library in just one instance.
// The borges.Library that will be added shouldn't contain other libraries inside.
type Libraries struct {
	libs map[borges.LibraryID]borges.Library
	opts *Options
}

var _ borges.Library = (*Libraries)(nil)

const (
	timeout = 60 * time.Second
)

// New create a new Libraries instance.
func New(options *Options) *Libraries {
	var opts *Options
	if options == nil {
		opts = &Options{}
	} else {
		opts = &(*options)
	}

	if opts.RepositoryIterOrder == nil {
		opts.RepositoryIterOrder = RepositoryDefaultIter
	}

	if opts.Timeout == 0 {
		opts.Timeout = timeout
	}

	return &Libraries{
		libs: map[borges.LibraryID]borges.Library{},
		opts: opts,
	}
}

// Add adds a new borges.Library. It will fail with ErrLibraryExists
// if the library was already added.
func (l *Libraries) Add(lib borges.Library) error {
	_, ok := l.libs[lib.ID()]
	if ok {
		return ErrLibraryExists.New(lib.ID())
	}

	l.libs[lib.ID()] = lib
	return nil
}

// ID implements the Library interface.
func (l *Libraries) ID() borges.LibraryID {
	return ""
}

// Init implements the Library interface.
func (l *Libraries) Init(borges.RepositoryID) (borges.Repository, error) {
	return nil, borges.ErrNotImplemented.New()
}

// Get implements the Library interface.
func (l *Libraries) Get(id borges.RepositoryID, mode borges.Mode) (borges.Repository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.opts.Timeout)
	defer cancel()

	for _, lib := range l.libs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		r, err := lib.Get(id, mode)
		if err != nil {
			if borges.ErrRepositoryNotExists.Is(err) {
				continue
			}

			return nil, err
		}

		return r, nil
	}

	return nil, borges.ErrRepositoryNotExists.New(id)
}

// GetOrInit implements the Library interface.
func (l *Libraries) GetOrInit(borges.RepositoryID) (borges.Repository, error) {
	return nil, borges.ErrNotImplemented.New()
}

// Has implements the Library interface.
func (l *Libraries) Has(id borges.RepositoryID) (bool, borges.LibraryID, borges.LocationID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.opts.Timeout)
	defer cancel()

	for _, lib := range l.libs {
		select {
		case <-ctx.Done():
			return false, "", "", ctx.Err()
		default:
		}

		has, libID, locID, err := lib.Has(id)
		if err != nil {
			return false, "", "", err
		}

		if has {
			return has, libID, locID, nil
		}
	}

	return false, "", "", nil
}

// Repositories implements the Library interface.
func (l *Libraries) Repositories(mode borges.Mode) (borges.RepositoryIterator, error) {
	return l.opts.RepositoryIterOrder(l, mode)
}

// Location implements the Library interface.
func (l *Libraries) Location(id borges.LocationID) (borges.Location, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.opts.Timeout)
	defer cancel()

	for _, lib := range l.libs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		loc, err := lib.Location(id)
		if err != nil {
			if borges.ErrLocationNotExists.Is(err) {
				continue
			}

			return nil, err
		}

		return loc, nil
	}

	return nil, borges.ErrLocationNotExists.New(id)
}

// Locations implements the Library interface.
func (l *Libraries) Locations() (borges.LocationIterator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.opts.Timeout)
	defer cancel()

	var locations []borges.LocationIterator
	for _, lib := range l.libs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		locs, err := lib.Locations()
		if err != nil {
			return nil, err
		}

		locations = append(locations, locs)
	}

	return MergeLocationIterators(locations), nil
}

// Library implements the Library interface.
func (l *Libraries) Library(id borges.LibraryID) (borges.Library, error) {
	lib, ok := l.libs[id]
	if !ok {
		return nil, borges.ErrLibraryNotExists.New(id)
	}

	return lib, nil
}

// Libraries implements the Library interface.
func (l *Libraries) Libraries() (borges.LibraryIterator, error) {
	return l.FilteredLibraries(func(borges.Library) (bool, error) {
		return true, nil
	})
}

// FilteredLibraries returns an iterator containing only those libraries
// accomplishing with the FilteredLibraryFunc function.
func (l *Libraries) FilteredLibraries(filter FilterLibraryFunc) (borges.LibraryIterator, error) {
	libs, err := l.libraries(filter)
	if err != nil {
		return nil, err
	}

	return util.NewLibraryIterator(libs), nil
}

func (l *Libraries) libraries(filter FilterLibraryFunc) ([]borges.Library, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.opts.Timeout)
	defer cancel()

	libs := make([]borges.Library, 0, len(l.libs))
	for _, lib := range l.libs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ok, err := filter(lib)
		if err != nil {
			return nil, err
		}

		if ok {
			libs = append(libs, lib)
		}
	}

	return libs, nil
}
