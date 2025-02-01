PLAKAR-SERVER(1) - General Commands Manual

# NAME

**plakar server** - Start a Plakar server instance

# SYNOPSIS

**plakar server**
\[**-protocol**&nbsp;*protocol*]
\[**-allow-delete**]
\[*address*]

# DESCRIPTION

The
**plakar server**
command starts a Plakar server instance at the provided
*address*,
allowing remote interaction with a Plakar repository over a network.
If no
*address*
is given, the server listens on localhost at port 9876.

The options are as follows:

**-protocol** *protocol*

> Specify the protocol for the server to use.
> Options are:

> http

> > Start an HTTP server.

> plakar

> > Start a Plakar-native server (default).

**-allow-delete**

> Enable delete operations.
> By default, delete operations are disabled to prevent accidental data
> loss.

# EXAMPLES

Start server with default Plakar protocol:

	plakar server

Start HTTP server with delete operations enabled:

	plakar server -protocol http -allow-delete :8080

# DIAGNOSTICS

The **plakar server** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

0

> Command completed successfully.

&gt;0

> An error occurred, such as an unsupported protocol or invalid
> configuration.

# SEE ALSO

plakar(1)

Nixpkgs - January 29, 2025
