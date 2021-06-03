package worker

import (
	"errors"
	"io"
	"os"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

// watchFile watches `filename` for changes, sending a value every time a change is detected. The watcher is closed when the `done` channel receives input.
func watchFile(done <-chan struct{}, filename string) (<-chan struct{}, error) {
	logger := log.WithFields(log.Fields{"func": "WatchFile", "filename": filename})

	// Create a filesystem watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.WithError(err).Error("unable to initialize a new watcher")
		return nil, err
	}

	err = watcher.Add(filename)
	if err != nil {
		logger.WithError(err).Error("unable to watch file")

		wErr := watcher.Close()
		if wErr != nil {
			logger.WithError(wErr).Error("unable to close watcher")
		}

		return nil, err
	}

	// Initialize a channel to send a value to whenever the file contents change.
	fileChanged := make(chan struct{})

	go func() {
		defer close(fileChanged)
		defer watcher.Close()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op != fsnotify.Write {
					continue
				}
				// TODO: handle other types of events if needed

				logger.WithField("event", event).Debug("received filesystem write event")
				fileChanged <- struct{}{}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.WithError(err).Error("encountered a watcher error")

			case <-done:
				logger.Debug("done signal received")
				return
			}

		}
	}()

	return fileChanged, nil
}

// TailFollowFile reads and follows a local file, similar to `tail -f` but without log rotation or other advanced features, and outputs it to a channel. Tailing stops when `true` is received on `done`.
func TailFollowFile(done <-chan struct{}, filename string) (<-chan []byte, error) {
	logger := log.WithFields(log.Fields{"func": "TailFollowFile", "filename": filename})

	file, err := os.Open(filename)
	if err != nil {
		logger.WithError(err).Error("unable to open file for reading")
		return nil, err
	}

	// watch file for changes
	watchDone := make(chan struct{})
	fileChanged, err := watchFile(watchDone, filename)
	if err != nil {
		logger.Error("unable to monitor file changes")
		close(watchDone)
		return nil, err
	}

	fileContentsChan := make(chan []byte) // read file contents will be sent to this channel
	var seekPosition int64 = 0            // the first unread position of the file

	go func() {
		// housekeeping
		defer file.Close()
		defer close(fileContentsChan)
		defer close(watchDone)

	SelectLoop:
		for {
			select {
			case <-fileChanged:
				logger.Debug("received file change event")
				seekPosition, err = sendContentsUntilEOF(file, fileContentsChan, seekPosition)
				if err != nil {
					logger.WithError(err).Error("unable to send contents of file to channel")
					return
				}

			case <-done:
				logger.Debug("stopped following file since a done signal was received")
				break SelectLoop
			}
		}

		// done signal received. Let's just finish reading the file and exit
		_, err = sendContentsUntilEOF(file, fileContentsChan, seekPosition)
		if err != nil {
			logger.WithError(err).Error("unable to send contents of file to channel")
			return
		}
	}()

	return fileContentsChan, nil
}

// sendContentsUntilEOF reads from file until EOF is reached. Returns seek position.
func sendContentsUntilEOF(file *os.File, fileContentsChan chan<- []byte, seekPosition int64) (int64, error) {
	readBytes := make([]byte, 16*1024) // we choose a small buffer here for more realtime

	for {
		// load new content into buffer
		numBytes, err := file.ReadAt(readBytes, seekPosition)

		// even if there was an error, let's send whatever data we have
		if numBytes > 0 {
			seekPosition += int64(numBytes)
			fileContentsChan <- readBytes[:numBytes]
		}

		// handle error
		if err != nil {
			if errors.Is(err, io.EOF) {
				return seekPosition, nil
			} else {
				return seekPosition, err
			}
		}
	}
}
