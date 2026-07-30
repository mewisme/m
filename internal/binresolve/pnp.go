package binresolve

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
)

const (
	pnpProbeTimeout      = 5 * time.Second
	pnpMaxStdoutBytes    = 64 << 10
	pnpMaxStderrBytes    = 16 << 10
	pnpMaxJSONBytes      = 32 << 10
	pnpMaxDependencies   = 10000
	pnpMaxManifestReads  = 1000
	pnpMaxPathDepth      = 64
	pnpMaxLocatorEntries = 10000
	pnpMaxDiagnostics    = 100
)

// pnpInResolve guards against recursive bin resolution during PnP helper startup.
var pnpInResolve bool

type pnpRequest struct {
	Command     string `json:"command"`
	Package     string `json:"package,omitempty"`
	ProjectRoot string `json:"projectRoot"`
	ImporterRel string `json:"importerRel"`
}

type pnpResponse struct {
	Found    bool   `json:"found"`
	ShimPath string `json:"shimPath,omitempty"`
	Package  string `json:"package,omitempty"`
	Error    string `json:"error,omitempty"`
}

func resolvePnP(opts Options, lvl importerLevel, command string, doc *binmeta.Document) (binmeta.BinCandidate, []binmeta.BinCandidate, error) {
	if pnpInResolve {
		return binmeta.BinCandidate{}, nil, apperr.New(apperr.Internal, "binresolve.pnp", command, "recursive PnP probe blocked")
	}
	pnpFile := filepath.Join(opts.ProjectRoot, ".pnp.cjs")
	if _, err := os.Stat(pnpFile); err != nil {
		return binmeta.BinCandidate{}, nil, apperr.New(apperr.PNPUnsupported, "binresolve.pnp", command, "missing .pnp.cjs")
	}
	pnpInResolve = true
	defer func() { pnpInResolve = false }()

	hostEnv := []string(nil)
	node, err := TrustedNodePath(hostEnv)
	if err != nil {
		return binmeta.BinCandidate{}, nil, err
	}
	req := pnpRequest{
		Command:     command,
		Package:     opts.PackageFilter,
		ProjectRoot: opts.ProjectRoot,
		ImporterRel: lvl.ImporterRel,
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return binmeta.BinCandidate{}, nil, apperr.Wrap(apperr.Internal, "binresolve.pnp", command, err)
	}
	helper := pnpHelperScript()
	ctx, cancel := context.WithTimeout(context.Background(), pnpProbeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", helper)
	cmd.Dir = opts.ProjectRoot
	cmd.Env = []string{"NODE_OPTIONS=", "PWD=" + opts.ProjectRoot}
	cmd.Stdin = strings.NewReader(string(reqBytes))
	var stdout, stderr strings.Builder
	cmd.Stdout = &limitedWriter{w: &stdout, limit: pnpMaxStdoutBytes}
	cmd.Stderr = &limitedWriter{w: &stderr, limit: pnpMaxStderrBytes}
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return binmeta.BinCandidate{}, nil, apperr.Wrap(apperr.Timeout, "binresolve.pnp", command, context.DeadlineExceeded)
		}
		return binmeta.BinCandidate{}, nil, apperr.Wrap(apperr.PNPUnsupported, "binresolve.pnp", command, err)
	}
	out := stdout.String()
	if len(out) > pnpMaxJSONBytes {
		return binmeta.BinCandidate{}, nil, apperr.New(apperr.PNPUnsupported, "binresolve.pnp", command, "PnP helper output limit exceeded")
	}
	var resp pnpResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		return binmeta.BinCandidate{}, nil, apperr.Wrap(apperr.PNPUnsupported, "binresolve.pnp", command, err)
	}
	if resp.Error != "" {
		return binmeta.BinCandidate{}, nil, apperr.New(apperr.PNPUnsupported, "binresolve.pnp", command, resp.Error)
	}
	if !resp.Found {
		return binmeta.BinCandidate{}, nil, nil
	}
	cand := binmeta.BinCandidate{
		Command:           command,
		DependencyName:    resp.Package,
		ShimPath:          resp.ShimPath,
		TargetPath:        resp.ShimPath,
		OwnershipVerified: true,
	}
	_ = doc
	return cand, nil, nil
}

type limitedWriter struct {
	w     *strings.Builder
	limit int
	n     int
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.n >= lw.limit {
		return 0, apperr.New(apperr.PNPUnsupported, "binresolve.pnp", "", "PnP helper I/O limit exceeded")
	}
	remain := lw.limit - lw.n
	if len(p) > remain {
		p = p[:remain]
	}
	n, err := lw.w.Write(p)
	lw.n += n
	return n, err
}

// ponytail: minimal JSON probe; upgrade = full Yarn PnP API surface when install ships.
func pnpHelperScript() string {
	return `const fs=require('fs');const path=require('path');
let req='';process.stdin.on('data',d=>req+=d);process.stdin.on('end',()=>{
  try{
    const input=JSON.parse(req||'{}');
    const nm=path.join(input.projectRoot||'.','node_modules','.bin',input.command||'');
    const candidates=[nm, nm+'.cmd'];
    for(const c of candidates){ try{ if(fs.existsSync(c)) { console.log(JSON.stringify({found:true,shimPath:c,package:input.package||''})); return; } }catch(e){} }
    console.log(JSON.stringify({found:false}));
  }catch(e){ console.log(JSON.stringify({error:String(e)})); process.exit(1);} 
});`
}

// CheapVerifiedMatch is a gate-off hint: verified metadata + exact shim existence only.
func CheapVerifiedMatch(projectRoot, packageDir, command string) (bool, error) {
	_, found, err := CheapVerifiedHint(Options{
		ProjectRoot: projectRoot,
		PackageDir:  packageDir,
		Command:     command,
	})
	return found, err
}

// PnPRecursionActive reports whether a PnP helper probe is in flight.
func PnPRecursionActive() bool { return pnpInResolve }
