package count_reader


import "io"

// countReader оборачивает io.ReadCloser и считает, сколько байт было прочитано.
type CountReader struct {
    // внутренняя реализация — оригинальный поток (например, c.Request.Body)
    rc io.ReadCloser
    // N хранит общее число прочитанных байт
    N int64
}

// Read читает из вложенного rc и увеличивает счётчик N
func (c *CountReader) Read(p []byte) (int, error) {
    n, err := c.rc.Read(p)
    c.N += int64(n)
    return n, err
}

// Close закрывает вложенный поток
func (c *CountReader) Close() error {
    return c.rc.Close()
}

// newCountReader создаёт countReader поверх заданного ReadCloser.
// пример cr := newCountReader(c.Request.Body)
func NewCountReader(rc io.ReadCloser) *CountReader {
    return &CountReader{rc: rc}
}