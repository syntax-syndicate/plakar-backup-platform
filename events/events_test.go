package events

import (
	"testing"
)

func TestStartTimestamp(t *testing.T) {
	start := StartEvent()
	if start.Timestamp.IsZero() {
		t.Errorf("StartEvent().Timestamp returned a zero timestamp")
	}
}

func TestDoneEvent(t *testing.T) {
	done := DoneEvent()
	if done.Timestamp.IsZero() {
		t.Errorf("DoneEvent().Timestamp returned a zero timestamp")
	}
}

func TestWarning(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	warning := WarningEvent(snapshotId, "Test warning message")
	if warning.Timestamp.IsZero() {
		t.Errorf("WarningEvent().Timestamp returned a zero timestamp")
	}
	if len(warning.SnapshotID) != 32 {
		t.Errorf("WarningEvent SnapshotID length is not 32")
	}
	if warning.Message == "" {
		t.Errorf("WarningEvent message is empty")
	}
}

func TestError(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	errorEvent := ErrorEvent(snapshotId, "Test error message")
	if errorEvent.Timestamp.IsZero() {
		t.Errorf("ErrorEvent().Timestamp returned a zero timestamp")
	}
	if len(errorEvent.SnapshotID) != 32 {
		t.Errorf("ErrorEvent SnapshotID length is not 32")
	}
	if errorEvent.Message == "" {
		t.Errorf("ErrorEvent message is empty")
	}
}

func TestDirectory(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	directory := DirectoryEvent(snapshotId, "Test pathname")
	if directory.Timestamp.IsZero() {
		t.Errorf("DirectoryEvent().Timestamp returned a zero timestamp")
	}
	if len(directory.SnapshotID) != 32 {
		t.Errorf("DirectoryEvent SnapshotID length is not 32")
	}
	if directory.Pathname == "" {
		t.Errorf("DirectoryEvent pathname is empty")
	}
}

func TestFile(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	file := FileEvent(snapshotId, "Test pathname")
	if file.Timestamp.IsZero() {
		t.Errorf("FileEvent().Timestamp returned a zero timestamp")
	}
	if len(file.SnapshotID) != 32 {
		t.Errorf("FileEvent SnapshotID length is not 32")
	}
	if file.Pathname == "" {
		t.Errorf("FileEvent pathname is empty")
	}
}
func TestPathEvent(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	path := PathEvent(snapshotId, "Test pathname")
	if path.Timestamp.IsZero() {
		t.Errorf("PathEvent().Timestamp returned a zero timestamp")
	}
	if len(path.SnapshotID) != 32 {
		t.Errorf("PathEvent SnapshotID length is not 32")
	}
	if path.Pathname == "" {
		t.Errorf("PathEvent pathname is empty")
	}
}

func TestPathError(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	pathError := PathErrorEvent(snapshotId, "Test path name", "Test error message")
	if pathError.Timestamp.IsZero() {
		t.Errorf("PathErrorEvent().Timestamp returned a zero timestamp")
	}
	if len(pathError.SnapshotID) != 32 {
		t.Errorf("PathErrorEvent SnapshotID length is not 32")
	}
	if pathError.Pathname == "" {
		t.Errorf("PathErrorEvent pathname is empty")
	}
	if pathError.Message == "" {
		t.Errorf("PathErrorEvent message is empty")
	}
}

func TestDirectoryMissing(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	directoryMissing := DirectoryMissingEvent(snapshotId, "Test pathname")
	if directoryMissing.Timestamp.IsZero() {
		t.Errorf("DirectoryMissingEvent().Timestamp returned a zero timestamp")
	}
	if len(directoryMissing.SnapshotID) != 32 {
		t.Errorf("DirectoryMissingEvent SnapshotID length is not 32")
	}
	if directoryMissing.Pathname == "" {
		t.Errorf("DirectoryMissingEvent pathname is empty")
	}
}

func TestFileOK(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	fileOK := FileOKEvent(snapshotId, "Test pathname", 42)
	if fileOK.Timestamp.IsZero() {
		t.Errorf("FileOKEvent().Timestamp returned a zero timestamp")
	}
	if len(fileOK.SnapshotID) != 32 {
		t.Errorf("FileOKEvent SnapshotID length is not 32")
	}
	if fileOK.Pathname == "" {
		t.Errorf("FileOKEvent pathname is empty")
	}
	if fileOK.Size != 42 {
		t.Errorf("FileOKEvent size is not 42")
	}
}

func TestFileError(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	fileError := FileErrorEvent(snapshotId, "Test pathname", "Test error message")
	if fileError.Timestamp.IsZero() {
		t.Errorf("FileErrorEvent().Timestamp returned a zero timestamp")
	}
	if len(fileError.SnapshotID) != 32 {
		t.Errorf("FileErrorEvent SnapshotID length is not 32")
	}
	if fileError.Pathname == "" {
		t.Errorf("FileErrorEvent pathname is empty")
	}
	if fileError.Message == "" {
		t.Errorf("FileErrorEvent message is empty")
	}
}

func TestFileMissing(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	fileMissing := FileMissingEvent(snapshotId, "Test pathname")
	if fileMissing.Timestamp.IsZero() {
		t.Errorf("FileMissingEvent().Timestamp returned a zero timestamp")
	}
	if len(fileMissing.SnapshotID) != 32 {
		t.Errorf("FileMissingEvent SnapshotID length is not 32")
	}
	if fileMissing.Pathname == "" {
		t.Errorf("FileMissingEvent pathname is empty")
	}
}

func TestFileCorrupted(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	fileCorrupted := FileCorruptedEvent(snapshotId, "Test pathname")
	if fileCorrupted.Timestamp.IsZero() {
		t.Errorf("FileCorruptedEvent().Timestamp returned a zero timestamp")
	}
	if len(fileCorrupted.SnapshotID) != 32 {
		t.Errorf("FileCorruptedEvent SnapshotID length is not 32")
	}
	if fileCorrupted.Pathname == "" {
		t.Errorf("FileCorruptedEvent pathname is empty")
	}
}

func TestDirectoryCorrupted(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	directoryCorrupted := DirectoryCorruptedEvent(snapshotId, "Test pathname")
	if directoryCorrupted.Timestamp.IsZero() {
		t.Errorf("DirectoryCorruptedEvent().Timestamp returned a zero timestamp")
	}
	if len(directoryCorrupted.SnapshotID) != 32 {
		t.Errorf("DirectoryCorruptedEvent SnapshotID length is not 32")
	}
	if directoryCorrupted.Pathname == "" {
		t.Errorf("DirectoryCorruptedEvent pathname is empty")
	}
}

func TestDirectoryError(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	directoryError := DirectoryErrorEvent(snapshotId, "Test pathname", "Test error message")
	if directoryError.Timestamp.IsZero() {
		t.Errorf("DirectoryErrorEvent().Timestamp returned a zero timestamp")
	}
	if len(directoryError.SnapshotID) != 32 {
		t.Errorf("DirectoryErrorEvent SnapshotID length is not 32")
	}
	if directoryError.Pathname == "" {
		t.Errorf("DirectoryErrorEvent pathname is empty")
	}
	if directoryError.Message == "" {
		t.Errorf("DirectoryErrorEvent message is empty")
	}
}

func TestDirectoryOKEvent(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	directoryOK := DirectoryOKEvent(snapshotId, "Test pathname")
	if directoryOK.Timestamp.IsZero() {
		t.Errorf("DirectoryOKEvent().Timestamp returned a zero timestamp")
	}
	if len(directoryOK.SnapshotID) != 32 {
		t.Errorf("DirectoryOKEvent SnapshotID length is not 32")
	}
	if directoryOK.Pathname == "" {
		t.Errorf("DirectoryOKEvent pathname is empty")
	}
}

func TestObjectEvent(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	mac := [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	object := ObjectEvent(snapshotId, mac)
	if object.Timestamp.IsZero() {
		t.Errorf("ObjectEvent().Timestamp returned a zero timestamp")
	}
	if len(object.SnapshotID) != 32 {
		t.Errorf("ObjectEvent SnapshotID length is not 32")
	}
	if len(object.MAC) != 32 {
		t.Errorf("ObjectEvent MAC length is not 32")
	}
}

func TestObjectOKEvent(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	mac := [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	objectOK := ObjectOKEvent(snapshotId, mac)
	if objectOK.Timestamp.IsZero() {
		t.Errorf("ObjectOKEvent().Timestamp returned a zero timestamp")
	}
	if len(objectOK.SnapshotID) != 32 {
		t.Errorf("ObjectOKEvent SnapshotID length is not 32")
	}
	if len(objectOK.MAC) != 32 {
		t.Errorf("ObjectOKEvent MAC length is not 32")
	}
}

func TestObjectMissingEvent(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	mac := [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	objectMissing := ObjectMissingEvent(snapshotId, mac)
	if objectMissing.Timestamp.IsZero() {
		t.Errorf("ObjectMissingEvent().Timestamp returned a zero timestamp")
	}
	if len(objectMissing.SnapshotID) != 32 {
		t.Errorf("ObjectMissingEvent SnapshotID length is not 32")
	}
	if len(objectMissing.MAC) != 32 {
		t.Errorf("ObjectMissingEvent MAC length is not 32")
	}
}

func TestObjectCorruptedEvent(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	mac := [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	objectCorrupted := ObjectCorruptedEvent(snapshotId, mac)
	if objectCorrupted.Timestamp.IsZero() {
		t.Errorf("ObjectCorruptedEvent().Timestamp returned a zero timestamp")
	}
	if len(objectCorrupted.SnapshotID) != 32 {
		t.Errorf("ObjectCorruptedEvent SnapshotID length is not 32")
	}
	if len(objectCorrupted.MAC) != 32 {
		t.Errorf("ObjectCorruptedEvent MAC length is not 32")
	}
}

func TestChunkEvent(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	mac := [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	chunk := ChunkEvent(snapshotId, mac)
	if chunk.Timestamp.IsZero() {
		t.Errorf("ChunkEvent().Timestamp returned a zero timestamp")
	}
	if len(chunk.SnapshotID) != 32 {
		t.Errorf("ChunkEvent SnapshotID length is not 32")
	}
	if len(chunk.MAC) != 32 {
		t.Errorf("ChunkEvent MAC length is not 32")
	}
}

func TestChunkOKEvent(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	mac := [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	chunkOK := ChunkOKEvent(snapshotId, mac)
	if chunkOK.Timestamp.IsZero() {
		t.Errorf("ChunkOKEvent().Timestamp returned a zero timestamp")
	}
	if len(chunkOK.SnapshotID) != 32 {
		t.Errorf("ChunkOKEvent SnapshotID length is not 32")
	}
	if len(chunkOK.MAC) != 32 {
		t.Errorf("ChunkOKEvent MAC length is not 32")
	}
}

func TestChunkMissingEvent(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	mac := [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	chunkMissing := ChunkMissingEvent(snapshotId, mac)
	if chunkMissing.Timestamp.IsZero() {
		t.Errorf("ChunkMissingEvent().Timestamp returned a zero timestamp")
	}
	if len(chunkMissing.SnapshotID) != 32 {
		t.Errorf("ChunkMissingEvent SnapshotID length is not 32")
	}
	if len(chunkMissing.MAC) != 32 {
		t.Errorf("ChunkMissingEvent MAC length is not 32")
	}
}

func TestChunkCorruptedEvent(t *testing.T) {
	snapshotId := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	mac := [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	chunkCorrupted := ChunkCorruptedEvent(snapshotId, mac)
	if chunkCorrupted.Timestamp.IsZero() {
		t.Errorf("ChunkCorruptedEvent().Timestamp returned a zero timestamp")
	}
	if len(chunkCorrupted.SnapshotID) != 32 {
		t.Errorf("ChunkCorruptedEvent SnapshotID length is not 32")
	}
	if len(chunkCorrupted.MAC) != 32 {
		t.Errorf("ChunkCorruptedEvent MAC length is not 32")
	}
}
