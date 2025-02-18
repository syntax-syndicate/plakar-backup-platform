PLAKAR-CONFIG(1) - General Commands Manual

# NAME

**plakar config** - Manages plakar configuration

# SYNOPSIS

**plakar config**
\[*key*\[=*value*]]

# DESCRIPTION

The
**plakar config**
command manages configuration of the Plakar software.

Without arguments show all the configuration options currently set on the repository.
With just
*key*,
show the value defined for that key.
Otherwise, set
*key*
to
*value*.

*key*
is of the form
'*category*.*option*'.

# DIAGNOSTICS

The **plakar config** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

plakar(1)

Plakar - February 17, 2025
