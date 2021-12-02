package logext

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// CycleWriter defines an io.Reader/io.Writer that will close and
// reopen (i.e. cycle) a file either programmatically or upon
// a given os.Signal.
type CycleWriter struct {
	lock     sync.Mutex
	filename string // should be set to the actual filename
	fp       *os.File
}

// NewCycleWriter makes a new CycleWriter.
// Return nil if error occurs during setup.
func NewCycleWriter(filename string) *CycleWriter {
	w := &CycleWriter{filename: filename}
	err := w.Cycle()
	if err != nil {
		return nil
	}
	return w
}

// Write satisfies the io.Writer interface.
func (w *CycleWriter) Write(output []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.fp.Write(output)
}

// Cycle performs the actual act of closing and reopening file.
func (w *CycleWriter) Cycle() (err error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// Close existing file if open
	if w.fp != nil {
		err = w.fp.Close()
		w.fp = nil
		if err != nil {
			return err
		}
	}

	// Open/Create a file.
	w.fp, err = os.OpenFile(w.filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	return err
}

// OnSignal registers the writer against a specific signal to cycle against
func (w *CycleWriter) OnSignal(sig syscall.Signal) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig)
	go func() {
		for range ch {
			log.Printf("received signal %v\n", sig)
			err := w.Cycle()
			if err != nil {
				log.SetOutput(os.Stdout)
				log.Println("could not cycle log file. using stdout.")
				signal.Reset(sig)
				return
			}
			log.Println("starting new file")
		}
	}()
}
