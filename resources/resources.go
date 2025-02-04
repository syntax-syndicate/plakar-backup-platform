package resources

type Resource uint32

const (
	RT_STATE           Resource = 1
	RT_PACKFILE        Resource = 2
	RT_PACKFILE_INDEX  Resource = 3
	RT_PACKFILE_FOOTER Resource = 4
	RT_SNAPSHOT        Resource = 5
	RT_CHUNK           Resource = 6
	RT_OBJECT          Resource = 7
	RT_VFS             Resource = 8
	RT_VFS_ENTRY       Resource = 9
	RT_CHILD           Resource = 10
	RT_SIGNATURE       Resource = 11
	RT_ERROR           Resource = 12
)
