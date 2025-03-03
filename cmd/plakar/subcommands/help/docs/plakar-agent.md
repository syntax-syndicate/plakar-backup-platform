PLAKAR-AGENT(1) - General Commands Manual

# NAME

**plakar agent** - Run the Plakar agent

# SYNOPSIS

**plakar agent**
\[**-foreground**]
\[**-log**&nbsp;*filename*]
\[**-stop**]

# DESCRIPTION

The
**plakar agent**
command starts the Plakar agent which will execute subsequent
plakar
commands on their behalfs for faster processing.
**plakar agent**
continues running indefinitely.

The options are as follows:

**-foreground**

> Do not daemonize agent,
> run in foreground.

**-log** *filename*

> Redirect all output to
> *filename*.

**-stop**

> Terminate an agent running in the background.

# DIAGNOSTICS

The **plakar agent** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as invalid parameters, inability to create the
> repository, or configuration issues.

# SEE ALSO

plakar(1)

Plakar - February 1, 2025
