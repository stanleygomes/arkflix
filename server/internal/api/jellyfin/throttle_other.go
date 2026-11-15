//go:build !unix

package jellyfin

// pauseProc: fora do Unix não congela (o ffmpeg só corre na frente).
func pauseProc(r *ffRun, pause bool) bool { return false }
