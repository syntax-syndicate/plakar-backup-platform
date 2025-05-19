PLAKAR-CONFIG(1) - General Commands Manual

# NAME

**plakar config** - Manage Plakar configuration

# SYNOPSIS

**plakar config**
\[**remote**&nbsp;|&nbsp;**repository**]

# DESCRIPTION

The
**plakar config**
command manages configuration of the Plakar software.

Without arguments show all the configuration options currently set on
the repository.

The subcommands are as follows:

**remote**

> Manage remotes configuration.
> The arguments are as follows:

> **create** *name*

> > Create a new remote identified by
> > *name*.

> **set** *name option value*

> > Set the
> > *option*
> > to
> > *value*
> > for the remote identified by
> > *name*.
> > Different remotes have different options available.

> **unset** *name option*

> > Remove the
> > *option*
> > for the remote identified by
> > *name*.

> **validate** *name*

> > Attempt to validate the configuration for the remote
> > *name*
> > to ensure whether it is working.

**repository** or **repo**

> Manage repositories configuration.
> The arguments are as follows:

> **create** *name*

> > Create a new repository configuration for
> > *name*.

> **default** *name*

> > Set the repository identified by
> > *name*
> > as the default repository used by
> > plakar(1).

> **set** *name option value*

> > Set the
> > *option*
> > to
> > *value*
> > for the repository identified by
> > *name*.

> **unset** *name option*

> > Remove the
> > *option*
> > for the repository
> > *name*.

> **validate** *name*

> > Attempt to validate the configuration for the repository
> > *name*
> > to ensure whether the parameters are correct.

# EXAMPLES

Create a new repository configuration called
"nas"
that connects over SFTP:

	$ plakar config repository create nas
	$ plakar config repository set nas location sftp://mynas/var/plakar

Perform a backup on the
"nas"
repository:

	$ plakar at @nas backup /var/www

The set the
"nas"
repository as the default one:

	$ plakar config repository default nas

# DIAGNOSTICS

The **plakar config** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - February 27, 2025
