package logging

import (
	"bytes"
	"sync"
	"testing"
)

func TestLogger(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	// Create a new logger
	logger := NewLogger(bufOut, bufErr)

	// Test Printf
	logger.Printf("Test message")
	if bufOut.String() != "info: Test message\n" {
		t.Errorf("Printf did not produce expected output")
	}
	bufOut.Reset()

	// Test Stdout
	logger.Stdout("Test message")
	if bufOut.String() != "Test message\n" {
		t.Errorf("Stdout did not produce expected output")
	}
	bufOut.Reset()

	// Test Stderr
	logger.Stderr("Test message")
	if bufErr.String() != "Test message\n" {
		t.Errorf("Stderr did not produce expected output")
	}
	bufErr.Reset()

	// Test Warn
	logger.Warn("Test message")
	if bufErr.String() != "warn: Test message\n" {
		t.Errorf("Warn did not produce expected output")
	}
	bufErr.Reset()

	// Test Error
	logger.Error("Test message")
	if bufErr.String() != "error: Test message\n" {
		t.Errorf("Error did not produce expected output")
	}
	bufErr.Reset()

	// Test Debug
	logger.Debug("Test message")
	if bufOut.String() != "debug: Test message\n" {
		t.Errorf("Debug did not produce expected output")
	}
	bufOut.Reset()

	// Test Info without enabling info
	logger.Info("Test message")
	if bufOut.String() != "" {
		t.Errorf("Info should not produce output")
	}

	// Test EnableInfo
	logger.EnableInfo()
	if !logger.EnabledInfo {
		t.Errorf("EnableInfo did not enable info logging")
	}

	// Test Info
	logger.Info("Test message")
	if bufOut.String() != "info: Test message\n" {
		t.Errorf("Info did not produce expected output")
	}
	bufOut.Reset()

	// Test Trace without enabling trace
	logger.Trace("subsystem", "Test message")
	if bufOut.String() != "" {
		t.Errorf("Trace should not produce output")
	}
	bufOut.Reset()

	// Test EnableTrace
	logger.EnableTracing("subsystem")
	if logger.EnabledTracing == "" {
		t.Errorf("EnableTrace did not enable tracing")
	}
	if _, ok := logger.traceSubsystems["subsystem"]; !ok {
		t.Errorf("EnableTrace did not add subsystem to tracing")
	}

	// Test Trace
	logger.Trace("subsystem", "Test message")
	if bufOut.String() != "trace: subsystem: Test message\n" {
		t.Errorf("Trace did not produce expected output")
	}
	bufOut.Reset()

	// Test Trace with unknown subsystem but not all tracing subsystem enabled
	logger.Trace("unknown", "Test message")
	if bufOut.String() != "" {
		t.Errorf("Trace should not produce output")
	}
	logger.EnableTracing("all")
	logger.Trace("unknown", "Test message")
	if bufOut.String() != "trace: unknown: Test message\n" {
		t.Errorf("Trace did not produce expected output")
	}
	bufOut.Reset()
}

func TestLoggerConcurrency(t *testing.T) {
	// Create a new logger
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	// Create a new logger
	logger := NewLogger(bufOut, bufErr)

	// Test concurrent logging
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Printf("Test message %d", i)
		}()
	}
	wg.Wait()
	if bufOut.String() == "" {
		t.Errorf("Concurrent logging produced unexpected output")
	}
}

func TestLoggerPanic(t *testing.T) {
	// Create a new logger
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	// Create a new logger
	logger := NewLogger(bufOut, bufErr)

	// Test panic logging
	defer func() {
		if r := recover(); r != nil {
			logger.Printf("Recovered panic: %v", r)
		}
		if bufOut.String() != "info: Recovered panic: Test panic\n" {
			t.Errorf("Panic logging did not produce expected output")
		}
	}()
	panic("Test panic")
}
