PLAKAR-INFO(1) - General Commands Manual

# NAME

**plakar info** - Display detailed information about a Plakar repository, snapshot and filesystem entries

# SYNOPSIS

**plakar info**
\[**\[\[SNAPSHOT]\[:/path/to/file-or-directory]]**]

# DESCRIPTION

The
**plakar info**
command provides detailed information about a Plakar repository,
snapshots and filesystem entries.
The type of information displayed depends on the specified argument.
Without any arguents, display information about the repository.

# EXAMPLES

Show repository information:

	$ plakar info

Show detailed information for a snapshot:

	$ plakar info abc123

Show detailed information for a file within a snapshot:

	$ plakar info abcd123:/etc/passwd

# DIAGNOSTICS

The **plakar info** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as an invalid snapshot or object ID, or a
> failure to retrieve the requested data.

# SEE ALSO

plakar(1),
plakar-snapshot(1)

Plakar - February 3, 2025
