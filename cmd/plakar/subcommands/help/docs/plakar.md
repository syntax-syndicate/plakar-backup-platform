PLAKAR(1) - General Commands Manual

# NAME

**plakar** - effortless backups

# SYNOPSIS

**plakar**
\[**-config**&nbsp;*path*]
\[**-cpu**&nbsp;*number*]
\[**-hostname**&nbsp;*name*]
\[**-keyfile**&nbsp;*path*]
\[**-no-agent**]
\[**-quiet**]
\[**-trace**&nbsp;*what*]
\[**-username**&nbsp;*name*]
\[**at**&nbsp;*repository*]
*subcommand&nbsp;...*

# DESCRIPTION

**plakar**
is a tool to create distributed, versioned backups with compression,
encryption and data deduplication.

By default,
**plakar**
operates on the repository at
*~/.plakar*.
This can be changed either by using the
**at**
keyword or by setting a default repository using
plakar-config(1).

The following options are available:

**-config** *path*

> Use the configuration at
> *path*.

**-cpu** *number*

> Limit the number of parallelism that
> **plakar**
> uses to
> *number*.
> By default it's the number of online CPUs.

**-hostname** *name*

> Change the hostname used for backups.
> Defaults to the current hostname.

**-keyfile** *path*

> Use the passphrase from the key file at
> *path*
> instead of prompting to unlock.

**-no-agent**

> Run without attempting to connect to the agent.

**-quiet**

> Disable all output except for errors.

**-trace** *what*

> Display trace logs.
> *what*
> is a comma-separated series of keywords to enable the trace logs for
> different subsystems:
> **all**, **trace**, **repository**, **snapshot** and **server**.

**-username** *name*

> Change the username used for backups.
> Defaults to the current user name.

**at** *repository*

> Operate on the given
> *repository*.
> It could be a path, an URI, or a label in the form
> "@*name*"
> to reference a configuration created with
> plakar-config(1).

The following commands are available:

**agent**

> Run the plakar agent, documented in
> plakar-agent(1).

**archive**

> Create an archive from a Plakar snapshot, documented in
> plakar-archive(1).

**backup**

> Create a new snapshot, documented in
> plakar-backup(1).

**cat**

> Display file contents from a Plakar snapshot, documented in
> plakar-cat(1).

**check**

> Check data integrity in a Plakar repository, documented in
> plakar-check(1).

**clone**

> Clone a Plakar repository to a new location, documented in
> plakar-clone(1).

**config**

> Manage Plakar configuration, documented in
> plakar-config(1).

**create**

> Create a new Plakar repository, documented in
> plakar-create(1).

**diff**

> Show differences between files in a Plakar snapshot, documented in
> plakar-diff(1).

**digest**

> Compute digests for files in a Plakar snapshot, documented in
> plakar-digest(1).

**exec**

> Execute a file from a Plakar snapshot, documented in
> plakar-exec(1).

**help**

> Show this manpage and the ones for the subcommands.

**info**

> Display detailed information about internal structures, documented in
> plakar-info(1).

**locate**

> Find filenames in a Plakar snapshot, documented in
> plakar-locate(1).

**ls**

> List snapshots and their contents in a Plakar repository, documented in
> plakar-ls(1).

**maintenance**

> Remove unused data from a Plakar repository, documented in
> plakar-mantenance(1).

**mount**

> Mount Plakar snapshots as read-only filesystem, documented in
> plakar-mount(1).

**restore**

> Restore files from a Plakar snapshot, documented in
> plakar-restore(1).

**rm**

> Remove snapshots from a Plakar repository, documented in
> plakar-rm(1).

**server**

> Start a Plakar server, documented in
> plakar-server(1).

**sync**

> Synchronize sanpshots between Plakar repositories, documented in
> plakar-sync(1).

**ui**

> Serve the Plakar web user interface, documented in
> plakar-ui(1).

**version**

> Display the current Plakar version, documented in
> plakar-version(1).

# ENVIRONMENT

`PLAKAR_PASSPHRASE`

> Passphrase to unlock the repository, overrides the one from the configuration.
> If set,
> **plakar**
> won't prompt to unlock.

`PLAKAR_REPOSITORY`

> Path to the default repository, overrides the configuration set with
> **plakar config repository default**.

# FILES

*~/.cache/plakar and* *~/.cache/plakar-agentless*

> Plakar cache directories.

*~/.config/plakar/plakar.yml*

> Default configuration file.

*~/.plakar*

> Default repository location.

# EXAMPLES

Create an encrypted repository at the default location:

	$ plakar create

Create an encrypted repository on AWS S3:

	$ plakar config repository create mys3bucket
	$ plakar config repository set mys3bucket location \
		s3://s3.eu-west-3.amazonaws.com/backups
	$ plakar config repository set mys3bucket access_key "access_key"
	$ plakar config repository set mys3bucket secret_access_key "secret_key"
	$ plakar at @mys3bucket create

Set the
"mys3bucket"
repository just created as the default one used by
**plakar**:

	$ plakar config repository default mys3bucket

Create a snapshot of the current directory:

	$ plakar backup

List the snapshots:

	$ plakar ls

Restore the file
"notes.md"
in the current directory from the snapshot with id
"abcd":

	$ plakar restore -to . abcd:notes.md

Remove snapshots older than a 30 days:

	$ plakar rm -before 30d

Plakar - March 3, 2025
