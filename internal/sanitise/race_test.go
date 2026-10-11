//go:build race

package sanitise

// raceEnabled is true when the race detector instruments the build, which
// slows the pass several times over and makes a wall-clock budget meaningless.
const raceEnabled = true
