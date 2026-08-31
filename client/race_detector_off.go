//go:build !race

package client

// raceDetectorEnabled сообщает тестам, что сборка идёт без -race.
const raceDetectorEnabled = false
