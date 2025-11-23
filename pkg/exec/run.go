package exec

import (
	"bufio"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

// Run executes a command in workDir and returns stdout and error.
// The spawned process will exit upon termination of this application
// to ensure a clean exit
func Run(path string, workDir string, args ...string) error {
	_, err := exec.LookPath(path)
	if err != nil {
		log.Error().Msgf("%s is required, please install it", path)
		os.Exit(1)
	}
	cmd := exec.Command(path, args...)
	cmd.Dir = workDir
	log.Debug().Msg("Executing " + strings.Join(cmd.Args, " "))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal().Msgf("Failed to get stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal().Msgf("Failed to start command: %v", err)
	}

	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		log.Debug().Msg(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal().Msgf("Error reading stdout: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal().Msgf("Command finished with error: %v", err)
		return err
	}
	return nil
}
