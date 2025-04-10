package processor

import "io"

type Processor interface {
	Process(r io.Reader, w io.Writer) error
	Stop() error
}
