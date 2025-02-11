package events

import (
	"fmt"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

type SerializedEvent struct {
	Type string
	Data []byte
}

func Serialize(event Event) ([]byte, error) {
	var serialized SerializedEvent
	var err error
	switch e := event.(type) {
	case Start:
		serialized.Type = "Start"
		serialized.Data, err = msgpack.Marshal(e)
	case Done:
		serialized.Type = "Done"
		serialized.Data, err = msgpack.Marshal(e)
	case Warning:
		serialized.Type = "Warning"
		serialized.Data, err = msgpack.Marshal(e)
	case Error:
		serialized.Type = "Error"
		serialized.Data, err = msgpack.Marshal(e)
	case Path:
		serialized.Type = "Path"
		serialized.Data, err = msgpack.Marshal(e)
	case PathError:
		serialized.Type = "PathError"
		serialized.Data, err = msgpack.Marshal(e)
	case Directory:
		serialized.Type = "Directory"
		serialized.Data, err = msgpack.Marshal(e)
	case File:
		serialized.Type = "File"
		serialized.Data, err = msgpack.Marshal(e)
	case Object:
		serialized.Type = "Object"
		serialized.Data, err = msgpack.Marshal(e)
	case Chunk:
		serialized.Type = "Chunk"
		serialized.Data, err = msgpack.Marshal(e)
	case DirectoryOK:
		serialized.Type = "DirectoryOK"
		serialized.Data, err = msgpack.Marshal(e)
	case DirectoryError:
		serialized.Type = "DirectoryError"
		serialized.Data, err = msgpack.Marshal(e)
	case DirectoryMissing:
		serialized.Type = "DirectoryMissing"
		serialized.Data, err = msgpack.Marshal(e)
	case DirectoryCorrupted:
		serialized.Type = "DirectoryCorrupted"
		serialized.Data, err = msgpack.Marshal(e)
	case FileOK:
		serialized.Type = "FileOK"
		serialized.Data, err = msgpack.Marshal(e)
	case FileError:
		serialized.Type = "FileError"
		serialized.Data, err = msgpack.Marshal(e)
	case FileMissing:
		serialized.Type = "FileMissing"
		serialized.Data, err = msgpack.Marshal(e)
	case FileCorrupted:
		serialized.Type = "FileCorrupted"
		serialized.Data, err = msgpack.Marshal(e)
	case ObjectOK:
		serialized.Type = "ObjectOK"
		serialized.Data, err = msgpack.Marshal(e)
	case ObjectMissing:
		serialized.Type = "ObjectMissing"
		serialized.Data, err = msgpack.Marshal(e)
	case ObjectCorrupted:
		serialized.Type = "ObjectCorrupted"
		serialized.Data, err = msgpack.Marshal(e)
	case ChunkOK:
		serialized.Type = "ChunkOK"
		serialized.Data, err = msgpack.Marshal(e)
	case ChunkMissing:
		serialized.Type = "ChunkMissing"
		serialized.Data, err = msgpack.Marshal(e)
	case ChunkCorrupted:
		serialized.Type = "ChunkCorrupted"
		serialized.Data, err = msgpack.Marshal(e)
	case StartImporter:
		serialized.Type = "StartImporter"
		serialized.Data, err = msgpack.Marshal(e)
	case DoneImporter:
		serialized.Type = "DoneImporter"
		serialized.Data, err = msgpack.Marshal(e)
	default:
		return nil, fmt.Errorf("unknown event type")
	}
	if err != nil {
		return nil, err
	}
	return msgpack.Marshal(serialized)
}

func Deserialize(data []byte) (Event, error) {
	var serialized SerializedEvent
	if err := msgpack.Unmarshal(data, &serialized); err != nil {
		return nil, err
	}
	switch serialized.Type {
	case "Start":
		var e Start
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "Done":
		var e Done
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "Warning":
		var e Warning
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "Error":
		var e Error
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "Path":
		var e Path
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "PathError":
		var e PathError
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "Directory":
		var e Directory
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "File":
		var e File
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "Object":
		var e Object
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "Chunk":
		var e Chunk
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "DirectoryOK":
		var e DirectoryOK
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "DirectoryError":
		var e DirectoryError
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "DirectoryMissing":
		var e DirectoryMissing
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "DirectoryCorrupted":
		var e DirectoryCorrupted
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "FileOK":
		var e FileOK
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "FileError":
		var e FileError
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "FileMissing":
		var e FileMissing
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "FileCorrupted":
		var e FileCorrupted
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "ObjectOK":
		var e ObjectOK
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "ObjectMissing":
		var e ObjectMissing
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "ObjectCorrupted":
		var e ObjectCorrupted
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "ChunkOK":
		var e ChunkOK
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "ChunkMissing":
		var e ChunkMissing
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "ChunkCorrupted":
		var e ChunkCorrupted
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "StartImporter":
		var e StartImporter
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "DoneImporter":
		var e DoneImporter
		if err := msgpack.Unmarshal(serialized.Data, &e); err != nil {
			return nil, err
		}
		return e, nil
	default:
		return nil, fmt.Errorf("unknown event type")
	}
}

type Event interface {
}

/**/
type Start struct {
	Timestamp time.Time
}

func StartEvent() Start {
	return Start{Timestamp: time.Now()}
}

/**/
type Done struct {
	Timestamp time.Time
}

func DoneEvent() Done {
	return Done{Timestamp: time.Now()}
}

/**/
type Warning struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Message    string
}

func WarningEvent(snapshotID [32]byte, message string) Warning {
	return Warning{Timestamp: time.Now(), SnapshotID: snapshotID, Message: message}
}

/**/
type Error struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Message    string
}

func ErrorEvent(snapshotID [32]byte, message string) Error {
	return Error{Timestamp: time.Now(), SnapshotID: snapshotID, Message: message}
}

/**/
type Path struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
}

func PathEvent(snapshotID [32]byte, pathname string) Path {
	return Path{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname}
}

/**/
type PathError struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
	Message    string
}

func PathErrorEvent(snapshotID [32]byte, pathname string, message string) PathError {
	return PathError{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname, Message: message}
}

/**/
type Directory struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
}

func DirectoryEvent(snapshotID [32]byte, pathname string) Directory {
	return Directory{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname}
}

/**/
type File struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
}

func FileEvent(snapshotID [32]byte, pathname string) File {
	return File{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname}
}

/**/
type Object struct {
	Timestamp time.Time

	SnapshotID [32]byte
	MAC        [32]byte
}

func ObjectEvent(snapshotID [32]byte, mac [32]byte) Object {
	return Object{Timestamp: time.Now(), SnapshotID: snapshotID, MAC: mac}
}

/**/
type Chunk struct {
	Timestamp time.Time

	SnapshotID [32]byte
	MAC        [32]byte
}

func ChunkEvent(snapshotID [32]byte, mac [32]byte) Chunk {
	return Chunk{Timestamp: time.Now(), SnapshotID: snapshotID, MAC: mac}
}

/**/
type DirectoryOK struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
}

func DirectoryOKEvent(snapshotID [32]byte, pathname string) DirectoryOK {
	return DirectoryOK{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname}
}

/**/
type DirectoryError struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
	Message    string
}

func DirectoryErrorEvent(snapshotID [32]byte, pathname string, message string) DirectoryError {
	return DirectoryError{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname, Message: message}
}

/**/
type DirectoryMissing struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
}

func DirectoryMissingEvent(snapshotID [32]byte, pathname string) DirectoryMissing {
	return DirectoryMissing{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname}
}

/**/
type DirectoryCorrupted struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
}

func DirectoryCorruptedEvent(snapshotID [32]byte, pathname string) DirectoryCorrupted {
	return DirectoryCorrupted{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname}
}

/**/
type FileOK struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
	Size       int64
}

func FileOKEvent(snapshotID [32]byte, pathname string, size int64) FileOK {
	return FileOK{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname, Size: size}
}

/**/
type FileError struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
	Message    string
}

func FileErrorEvent(snapshotID [32]byte, pathname string, message string) FileError {
	return FileError{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname, Message: message}
}

/**/
type FileMissing struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
}

func FileMissingEvent(snapshotID [32]byte, pathname string) FileMissing {
	return FileMissing{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname}
}

/**/
type FileCorrupted struct {
	Timestamp time.Time

	SnapshotID [32]byte
	Pathname   string
}

func FileCorruptedEvent(snapshotID [32]byte, pathname string) FileCorrupted {
	return FileCorrupted{Timestamp: time.Now(), SnapshotID: snapshotID, Pathname: pathname}
}

/**/
type ObjectOK struct {
	Timestamp time.Time

	SnapshotID [32]byte
	MAC        [32]byte
}

func ObjectOKEvent(snapshotID [32]byte, mac [32]byte) ObjectOK {
	return ObjectOK{Timestamp: time.Now(), SnapshotID: snapshotID, MAC: mac}
}

/**/
type ObjectMissing struct {
	Timestamp time.Time

	SnapshotID [32]byte
	MAC        [32]byte
}

func ObjectMissingEvent(snapshotID [32]byte, mac [32]byte) ObjectMissing {
	return ObjectMissing{Timestamp: time.Now(), SnapshotID: snapshotID, MAC: mac}
}

/**/
type ObjectCorrupted struct {
	Timestamp time.Time

	SnapshotID [32]byte
	MAC        [32]byte
}

func ObjectCorruptedEvent(snapshotID [32]byte, mac [32]byte) ObjectCorrupted {
	return ObjectCorrupted{Timestamp: time.Now(), SnapshotID: snapshotID, MAC: mac}
}

/**/
type ChunkOK struct {
	Timestamp time.Time

	SnapshotID [32]byte
	MAC        [32]byte
}

func ChunkOKEvent(snapshotID [32]byte, mac [32]byte) ChunkOK {
	return ChunkOK{Timestamp: time.Now(), SnapshotID: snapshotID, MAC: mac}
}

/**/
type ChunkMissing struct {
	Timestamp time.Time

	SnapshotID [32]byte
	MAC        [32]byte
}

func ChunkMissingEvent(snapshotID [32]byte, mac [32]byte) ChunkMissing {
	return ChunkMissing{Timestamp: time.Now(), SnapshotID: snapshotID, MAC: mac}
}

/**/
type ChunkCorrupted struct {
	Timestamp time.Time

	SnapshotID [32]byte
	MAC        [32]byte
}

func ChunkCorruptedEvent(snapshotID [32]byte, mac [32]byte) ChunkCorrupted {
	return ChunkCorrupted{Timestamp: time.Now(), SnapshotID: snapshotID, MAC: mac}
}

/**/
type StartImporter struct {
	Timestamp time.Time

	SnapshotID [32]byte
}

func StartImporterEvent() StartImporter {
	return StartImporter{Timestamp: time.Now()}
}

/**/
type DoneImporter struct {
	Timestamp time.Time

	SnapshotID     [32]byte
	NumFiles       uint64
	NumDirectories uint64
	Size           uint64
}

func DoneImporterEvent() DoneImporter {
	return DoneImporter{Timestamp: time.Now()}
}
