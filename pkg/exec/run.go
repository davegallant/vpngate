package exec

import (
	"bufio"
	"io"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// Run executes a command in workDir and logs its output.
// If the command fails to start or setup fails, an error is logged and returned.
// If the command exits with a non-zero status, the error is returned without logging
// (this allows the caller to decide how to handle it).
func Run(path string, workDir string, args ...string) error {
	_, err := exec.LookPath(path)
	if err != nil {
		log.Error().Msgf("%s is required, please install it", path)
		return err
	}

	cmd := exec.Command(path, args...)
	cmd.Dir = workDir

	log.Debug().Strs("command", cmd.Args).Msg("Executing command")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Msgf("Failed to get stdout pipe: %v", err)
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Error().Msgf("Failed to get stderr pipe: %v", err)
		return err
	}

	if err := cmd.Start(); err != nil {
		log.Error().Msgf("Failed to start command: %v", err)
		return err
	}

	// Combine stdout and stderr into a single reader
	combined := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(combined)
	for scanner.Scan() {
		log.Debug().Msg(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Error().Msgf("Error reading output: %v", err)
		return err
	}

	// cmd.Wait() returns an error if the command exits with non-zero status
	// We return this without logging since it's expected behavior for some commands
	return cmd.Wait()
}