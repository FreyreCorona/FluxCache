package resp

import "io"

// Writer serializes and writes RESP values to an io.Writer.
type Writer struct {
	writer io.Writer
}

// NewWriter creates a new Writer that writes to the given io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

// Write serializes and writes a Value to the underlying writer.
func (w *Writer) Write(v Value) error {
	bytes := v.Marshal()
	_, err := w.writer.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}
