package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
)

type LoggerLevel byte

// Définir les niveaux de log
const (
	LVL_TRACE    LoggerLevel = 0
	LVL_DEBUG    LoggerLevel = 1
	LVL_INFO     LoggerLevel = 2
	LVL_WARNING  LoggerLevel = 3
	LVL_ERROR    LoggerLevel = 4
	LVL_CRITICAL LoggerLevel = 5
)

var (
	currentLogLevel LoggerLevel = LVL_INFO // Changer à DEBUG pour activer les logs de débogage
	logFile         *os.File    = nil
	todoLogger      *log.Logger = nil
	traceLogger     *log.Logger = nil
	debugLogger     *log.Logger = nil
	successLogger   *log.Logger = nil
	infoLogger      *log.Logger = nil
	warningLogger   *log.Logger = nil
	errorLogger     *log.Logger = nil
	criticalLogger  *log.Logger = nil
)

func LogTodo(format string, args ...interface{}) {
	if currentLogLevel > LVL_DEBUG {
		return
	}
	if todoLogger == nil {
		DefaultInit()
	}
	todoLogger.Output(2, fmt.Sprintf(format, args...))
}

func LogTrace(format string, args ...interface{}) {
	if currentLogLevel > LVL_TRACE {
		return
	}
	if traceLogger == nil {
		DefaultInit()
	}
	traceLogger.Output(2, fmt.Sprintf(format, args...))
}

func LogDebug(format string, args ...interface{}) {
	if currentLogLevel > LVL_DEBUG {
		return
	}
	if debugLogger == nil {
		DefaultInit()
	}
	debugLogger.Output(2, fmt.Sprintf(format, args...))
}

func LogSuccess(format string, args ...interface{}) {
	if currentLogLevel > LVL_INFO {
		return
	}
	if successLogger == nil {
		DefaultInit()
	}
	successLogger.Output(2, fmt.Sprintf(format, args...))
}

func LogInfo(format string, args ...interface{}) {
	if currentLogLevel > LVL_INFO {
		return
	}
	if infoLogger == nil {
		DefaultInit()
	}
	infoLogger.Output(2, fmt.Sprintf(format, args...))
}

func LogError(format string, args ...interface{}) {
	if currentLogLevel > LVL_ERROR {
		return
	}
	if errorLogger == nil {
		DefaultInit()
	}
	errorLogger.Output(2, fmt.Sprintf(format, args...))
}

func LogWarning(format string, args ...interface{}) {
	if currentLogLevel > LVL_WARNING {
		return
	}
	if warningLogger == nil {
		DefaultInit()
	}
	warningLogger.Output(2, fmt.Sprintf(format, args...))
}

func LogCritical(format string, args ...interface{}) {
	if criticalLogger == nil {
		DefaultInit()
	}
	criticalLogger.Output(2, fmt.Sprintf(format, args...))
}

func LogCriticalE(format string, args ...interface{}) {
	if criticalLogger == nil {
		DefaultInit()
	}
	criticalLogger.Output(2, fmt.Sprintf(format, args...))
	criticalLogger.Output(2, "This program cannot continue and will now exit.")
	os.Exit(42)
}

// ************************************************************************************************
// SetDebugLevel sets the logging level and initializes the log file.
//
// Parameters:
//   - level: A string representing the desired logging level. Valid values are "debug", "info", "warning", "error", and "critical".
//   - logPath: A string representing the file path where the log file will be created or appended to.
//
// Returns:
//   - error: An error if the log level is invalid or if there is an issue opening the log file.
//
// The function sets the global logging level based on the provided level string. It then attempts to open the log file at the specified path.
// If the log file cannot be opened, it logs a critical error message to stderr and initializes the loggers to write to both stdout and the log file.
// If the log file is successfully opened, it initializes the loggers to write to stdout only. The function also sets the trace mode.
func SetDebugLevel(level string, logPath string) error {
	switch level {
	case "trace":
		currentLogLevel = LVL_TRACE
	case "debug":
		currentLogLevel = LVL_DEBUG
	case "info":
		currentLogLevel = LVL_INFO
	case "warning":
		currentLogLevel = LVL_WARNING
	case "error":
		currentLogLevel = LVL_ERROR
	case "critical":
		currentLogLevel = LVL_CRITICAL
	default:
		return fmt.Errorf("invalid log level")
	}

	var err error = nil
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[41m\033[37mCRITICAL: \033[0mUnable to open a log file %s: %v", logPath, err)
		_initLoggers(io.MultiWriter(os.Stdout, logFile))
	} else {
		_initLoggers(os.Stdout)
	}
	return nil
}

// ************************************************************************************************
// DefaultInit initializes various loggers with different log levels and color-coded prefixes.
// The loggers are configured to output to standard output or standard error with date, time, and short file information.
// Additionally, it sets the trace mode to true.
func DefaultInit() {
	currentLogLevel = LVL_TRACE
	_initLoggers(os.Stdout)
	LogWarning("Logger not initialized. Using default configuration. Stdout only with trace mode.")
}

// ************************************************************************************************
// _initLoggers initializes various loggers with different log levels and color-coded prefixes.
// Each logger writes to the provided io.Writer and includes the date, time, and short file information in the log output.
//
// Parameters:
//   - writer: An io.Writer where the log messages will be written.
//
// Loggers initialized:
//   - todoLogger: Logs messages with a "TODO" prefix in white text on a gray background.
//   - traceLogger: Logs messages with a "TRACE" prefix in white text on a gray background.
//   - debugLogger: Logs messages with a "DEBUG" prefix in white text on a gray background.
//   - successLogger: Logs messages with a "SUCCESS" prefix in black text on a green background.
//   - infoLogger: Logs messages with an "INFO" prefix in white text on a blue background.
//   - warningLogger: Logs messages with a "WARNING" prefix in black text on a yellow background.
//   - errorLogger: Logs messages with an "ERROR" prefix in black text on a yellow background.
//   - criticalLogger: Logs messages with a "CRITICAL" prefix in white text on a red background.
func _initLoggers(writer io.Writer) {
	todoLogger = log.New(writer, "\033[48;5;240m\033[37mTODO:     \033[0m", log.Ldate|log.Ltime|log.Lshortfile)  // Fond gris, texte blanc
	traceLogger = log.New(writer, "\033[48;5;240m\033[37mTRACE:    \033[0m", log.Ldate|log.Ltime|log.Lshortfile) // Fond gris, texte blanc
	debugLogger = log.New(writer, "\033[48;5;240m\033[37mDEBUG:    \033[0m", log.Ldate|log.Ltime|log.Lshortfile) // Fond gris, texte blanc
	successLogger = log.New(writer, "\033[42m\033[30mSUCCESS:  \033[0m", log.Ldate|log.Ltime|log.Lshortfile)     // Fond vert, texte noir
	infoLogger = log.New(writer, "\033[44m\033[37mINFO:     \033[0m", log.Ldate|log.Ltime|log.Lshortfile)        // Fond bleu, texte blanc
	warningLogger = log.New(writer, "\033[43m\033[30mWARNING:  \033[0m", log.Ldate|log.Ltime|log.Lshortfile)     // Fond jaune, texte noir
	errorLogger = log.New(writer, "\033[43m\033[30mERROR:    \033[0m", log.Ldate|log.Ltime|log.Lshortfile)       // Fond jaune, texte noir
	criticalLogger = log.New(writer, "\033[41m\033[37mCRITICAL: \033[0m", log.Ldate|log.Ltime|log.Lshortfile)    // Fond rouge, texte blanc
}

// ************************************************************************************************
// fmtError formats an error message with additional context about the caller.
// It includes the file name, line number, and function name from where it was called.
//
// Parameters:
//   - format: A format string as described in fmt.Sprintf.
//   - args: A variadic list of arguments to be formatted according to the format string.
//
// Returns:
//   - error: An error that includes the formatted message along with caller context information.
func FmtError(format string, args ...interface{}) error {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return fmt.Errorf("could not get caller information")
	}
	fn := runtime.FuncForPC(pc)
	return fmt.Errorf("%s:%d %s: %s", file, line, fn.Name(), fmt.Sprintf(format, args...))
}
