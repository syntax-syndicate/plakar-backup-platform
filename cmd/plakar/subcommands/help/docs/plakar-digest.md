PLAKAR-DIGEST(1) - General Commands Manual

# NAME

**plakar digest** - Calculate digests for files in a Plakar snapshot

# SYNOPSIS

**plakar digest**
\[**-fast**]
*snapshotID*:*filepath*&nbsp;\[...]

# DESCRIPTION

The
**plakar digest**
command calculates and displays digests for specified
*filepath*
in a the given
*snapshotID*.
Multiple
*snapshotID*
and
*filepath*
may be given.
By default, the command computes the digest by reading the file
contents.

The options are as follows:

**-fast**

> Return the pre-recorded digest for the file without re-computing it
> from the file contents.
> It's faster, but it does not verify the integrity against the current
> contents.

# EXAMPLES

Calculate the digest of a file within a snapshot:

	$ plakar digest abc123:/etc/passwd

Retrieve the pre-recorded digest for faster output:

	$ plakar digest -fast abc123:/etc/netstart

# DIAGNOSTICS

The **plakar digest** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as failure to retrieve a file digest or
> invalid snapshot ID.

# SEE ALSO

plakar(1)

macOS 15.2 - February 3, 2025
