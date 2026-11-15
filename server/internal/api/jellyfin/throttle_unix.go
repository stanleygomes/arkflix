//go:build unix

package jellyfin

import "syscall"

// pauseProc congela (SIGSTOP) ou solta (SIGCONT) o ffmpeg.
func pauseProc(r *ffRun, pause bool) bool {
	if r == nil || r.proc == nil {
		return false
	}
	sig := syscall.SIGCONT
	if pause {
		sig = syscall.SIGSTOP
	}
	return r.proc.Signal(sig) == nil
}
