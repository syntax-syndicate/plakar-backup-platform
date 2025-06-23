PLAKAR-KLOSET(1) - General Commands Manual

# NAME

**plakar kloset** - Manage Plakar repository configurations

# SYNOPSIS

**plakar kloset**
\[subcommand&nbsp;...]

# DESCRIPTION

The
**plakar kloset**
command manages the Plakar repository configurations.

The configuration consists in a set of named entries, each of them
describing a Plakar repository (kloset) holding backups.

A repository is defined by at least a location, specifying the storage
implementation to use, and some storage-specific parameters.

The subcommands are as follows:

**add** *name* *location* \[option=value ...]

> Create a new repository entry identified by
> *name*
> with the specified
> *location*.
> Specific additional configuration parameters can be set by adding
> *option=value*
> parameters.

**check** *name*

> Check wether the repository identified by
> *name*
> is properly configured.

**ls**

> Display the current repositories configuration.
> This is the default if no subcommand is specified.

**ping** *name*

> Try to connect to the repository identified by
> *name*
> to make sure it is reachable.

**rm** *name*

> Remove the repository identified by
> *name*
> from the configuration.

**set** *name* \[option=value ...]

> Set the
> *option*
> to
> *value*
> for the repository identified by
> *name*.
> Multiple option/value pairs can be specified.

**unset** *name* \[option ...]

> Remove the
> *option*
> for the repository entry identified by
> *name*.

# DIAGNOSTICS

The **plakar kloset** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - February 27, 2025
