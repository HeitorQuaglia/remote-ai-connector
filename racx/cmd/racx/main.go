// Comando racx inicia o executor local: sandbox ao CWD, servidor HTTP com as
// 4 tools. A integração SSH virá no Plano 3; por enquanto o servidor escuta
// diretamente em uma porta local.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HeitorQuaglia/remote-ai-connector/racx/internal/audit"
	fspkg "github.com/HeitorQuaglia/remote-ai-connector/racx/internal/fs"
	"github.com/HeitorQuaglia/remote-ai-connector/racx/internal/server"
)

func main() {
	var (
		listen    = flag.String("listen", "127.0.0.1:0", "host:port to listen on")
		quiet     = flag.Bool("quiet", false, "suppress audit output on stderr")
		printPort = flag.Bool("print-port", false, "print listen address to stdout and exit on signal")
	)
	flag.Parse()

	if err := run(*listen, *quiet, *printPort); err != nil {
		fmt.Fprintf(os.Stderr, "racx: %v\n", err)
		os.Exit(1)
	}
}

func run(listen string, quiet, printPort bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}
	sb, err := fspkg.NewSandbox(cwd)
	if err != nil {
		return fmt.Errorf("init sandbox: %w", err)
	}
	gi, err := fspkg.LoadGitignore(cwd)
	if err != nil {
		return fmt.Errorf("load gitignore: %w", err)
	}
	vis := fspkg.NewVisibility(gi, false)

	var auditOut io.Writer = os.Stderr
	logger := audit.NewTextLogger(auditOut, quiet, time.Now)

	srv := server.New(sb, vis, logger)

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	httpSrv := &http.Server{Handler: srv.Handler(), ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		if printPort {
			fmt.Println(ln.Addr().String())
		} else {
			fmt.Fprintf(os.Stderr, "racx listening on %s (root=%s)\n", ln.Addr().String(), cwd)
		}
		if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "racx: server error: %v\n", err)
		}
	}()

	<-ctx.Done()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	return httpSrv.Shutdown(shutCtx)
}
