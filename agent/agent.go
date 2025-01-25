package agent

import (
	"fmt"
	"net"
	"os"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/scheduler"
)

type Daemon struct {
	socketPath    string
	listener      net.Listener
	ctx           *appcontext.AppContext
	configuration *config.AgentConfig
}

func NewDaemon(ctx *appcontext.AppContext, network string, address string, configuration *config.AgentConfig) (*Daemon, error) {
	if network != "unix" {
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	d := &Daemon{
		socketPath:    address,
		ctx:           ctx,
		configuration: configuration,
	}

	if _, err := os.Stat(d.socketPath); err == nil {
		if !d.checkSocket() {
			d.Close()
		} else {
			return nil, fmt.Errorf("already running")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return nil, err
	}
	d.listener = listener

	if err := os.Chmod(d.socketPath, 0600); err != nil {
		d.Close()
		return nil, err
	}

	return d, nil
}

func (d *Daemon) checkSocket() bool {
	conn, err := net.Dial("unix", d.socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (d *Daemon) Close() error {
	if d.listener != nil {
		d.listener.Close()
	}
	return os.Remove(d.socketPath)
}

func (d *Daemon) ListenAndServe() error {
	sched := scheduler.NewScheduler(d.ctx, d.configuration.Tasks)
	sched.Run()

	// for now, we only use the scheduling subsystem, no proxying CLI requests
	/*
		for {
			conn, err := d.listener.Accept()
			if err != nil {
				return err
			}
			// for now, just close the connection
			conn.Close()
		}
	*/
	return nil
}
