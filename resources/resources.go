package resources

type Type uint32

const (
	RT_CONFIG    Type = 1
	RT_LOCK      Type = 2
	RT_STATE     Type = 3
	RT_PACKFILE  Type = 4
	RT_SNAPSHOT  Type = 5
	RT_SIGNATURE Type = 6
	RT_OBJECT    Type = 7
	RT_CHUNK     Type = 9
	RT_VFS_BTREE Type = 10
	RT_VFS_ENTRY Type = 11
	RT_BTREE     Type = 12
	RT_ERROR     Type = 13
)

func Types() []Type {
	return []Type{
		RT_CONFIG,
   	RT_LOCK,
		RT_STATE,
		RT_PACKFILE,
		RT_SNAPSHOT,
		RT_SIGNATURE,
		RT_OBJECT,
		RT_CHUNK,
		RT_VFS_BTREE,
		RT_VFS_ENTRY,
		RT_BTREE,
		RT_ERROR,
	}
}

func (r Type) String() string {
	switch r {
	case RT_CONFIG:
		return "config"
  case RT_LOCK:
		return "config"
	case RT_STATE:
		return "state"
	case RT_PACKFILE:
		return "packfile"
	case RT_SNAPSHOT:
		return "snapshot"
	case RT_SIGNATURE:
		return "signature"
	case RT_OBJECT:
		return "object"
	case RT_CHUNK:
		return "chunk"
	case RT_VFS_BTREE:
		return "vfs btree"
	case RT_VFS_ENTRY:
		return "vfs entry"
	case RT_BTREE:
		return "btree"
	case RT_ERROR:
		return "error"
	default:
		return "unknown"
	}
}
