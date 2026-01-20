package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// LogEntry represents a single log record.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

var (
	mu          sync.RWMutex
	logEntries  []LogEntry
	maxEntries  = 1000 // Keep last 1000 in memory
	maxFileSize = int64(5 * 1024 * 1024) // 5MB limit
	logFilePath string
	logFile     *os.File
	logChan     = make(chan LogEntry, 100)
	done        chan struct{}
	workerDone  chan struct{}
	subscribers = make(map[chan LogEntry]bool)
	subsMu      sync.RWMutex

	// Redaction regex for sk-scooter keys
	scooterKeyRegex = regexp.MustCompile(`sk-scooter-[a-zA-Z0-9]+`)
)

// Init initializes the logging system.
func Init(appDir string) error {
	mu.Lock()
	defer mu.Unlock()

	logDir := filepath.Join(appDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logFileName := fmt.Sprintf("%s MCP Scooter Log.log", time.Now().Format("20060102"))
	logFilePath = filepath.Join(logDir, logFileName)
	
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	logFile = f

	// Load existing logs if any (optional, but good for persistence)
	// For now, we just start fresh in memory but append to file

	done = make(chan struct{})
	workerDone = make(chan struct{})
	go logWorker()

	return nil
}

// AddLog adds a new log entry.
func AddLog(level, message string) {
	// Redact sensitive info
	message = scooterKeyRegex.ReplaceAllString(message, "sk-scooter-REDACTED")

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
	}

	mu.Lock()
	logEntries = append(logEntries, entry)
	if len(logEntries) > maxEntries {
		logEntries = logEntries[len(logEntries)-maxEntries:]
	}
	mu.Unlock()

	// Print to console for development visibility
	fmt.Printf("[%s] [%s] %s\n", entry.Timestamp, level, message)

	// Send to file worker
	select {
	case logChan <- entry:
	default:
		// Drop log if channel is full to avoid blocking
	}

	// Notify subscribers
	subsMu.RLock()
	for sub := range subscribers {
		select {
		case sub <- entry:
		default:
			// Drop if subscriber is slow
		}
	}
	subsMu.RUnlock()
}

// Subscribe returns a channel that receives new log entries.
func Subscribe() chan LogEntry {
	subsMu.Lock()
	defer subsMu.Unlock()
	ch := make(chan LogEntry, 100)
	subscribers[ch] = true
	return ch
}

// Unsubscribe removes a log subscriber.
func Unsubscribe(ch chan LogEntry) {
	subsMu.Lock()
	defer subsMu.Unlock()
	delete(subscribers, ch)
	close(ch)
}

// GetLogs returns all logs currently in memory.
func GetLogs() []LogEntry {
	mu.RLock()
	defer mu.RUnlock()
	
	// Return a copy
	res := make([]LogEntry, len(logEntries))
	copy(res, logEntries)
	return res
}

// ClearLogs wipes both memory and file logs.
func ClearLogs() error {
	mu.Lock()
	defer mu.Unlock()

	logEntries = []LogEntry{}
	
	if logFile != nil {
		logFile.Close()
	}

	// Truncate file
	f, err := os.OpenFile(logFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	logFile = f
	
	return nil
}

// GetLogFilePath returns the path to the log file.
func GetLogFilePath() string {
	mu.RLock()
	defer mu.RUnlock()
	return logFilePath
}

// Close flushes and closes the log file.
func Close() {
	if done != nil {
		close(done)
		if workerDone != nil {
			<-workerDone // Wait for worker to finish
		}
	}
	
	mu.Lock()
	defer mu.Unlock()
	
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

func logWorker() {
	defer close(workerDone)
	for {
		select {
		case entry := <-logChan:
			writeEntry(entry)
		case <-done:
			// Flush remaining logs
			for {
				select {
				case entry := <-logChan:
					writeEntry(entry)
				default:
					return
				}
			}
		}
	}
}

func writeEntry(entry LogEntry) {
	mu.Lock()
	defer mu.Unlock()
	
	f := logFile
	if f == nil {
		return
	}

	// Check file size and truncate if needed (simple circular buffer strategy)
	if info, err := f.Stat(); err == nil && info.Size() > maxFileSize {
		f.Close()
		// Re-open with truncate
		f, err = os.OpenFile(logFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			logFile = f
			// Log that we truncated
			truncateEntry := LogEntry{
				Timestamp: time.Now().Format(time.RFC3339),
				Level:     "INFO",
				Message:   "Log file reached 5MB limit and was truncated.",
			}
			data, _ := json.Marshal(truncateEntry)
			f.Write(data)
			f.Write([]byte("\n"))
		} else {
			return
		}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	f.Write(data)
	f.Write([]byte("\n"))
}
