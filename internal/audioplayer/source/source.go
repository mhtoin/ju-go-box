package source

import "io"

type Source interface {
	Stream(w io.Writer) error
	Stop() error
}
